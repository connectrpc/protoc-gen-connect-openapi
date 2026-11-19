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
	"slices"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/utils"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// zeroValueKeywords copies the keywords that can reject a field's zero value.
// Maximum lengths and counts are left out because zero always satisfies them.
func zeroValueKeywords(schema *base.Schema) *base.Schema {
	return &base.Schema{
		Pattern:          schema.Pattern,
		Format:           schema.Format,
		MinLength:        schema.MinLength,
		Enum:             schema.Enum,
		Const:            schema.Const,
		Not:              schema.Not,
		Minimum:          schema.Minimum,
		Maximum:          schema.Maximum,
		ExclusiveMinimum: schema.ExclusiveMinimum,
		ExclusiveMaximum: schema.ExclusiveMaximum,
		MinItems:         schema.MinItems,
		MinProperties:    schema.MinProperties,
	}
}

// allowZeroValue makes schema accept the zero value of a field that has
// IGNORE_IF_ZERO_VALUE, since its rules don't run for that value. before holds
// the keywords from before the rules were applied. The keywords the rules
// changed move into an anyOf, next to a schema that only matches the zero
// value.
func allowZeroValue(schema, before *base.Schema, desc protoreflect.FieldDescriptor, onlyScalar bool) {
	if desc == nil {
		return
	}
	isContainer := !onlyScalar && (desc.IsList() || desc.IsMap())
	if !isContainer && desc.Kind() == protoreflect.EnumKind {
		allowZeroEnumValue(schema, before, desc.Enum())
		return
	}
	zero := zeroValueSchema(desc, isContainer)
	if zero == nil {
		return
	}

	rules := &base.Schema{}
	moved := false
	moveChanged(&moved, &schema.Pattern, &rules.Pattern, before.Pattern)
	moveChanged(&moved, &schema.MinLength, &rules.MinLength, before.MinLength)
	moveChanged(&moved, &schema.Const, &rules.Const, before.Const)
	moveChanged(&moved, &schema.Not, &rules.Not, before.Not)
	moveChanged(&moved, &schema.Minimum, &rules.Minimum, before.Minimum)
	moveChanged(&moved, &schema.Maximum, &rules.Maximum, before.Maximum)
	moveChanged(&moved, &schema.ExclusiveMinimum, &rules.ExclusiveMinimum, before.ExclusiveMinimum)
	moveChanged(&moved, &schema.ExclusiveMaximum, &rules.ExclusiveMaximum, before.ExclusiveMaximum)
	moveChanged(&moved, &schema.MinItems, &rules.MinItems, before.MinItems)
	moveChanged(&moved, &schema.MinProperties, &rules.MinProperties, before.MinProperties)
	// Only string rules set a format that validates the value. The float
	// rules also change it, but only to name the number's size.
	if desc.Kind() == protoreflect.StringKind {
		moveChanged(&moved, &schema.Format, &rules.Format, before.Format)
	}
	if !slices.Equal(schema.Enum, before.Enum) {
		rules.Enum, schema.Enum = schema.Enum, before.Enum
		moved = true
	}
	if !moved {
		return
	}
	// A map field is annotated twice, so this replaces the anyOf from the
	// first pass instead of nesting it.
	schema.AnyOf = []*base.SchemaProxy{
		base.CreateSchemaProxy(rules),
		base.CreateSchemaProxy(zero),
	}
}

// moveChanged moves a keyword from the field's schema to the rules schema if
// the rules changed it, and puts back the value it had before.
func moveChanged[T comparable](moved *bool, current, rules *T, before T) {
	if *current == before {
		return
	}
	*rules, *current = *current, before
	*moved = true
}

// zeroValueSchema returns a schema that only matches the zero value of a
// field, or nil if the field has none in JSON, as for a message.
func zeroValueSchema(desc protoreflect.FieldDescriptor, isContainer bool) *base.Schema {
	if isContainer {
		var empty int64
		if desc.IsMap() {
			return &base.Schema{MaxProperties: &empty}
		}
		return &base.Schema{MaxItems: &empty}
	}
	switch desc.Kind() {
	case protoreflect.StringKind, protoreflect.BytesKind:
		return &base.Schema{Const: utils.CreateStringNode("")}
	case protoreflect.BoolKind:
		return &base.Schema{Const: utils.CreateBoolNode("false")}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		// 64-bit integers are JSON strings.
		return &base.Schema{Const: utils.CreateStringNode("0")}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.FloatKind, protoreflect.DoubleKind:
		return &base.Schema{Const: utils.CreateIntNode("0")}
	default:
		return nil
	}
}

// allowZeroEnumValue adds the enum's zero value to the values the rules
// allow. An enum field keeps a $ref next to its keywords, and an anyOf would
// replace that $ref, so the value is added to the list instead.
func allowZeroEnumValue(schema, before *base.Schema, enumDesc protoreflect.EnumDescriptor) {
	if enumDesc == nil {
		return
	}
	zeroValue := enumDesc.Values().ByNumber(0)
	if zeroValue == nil {
		return
	}
	zeroName := string(zeroValue.Name())
	if schema.Const != before.Const {
		schema.Enum = []*yaml.Node{schema.Const}
		schema.Const = before.Const
	}
	if slices.Equal(schema.Enum, before.Enum) {
		return
	}
	if !slices.ContainsFunc(schema.Enum, func(node *yaml.Node) bool { return node.Value == zeroName }) {
		schema.Enum = append(schema.Enum, utils.CreateStringNode(zeroName))
	}
}
