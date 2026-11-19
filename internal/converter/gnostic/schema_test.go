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

package gnostic

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"connectrpc.com/protoc-gen-connect-openapi/internal/converter/options"
	goa3 "github.com/google/gnostic/openapiv3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandExampleEntry(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "object", value: "{id: 1, label: first}", want: "id: 1\nlabel: first\n"},
		{name: "nested object in sequence", value: "[{id: 1}, {id: 2}]", want: "- id: 1\n- id: 2\n"},
		{name: "alias", value: "{a: &x 1, b: *x}", want: "a: 1\nb: 1\n"},
		{name: "non-string key", value: "{1: one}", want: "1: one\n"},
		{name: "multiple non-string keys", value: "{2: two, 1: one}", want: "2: two\n1: one\n"},
		{name: "boolean", value: "true", want: "true\n"},
		{name: "integer", value: "42", want: "42\n"},
		{name: "hex integer", value: "0x1F", want: "31\n"},
		{name: "octal integer", value: "0o17", want: "15\n"},
		{name: "binary integer", value: "0b101", want: "5\n"},
		{name: "underscore integer", value: "1_000", want: "1000\n"},
		{name: "float", value: "1.5", want: "1.5\n"},
		{name: "underscore float", value: "1_000.5", want: "1000.5\n"},
		{name: "string", value: "Example value", want: "Example value\n"},
		{name: "quoted numeric string", value: "'123'", want: "\"123\"\n"},
		{name: "quoted timestamp string", value: "'2024-01-01'", want: "\"2024-01-01\"\n"},
		{name: "quoted boolean string", value: "'true'", want: "\"true\"\n"},
		{name: "timestamp", value: "2024-01-01", want: "\"2024-01-01\"\n"},
		{name: "binary", value: "!!binary aGVsbG8=", want: "hello\n"},
		{name: "large integer", value: "123456789012345678901234567890", want: "\"123456789012345678901234567890\"\n"},
		{name: "nan", value: ".nan", want: "\".nan\"\n"},
		{name: "infinity", value: ".inf", want: "\".inf\"\n"},
		{name: "negative infinity", value: "-.inf", want: "\"-.inf\"\n"},
		{name: "null", value: "null", want: "null\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.value), &node))
			expanded, err := expandExampleEntry(&node)
			require.NoError(t, err)
			out, err := yaml.Marshal(expanded)
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(out))
		})
	}
}

