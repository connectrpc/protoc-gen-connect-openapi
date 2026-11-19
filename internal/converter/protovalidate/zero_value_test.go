// Copyright 2023-2026 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protovalidate

import (
	"testing"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"connectrpc.com/protoc-gen-connect-openapi/internal/converter/options"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Fields of well-known messages stand in for the kinds of field under test.
var (
	fieldDescriptorFields = (&descriptorpb.FieldDescriptorProto{}).ProtoReflect().Descriptor().Fields()
	stringField           = fieldDescriptorFields.ByName("name")
	int32Field            = fieldDescriptorFields.ByName("number")
	boolField             = fieldDescriptorFields.ByName("proto3_optional")
	messageField          = fieldDescriptorFields.ByName("options")
	int64Field            = (&durationpb.Duration{}).ProtoReflect().Descriptor().Fields().ByName("seconds")
	floatField            = (&wrapperspb.FloatValue{}).ProtoReflect().Descriptor().Fields().ByName("value")
	bytesField            = (&wrapperspb.BytesValue{}).ProtoReflect().Descriptor().Fields().ByName("value")
	listField             = (&descriptorpb.FileDescriptorProto{}).ProtoReflect().Descriptor().Fields().ByName("dependency")
	mapField              = (&structpb.Struct{}).ProtoReflect().Descriptor().Fields().ByName("fields")
	// FeatureSet.FieldPresence has FIELD_PRESENCE_UNKNOWN = 0, EXPLICIT = 1
	// and IMPLICIT = 2.
	enumField = (&descriptorpb.FeatureSet{}).ProtoReflect().Descriptor().Fields().ByName("field_presence")
)

func marshalSchema(t *testing.T, schema *base.Schema) string {
	t.Helper()
	// A schema that wasn't parsed from a document only renders through a proxy.
	out, err := yaml.Marshal(base.CreateSchemaProxy(schema))
	require.NoError(t, err)
	return string(out)
}

func TestIgnoreIfZeroValue(t *testing.T) {
	t.Parallel()
	ignore := validate.Ignore_IGNORE_IF_ZERO_VALUE.Enum()
	for _, tc := range []struct {
		name       string
		schema     *base.Schema
		rules      *validate.FieldRules
		desc       protoreflect.FieldDescriptor
		onlyScalar bool
		want       string
	}{
		{
			name:   "string rules move next to the empty string",
			schema: &base.Schema{Type: []string{"string"}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_String_{String_: &validate.StringRules{
				Pattern:   proto.String("^[a-z]+$"),
				MinLen:    proto.Uint64(3),
				MaxLen:    proto.Uint64(10),
				WellKnown: &validate.StringRules_Email{Email: true},
			}}},
			desc: stringField,
			want: "type: string\nanyOf:\n    - minLength: 3\n      pattern: ^[a-z]+$\n      format: email\n    - const: \"\"\nmaxLength: 10\n",
		},
		{
			name:   "rules the zero value satisfies stay in place",
			schema: &base.Schema{Type: []string{"string"}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_String_{String_: &validate.StringRules{
				MaxLen: proto.Uint64(10),
			}}},
			desc: stringField,
			want: "type: string\nmaxLength: 10\n",
		},
		{
			name:   "without the ignore option nothing moves",
			schema: &base.Schema{Type: []string{"string"}},
			rules: &validate.FieldRules{Type: &validate.FieldRules_String_{String_: &validate.StringRules{
				Pattern: proto.String("^[a-z]+$"),
			}}},
			desc: stringField,
			want: "type: string\npattern: ^[a-z]+$\n",
		},
		{
			name:   "bytes",
			schema: &base.Schema{Type: []string{"string"}, Format: "byte"},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Bytes{Bytes: &validate.BytesRules{
				Const: []byte{1, 2, 3, 4},
			}}},
			desc: bytesField,
			want: "type: string\nanyOf:\n    - const: AQIDBA==\n    - const: \"\"\nformat: byte\n",
		},
		{
			name:   "32-bit integer keeps its own format",
			schema: &base.Schema{Type: []string{"integer"}, Format: "int32"},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Int32{Int32: &validate.Int32Rules{
				GreaterThan: &validate.Int32Rules_Gte{Gte: 18},
			}}},
			desc: int32Field,
			want: "type: integer\nanyOf:\n    - minimum: 18\n    - const: 0\nformat: int32\n",
		},
		{
			name:   "64-bit integer is a string",
			schema: &base.Schema{Type: []string{"string"}, Format: "int64"},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Int64{Int64: &validate.Int64Rules{
				In: []int64{5, 6},
			}}},
			desc: int64Field,
			want: "type: string\nanyOf:\n    - enum:\n        - \"5\"\n        - \"6\"\n    - const: \"0\"\nformat: int64\n",
		},
		{
			name: "float keeps the format the rules give the number",
			schema: &base.Schema{OneOf: []*base.SchemaProxy{
				base.CreateSchemaProxy(&base.Schema{Type: []string{"number"}, Format: "float"}),
				base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}),
			}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Float{Float: &validate.FloatRules{
				GreaterThan: &validate.FloatRules_Gt{Gt: 1},
			}}},
			desc: floatField,
			want: "type: number\nanyOf:\n    - exclusiveMinimum: 1\n    - const: 0\nformat: float\n",
		},
		{
			name:   "bool",
			schema: &base.Schema{Type: []string{"boolean"}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Bool{Bool: &validate.BoolRules{
				Const: proto.Bool(true),
			}}},
			desc: boolField,
			want: "type: boolean\nanyOf:\n    - const: true\n    - const: false\n",
		},
		{
			name:   "list",
			schema: &base.Schema{Type: []string{"array"}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Repeated{Repeated: &validate.RepeatedRules{
				MinItems: proto.Uint64(2),
				MaxItems: proto.Uint64(5),
			}}},
			desc: listField,
			want: "type: array\nanyOf:\n    - minItems: 2\n    - maxItems: 0\nmaxItems: 5\n",
		},
		{
			name:   "map",
			schema: &base.Schema{Type: []string{"object"}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Map{Map: &validate.MapRules{
				MinPairs: proto.Uint64(2),
			}}},
			desc: mapField,
			want: "type: object\nanyOf:\n    - minProperties: 2\n    - maxProperties: 0\n",
		},
		{
			name:   "list item uses the zero value of the item",
			schema: &base.Schema{Type: []string{"string"}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_String_{String_: &validate.StringRules{
				MinLen: proto.Uint64(3),
			}}},
			desc:       listField,
			onlyScalar: true,
			want:       "type: string\nanyOf:\n    - minLength: 3\n    - const: \"\"\n",
		},
		{
			name:   "enum list gains the zero value",
			schema: &base.Schema{},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Enum{Enum: &validate.EnumRules{
				In: []int32{2},
			}}},
			desc: enumField,
			want: "enum:\n    - IMPLICIT\n    - FIELD_PRESENCE_UNKNOWN\n",
		},
		{
			name:   "enum const becomes a list with the zero value",
			schema: &base.Schema{},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Enum{Enum: &validate.EnumRules{
				Const: proto.Int32(1),
			}}},
			desc: enumField,
			want: "enum:\n    - EXPLICIT\n    - FIELD_PRESENCE_UNKNOWN\n",
		},
		{
			name:   "enum list that has the zero value is unchanged",
			schema: &base.Schema{},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_Enum{Enum: &validate.EnumRules{
				In: []int32{0, 1},
			}}},
			desc: enumField,
			want: "enum:\n    - FIELD_PRESENCE_UNKNOWN\n    - EXPLICIT\n",
		},
		{
			name:   "message has no zero value",
			schema: &base.Schema{Type: []string{"string"}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_String_{String_: &validate.StringRules{
				MinLen: proto.Uint64(3),
			}}},
			desc: messageField,
			want: "type: string\nminLength: 3\n",
		},
		{
			name:   "unknown field is left alone",
			schema: &base.Schema{Type: []string{"string"}},
			rules: &validate.FieldRules{Ignore: ignore, Type: &validate.FieldRules_String_{String_: &validate.StringRules{
				MinLen: proto.Uint64(3),
			}}},
			want: "type: string\nminLength: 3\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			updateSchemaWithFieldRules(options.NewOptions(), tc.schema, tc.rules, tc.onlyScalar, tc.desc)
			assert.Equal(t, tc.want, marshalSchema(t, tc.schema))
		})
	}
}