func TestExpandExampleEntryJSONMatchesYAML(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "timestamp", value: "2024-01-01", want: `"2024-01-01"`},
		{name: "binary", value: "!!binary aGVsbG8=", want: `"hello"`},
		{name: "hex integer", value: "0x1F", want: `31`},
		{name: "underscore integer", value: "1_000", want: `1000`},
		{name: "underscore float", value: "1_000.5", want: `1000.5`},
		{name: "boolean", value: "true", want: `true`},
		{name: "large integer", value: "123456789012345678901234567890", want: `"123456789012345678901234567890"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.value), &node))
			expanded, err := expandExampleEntry(&node)
			require.NoError(t, err)

			var decoded any
			require.NoError(t, expanded.Decode(&decoded))
			out, err := json.Marshal(decoded)
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(out))
		})
	}
}

func TestExpandExampleEntryMergesAnchors(t *testing.T) {
	t.Parallel()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("base: &b {a: 1, b: 1}\nchild:\n  <<: *b\n  a: 2\n"), &node))

	expanded, err := expandExampleEntry(&node)
	require.NoError(t, err)
	out, err := yaml.Marshal(expanded)
	require.NoError(t, err)
	assert.YAMLEq(t, "base: {a: 1, b: 1}\nchild: {a: 2, b: 1}\n", string(out))
}

func TestExpandExampleEntryMergesSequenceOfAnchors(t *testing.T) {
	t.Parallel()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("a: &a {x: 1}\nb: &b {x: 2, y: 2}\nchild:\n  <<: [*a, *b]\n"), &node))

	expanded, err := expandExampleEntry(&node)
	require.NoError(t, err)
	out, err := yaml.Marshal(expanded)
	require.NoError(t, err)
	assert.YAMLEq(t, "a: {x: 1}\nb: {x: 2, y: 2}\nchild: {x: 1, y: 2}\n", string(out))
}

func TestExpandExampleEntryRejectsRecursiveMergeKey(t *testing.T) {
	t.Parallel()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("&a {<<: *a}"), &node))

	_, err := expandExampleEntry(&node)
	require.ErrorIs(t, err, errExampleLimit)
}

func TestExpandExampleEntryRejectsInvalidExamples(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "duplicate key", value: "{a: 1, a: 2}"},
		{name: "duplicate key after normalization", value: "{31: a, 0x1F: b}"},
		{name: "duplicate key in nested map", value: "{a: [{b: 1, b: 2}]}"},
		{name: "scalar does not match bool tag", value: "!!bool yes"},
		{name: "scalar does not match int tag", value: "!!int abc"},
		{name: "invalid base64", value: "!!binary 'not base64!'"},
		{name: "map as key", value: "{{a: 1}: b}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.value), &node))

			_, err := expandExampleEntry(&node)
			require.Error(t, err)
		})
	}
}

func TestExpandExampleEntryRejectsNonMapMergeValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "scalar", value: "{<<: 1, a: 2}"},
		{name: "sequence of scalars", value: "{<<: [1, 2], a: 2}"},
		{name: "alias to scalar", value: "{a: &x 1, b: {<<: *x}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.value), &node))

			_, err := expandExampleEntry(&node)
			require.ErrorIs(t, err, errExampleMerge)
		})
	}
}

func TestExpandExampleEntryRejectsRecursiveAlias(t *testing.T) {
	t.Parallel()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("&a [*a]"), &node))

	_, err := expandExampleEntry(&node)
	require.ErrorIs(t, err, errExampleLimit)
}

func TestExpandExampleEntryRejectsExcessiveAliasing(t *testing.T) {
	t.Parallel()
	const payload = `
a: &a ["lol","lol","lol","lol","lol","lol","lol","lol","lol"]
b: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a]
c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b]
d: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c]
e: &e [*d,*d,*d,*d,*d,*d,*d,*d,*d]
f: &f [*e,*e,*e,*e,*e,*e,*e,*e,*e]
g: &g [*f,*f,*f,*f,*f,*f,*f,*f,*f]
h: &h [*g,*g,*g,*g,*g,*g,*g,*g,*g]
i: &i [*h,*h,*h,*h,*h,*h,*h,*h,*h]
`
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(payload), &node))

	_, err := expandExampleEntry(&node)
	require.ErrorIs(t, err, errExampleLimit)
}

func TestExpandExampleEntryAllowsRepeatedAlias(t *testing.T) {
	t.Parallel()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("{a: &x 1, b: *x, c: *x}"), &node))

	expanded, err := expandExampleEntry(&node)
	require.NoError(t, err)
	out, err := yaml.Marshal(expanded)
	require.NoError(t, err)
	assert.Equal(t, "a: 1\nb: 1\nc: 1\n", string(out))
}

func TestSchemaWithAnnotationsSkipsRecursiveExample(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	opts := options.NewOptions()
	opts.Logger = slog.New(slog.NewTextHandler(&logs, nil))

	schema := schemaWithAnnotations(opts, &base.Schema{}, &goa3.Schema{
		Example: &goa3.Any{Yaml: "&a [*a]"},
	})

	assert.Empty(t, schema.Examples)
	assert.Contains(t, logs.String(), "unable to expand example")
}

func TestSchemaWithAnnotationsSkipsInvalidExample(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "duplicate key", value: "{a: 1, a: 2}"},
		{name: "scalar does not match tag", value: "!!bool yes"},
		{name: "non-map merge value", value: "{<<: 1, a: 2}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			opts := options.NewOptions()
			opts.Logger = slog.New(slog.NewTextHandler(&logs, nil))

			schema := schemaWithAnnotations(opts, &base.Schema{}, &goa3.Schema{
				Example: &goa3.Any{Yaml: tc.value},
			})

			assert.Empty(t, schema.Examples)
			assert.Contains(t, logs.String(), "unable to expand example")
		})
	}
}

func TestSchemaWithAnnotationsExampleTypes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "object", value: "{id: 1, label: first}", want: "id: 1\nlabel: first\n"},
		{name: "boolean", value: "true", want: "true\n"},
		{name: "integer", value: "42", want: "42\n"},
		{name: "sequence", value: "[20, 40, 60]", want: "- 20\n- 40\n- 60\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := schemaWithAnnotations(options.Options{}, &base.Schema{}, &goa3.Schema{
				Example: &goa3.Any{Yaml: tc.value},
			})
			require.Len(t, schema.Examples, 1)
			out, err := yaml.Marshal(schema.Examples[0])
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(out))
		})
	}
}

func TestSchemaWithAnnotationsTypeOverrideKeepsDependentKeywords(t *testing.T) {
	t.Parallel()
	schema := schemaWithAnnotations(options.NewOptions(), &base.Schema{Type: []string{"string"}}, &goa3.Schema{
		Type:        "array",
		MaxItems:    3,
		MinItems:    1,
		UniqueItems: true,
	})

	assert.Equal(t, []string{"array"}, schema.Type)
	require.NotNil(t, schema.MaxItems)
	assert.Equal(t, int64(3), *schema.MaxItems)
	require.NotNil(t, schema.MinItems)
	assert.Equal(t, int64(1), *schema.MinItems)
	require.NotNil(t, schema.UniqueItems)
	assert.True(t, *schema.UniqueItems)

	schema = schemaWithAnnotations(options.NewOptions(), &base.Schema{Type: []string{"string"}}, &goa3.Schema{
		Type:          "object",
		MaxProperties: 2,
	})
	require.NotNil(t, schema.MaxProperties)
	assert.Equal(t, int64(2), *schema.MaxProperties)
}