// A map field's schema is annotated twice, and the second pass must not nest
// or duplicate the anyOf from the first.
func TestIgnoreIfZeroValueAppliedTwice(t *testing.T) {
	t.Parallel()
	schema := &base.Schema{Type: []string{"object"}}
	for range 2 {
		rules := &validate.FieldRules{
			Ignore: validate.Ignore_IGNORE_IF_ZERO_VALUE.Enum(),
			Type:   &validate.FieldRules_Map{Map: &validate.MapRules{MinPairs: proto.Uint64(2)}},
		}
		updateSchemaWithFieldRules(options.NewOptions(), schema, rules, false, mapField)
	}
	assert.Equal(t, "type: object\nanyOf:\n    - minProperties: 2\n    - maxProperties: 0\n", marshalSchema(t, schema))
}

func TestIgnoreAlways(t *testing.T) {
	t.Parallel()
	schema := &base.Schema{Type: []string{"array"}}
	rules := &validate.FieldRules{Type: &validate.FieldRules_Repeated{Repeated: &validate.RepeatedRules{
		MinItems: proto.Uint64(2),
		Items: &validate.FieldRules{
			Ignore: validate.Ignore_IGNORE_ALWAYS.Enum(),
			Type:   &validate.FieldRules_String_{String_: &validate.StringRules{MinLen: proto.Uint64(3)}},
		},
	}}}
	schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
		A: base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}}),
	}
	updateSchemaWithFieldRules(options.NewOptions(), schema, rules, false, listField)
	assert.Equal(t, "type: array\nitems:\n    type: string\nminItems: 2\n", marshalSchema(t, schema))

	rules.Ignore = validate.Ignore_IGNORE_ALWAYS.Enum()
	schema = &base.Schema{Type: []string{"array"}}
	updateSchemaWithFieldRules(options.NewOptions(), schema, rules, false, listField)
	assert.Equal(t, "type: array\n", marshalSchema(t, schema))
}
