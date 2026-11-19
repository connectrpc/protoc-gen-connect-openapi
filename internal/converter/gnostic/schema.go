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
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/protoc-gen-connect-openapi/internal/converter/options"
	"connectrpc.com/protoc-gen-connect-openapi/internal/converter/util"
	goa3 "github.com/google/gnostic/openapiv3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func SchemaWithSchemaAnnotations(opts options.Options, schema *base.Schema, desc protoreflect.MessageDescriptor) *base.Schema {
	if !proto.HasExtension(desc.Options(), goa3.E_Schema.TypeDescriptor().Type()) {
		return schema
	}

	ext := proto.GetExtension(desc.Options(), goa3.E_Schema.TypeDescriptor().Type())
	gnosticSchema, ok := ext.(*goa3.Schema)
	if !ok {
		return schema
	}
	return schemaWithAnnotations(opts, schema, gnosticSchema)
}

func SchemaWithPropertyAnnotations(opts options.Options, schema *base.Schema, desc protoreflect.FieldDescriptor) *base.Schema {
	if !proto.HasExtension(desc.Options(), goa3.E_Property.TypeDescriptor().Type()) {
		return schema
	}

	ext := proto.GetExtension(desc.Options(), goa3.E_Property.TypeDescriptor().Type())
	gnosticSchema, ok := ext.(*goa3.Schema)
	if !ok {
		return schema
	}
	// Preserve the numeric base for explicit overrides and constraints.
	if (desc.Kind() == protoreflect.FloatKind || desc.Kind() == protoreflect.DoubleKind) && len(schema.OneOf) > 0 {
		schema.Type = []string{"number"}
		schema.Format = desc.Kind().String()
		schema.OneOf = nil
	}
	return schemaWithAnnotations(opts, schema, gnosticSchema)
}

//gocyclo:ignore
func schemaWithAnnotations(opts options.Options, schema *base.Schema, gnosticSchema *goa3.Schema) *base.Schema {
	if gnosticSchema.Description != "" {
		schema.Description = gnosticSchema.Description
	}
	if gnosticSchema.Title != "" {
		schema.Title = gnosticSchema.Title
	}
	if gnosticSchema.Format != "" {
		schema.Format = gnosticSchema.Format
	}
	if gnosticSchema.Nullable {
		if !slices.Contains(schema.Type, "null") {
			schema.Type = append(schema.Type, "null")
		}
	}
	if gnosticSchema.ReadOnly {
		schema.ReadOnly = &gnosticSchema.ReadOnly
	}
	if gnosticSchema.WriteOnly {
		schema.WriteOnly = &gnosticSchema.WriteOnly
	}
	if gnosticSchema.Example != nil {
		// Decode as-is so scalars keep their YAML type.
		if yamlText := gnosticSchema.Example.GetYaml(); yamlText != "" {
			var node yaml.Node
			if err := yaml.Unmarshal([]byte(yamlText), &node); err != nil {
				opts.Logger.Warn("unable to unmarshal example", slog.Any("error", err))
			} else if expanded, err := expandExampleEntry(&node); err != nil {
				opts.Logger.Warn("unable to expand example", slog.Any("error", err))
			} else {
				schema.Examples = append(schema.Examples, expanded)
			}
		}
		// If the example is defined with google.protobuf.Any
		if gnosticSchema.Example.Value != nil {
			opts.Logger.Warn("unable to unmarshal pb.any example")
		}
	}
	if gnosticSchema.ExternalDocs != nil {
		schema.ExternalDocs = toExternalDocs(gnosticSchema.ExternalDocs)
	}
	if gnosticSchema.Deprecated {
		schema.Deprecated = &gnosticSchema.Deprecated
	}
	if gnosticSchema.MultipleOf != 0 {
		schema.MultipleOf = &gnosticSchema.MultipleOf
	}
	if gnosticSchema.Maximum != 0 {
		if gnosticSchema.ExclusiveMaximum {
			schema.ExclusiveMaximum = &base.DynamicValue[bool, float64]{N: 1, B: gnosticSchema.Maximum}
		} else {
			schema.Maximum = &gnosticSchema.Maximum
		}
	}
	if gnosticSchema.Minimum != 0 {
		if gnosticSchema.ExclusiveMinimum {
			schema.ExclusiveMinimum = &base.DynamicValue[bool, float64]{N: 1, B: gnosticSchema.Minimum}
		} else {
			schema.Minimum = &gnosticSchema.Minimum
		}
	}
	if gnosticSchema.MaxLength > 0 {
		schema.MaxLength = &gnosticSchema.MaxLength
	}
	if gnosticSchema.MinLength > 0 {
		v := gnosticSchema.MinLength
		schema.MinLength = &v
	}
	if gnosticSchema.Pattern != "" {
		schema.Pattern = gnosticSchema.Pattern
	}
	if gnosticSchema.MaxItems > 0 {
		if slices.Contains(schema.Type, "array") {
			schema.MaxItems = &gnosticSchema.MaxItems
		}
	}
	if gnosticSchema.MinItems > 0 {
		if slices.Contains(schema.Type, "array") {
			schema.MinItems = &gnosticSchema.MinItems
		}
	}
	if gnosticSchema.UniqueItems {
		if slices.Contains(schema.Type, "array") {
			schema.UniqueItems = &gnosticSchema.UniqueItems
		}
	}
	if gnosticSchema.MaxProperties > 0 {
		if slices.Contains(schema.Type, "object") {
			schema.MaxProperties = &gnosticSchema.MaxProperties
		}
	}
	if gnosticSchema.MinProperties > 0 {
		if slices.Contains(schema.Type, "object") {
			schema.MinProperties = &gnosticSchema.MinProperties
		}
	}
	if len(gnosticSchema.Required) > 0 {
		schema.Required = gnosticSchema.Required
	}
	if len(gnosticSchema.Enum) > 0 {
		enums := make([]*yaml.Node, len(gnosticSchema.Enum))
		for i, enum := range gnosticSchema.Enum {
			enums[i] = util.ConvertNodeV3toV4(enum.ToRawInfo())
		}
		schema.Enum = enums
	}
	if gnosticSchema.Type != "" {
		// Overriding the type keeps nullability, whether it came from the
		// field (proto3 optional) or from nullable above.
		nullable := slices.Contains(schema.Type, "null")
		schema.Type = []string{gnosticSchema.Type}
		if nullable {
			schema.Type = append(schema.Type, "null")
		}
	}

	if gnosticSchema.AdditionalProperties != nil {
		switch v := gnosticSchema.AdditionalProperties.GetOneof().(type) {
		case *goa3.AdditionalPropertiesItem_SchemaOrReference:
			if vv := toSchemaOrReference(opts, v.SchemaOrReference); vv != nil {
				schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{A: vv}
			}
		case *goa3.AdditionalPropertiesItem_Boolean:
			schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{N: 1, B: v.Boolean}
		}
	}

	if len(gnosticSchema.AllOf) > 0 {
		schema.AllOf = toSchemaOrReferences(opts, gnosticSchema.AllOf)
	}
	if len(gnosticSchema.OneOf) > 0 {
		schema.OneOf = toSchemaOrReferences(opts, gnosticSchema.OneOf)
	}
	if len(gnosticSchema.AnyOf) > 0 {
		schema.AnyOf = toSchemaOrReferences(opts, gnosticSchema.AnyOf)
	}
	if gnosticSchema.Not != nil {
		// The schema may already forbid shapes (oneof rules), so add to that
		// rather than replace it.
		util.AddViolations(schema, base.CreateSchemaProxy(toSchema(opts, gnosticSchema.Not)))
	}
	if gnosticSchema.Items != nil {
		items := toSchemaOrReferences(opts, gnosticSchema.Items.SchemaOrReference)
		var itemsSchema *base.SchemaProxy
		if len(items) == 1 {
			itemsSchema = items[0]
		} else {
			itemsSchema = base.CreateSchemaProxy(&base.Schema{OneOf: items})
		}
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{A: itemsSchema}
	}
	if gnosticSchema.Properties != nil {
		schema.Properties = toSchemaOrReferenceMap(opts, gnosticSchema.Properties.GetAdditionalProperties())
	}
	if gnosticSchema.Default != nil {
		schema.Default = toDefault(gnosticSchema.Default)
	}
	if gnosticSchema.AdditionalProperties != nil {
		schema.AdditionalProperties = toAdditionalPropertiesItem(opts, gnosticSchema.AdditionalProperties)
	}
	if gnosticSchema.Xml != nil {
		extensions := *orderedmap.New[string, *yaml.Node]()
		for _, namedAny := range gnosticSchema.Xml.GetSpecificationExtension() {
			extensions.Set(namedAny.Name, util.ConvertNodeV3toV4(namedAny.ToRawInfo()))
		}
		schema.XML = &base.XML{
			Name:       gnosticSchema.Xml.Name,
			Namespace:  gnosticSchema.Xml.Namespace,
			Prefix:     gnosticSchema.Xml.Prefix,
			Attribute:  gnosticSchema.Xml.Attribute,
			Wrapped:    gnosticSchema.Xml.Wrapped,
			Extensions: &extensions,
		}
	}
	if gnosticSchema.Discriminator != nil {
		mapping := orderedmap.New[string, string]()
		for _, prop := range gnosticSchema.Discriminator.GetMapping().GetAdditionalProperties() {
			mapping.Set(prop.Name, prop.Value)
		}
		schema.Discriminator = &base.Discriminator{
			PropertyName: gnosticSchema.Discriminator.GetPropertyName(),
			Mapping:      mapping,
		}
	}
	if gnosticSchema.SpecificationExtension != nil {
		schema.Extensions = toExtensions(gnosticSchema.SpecificationExtension)
	}

	return schema
}

const (
	maxExampleDepth = 100
	maxExampleNodes = 10_000
)

var (
	errExampleLimit = errors.New("example contains a recursive alias or exceeds expansion limits")
	errExampleMerge = errors.New("example merge key requires a map or a sequence of maps")
)

type exampleExpandState struct {
	seenAliases map[*yaml.Node]bool
	remaining   int
}

func expandExampleEntry(node *yaml.Node) (*yaml.Node, error) {
	state := &exampleExpandState{
		seenAliases: make(map[*yaml.Node]bool),
		remaining:   maxExampleNodes,
	}
	expanded, err := expandExampleNode(node, state, 0)
	if err != nil {
		return nil, err
	}
	// Decode the result to reject what the walk doesn't check: duplicate keys
	// and scalars that don't match their tag. Those produce invalid YAML or
	// fail JSON rendering. The expanded tree is bounded and has no aliases,
	// so decoding it is bounded too.
	var decoded any
	if err := expanded.Decode(&decoded); err != nil {
		return nil, err
	}
	return expanded, nil
}

func expandExampleNode(node *yaml.Node, state *exampleExpandState, depth int) (*yaml.Node, error) {
	if depth > maxExampleDepth || state.remaining <= 0 {
		return nil, errExampleLimit
	}
	if node == nil {
		return nullExampleNode(), nil
	}
	state.remaining--

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 1 {
			return expandExampleNode(node.Content[0], state, depth+1)
		}
		return nullExampleNode(), nil
	case yaml.MappingNode:
		return expandExampleMapping(node, state, depth)
	case yaml.SequenceNode:
		out := utils.CreateEmptySequenceNode()
		for _, child := range node.Content {
			expanded, err := expandExampleNode(child, state, depth+1)
			if err != nil {
				return nil, err
			}
			out.Content = append(out.Content, expanded)
		}
		return out, nil
	case yaml.AliasNode:
		if node.Alias == nil || state.seenAliases[node.Alias] {
			return nil, errExampleLimit
		}
		state.seenAliases[node.Alias] = true
		expanded, err := expandExampleNode(node.Alias, state, depth+1)
		delete(state.seenAliases, node.Alias)
		if err != nil {
			return nil, err
		}
		return expanded, nil
	default:
		return normalizeExampleScalar(node), nil
	}
}

func expandExampleMapping(node *yaml.Node, state *exampleExpandState, depth int) (*yaml.Node, error) {
	out := utils.CreateEmptyMapNode()
	explicit := make(map[string]bool)
	var merged []examplePair

	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		if keyNode.Tag == "!!merge" {
			pairs, err := collectMergePairs(valueNode, state, depth+1)
			if err != nil {
				return nil, err
			}
			merged = append(merged, pairs...)
			continue
		}
		key, err := expandExampleNode(keyNode, state, depth+1)
		if err != nil {
			return nil, err
		}
		value, err := expandExampleNode(valueNode, state, depth+1)
		if err != nil {
			return nil, err
		}
		explicit[exampleKeyID(key)] = true
		out.Content = append(out.Content, key, value)
	}

	seenMerged := make(map[string]bool)
	for _, pair := range merged {
		id := exampleKeyID(pair.key)
		if explicit[id] || seenMerged[id] {
			continue
		}
		seenMerged[id] = true
		out.Content = append(out.Content, pair.key, pair.value)
	}
	return out, nil
}

type examplePair struct {
	key   *yaml.Node
	value *yaml.Node
}

func collectMergePairs(node *yaml.Node, state *exampleExpandState, depth int) ([]examplePair, error) {
	if depth > maxExampleDepth || state.remaining <= 0 {
		return nil, errExampleLimit
	}
	if node == nil {
		return nil, nil
	}
	state.remaining--

	switch node.Kind {
	case yaml.AliasNode:
		if node.Alias == nil || state.seenAliases[node.Alias] {
			return nil, errExampleLimit
		}
		state.seenAliases[node.Alias] = true
		pairs, err := collectMergePairs(node.Alias, state, depth+1)
		delete(state.seenAliases, node.Alias)
		return pairs, err
	case yaml.SequenceNode:
		var pairs []examplePair
		for _, child := range node.Content {
			childPairs, err := collectMergePairs(child, state, depth+1)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, childPairs...)
		}
		return pairs, nil
	case yaml.MappingNode:
		expanded, err := expandExampleMapping(node, state, depth+1)
		if err != nil {
			return nil, err
		}
		pairs := make([]examplePair, 0, len(expanded.Content)/2)
		for i := 0; i+1 < len(expanded.Content); i += 2 {
			pairs = append(pairs, examplePair{key: expanded.Content[i], value: expanded.Content[i+1]})
		}
		return pairs, nil
	default:
		return nil, errExampleMerge
	}
}

func exampleKeyID(key *yaml.Node) string {
	return key.Tag + ":" + key.Value
}

func normalizeExampleScalar(node *yaml.Node) *yaml.Node {
	switch node.Tag {
	case "!!null":
		return nullExampleNode()
	case "!!bool":
		if value, err := strconv.ParseBool(strings.ToLower(node.Value)); err == nil {
			return utils.CreateYamlNode(value)
		}
	case "!!int":
		value := strings.ReplaceAll(node.Value, "_", "")
		if parsed, err := strconv.ParseInt(value, 0, 64); err == nil {
			return utils.CreateYamlNode(parsed)
		}
		if parsed, err := strconv.ParseUint(value, 0, 64); err == nil {
			return utils.CreateYamlNode(parsed)
		}
	case "!!float":
		value := strings.ReplaceAll(node.Value, "_", "")
		switch strings.ToLower(value) {
		case ".nan", ".inf", "+.inf", "-.inf":
			return utils.CreateStringNode(node.Value)
		}
		// Integers beyond int64/uint64 are tagged as floats but would lose
		// precision as JSON numbers, so keep their original text.
		if isIntegerLexeme(value) {
			return utils.CreateStringNode(node.Value)
		}
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return utils.CreateYamlNode(parsed)
		}
	case "!!timestamp":
		return utils.CreateStringNode(node.Value)
	case "!!binary":
		var value string
		if err := node.Decode(&value); err == nil {
			return utils.CreateStringNode(value)
		}
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: node.Tag, Value: node.Value}
}

func isIntegerLexeme(value string) bool {
	value = strings.TrimPrefix(strings.TrimPrefix(value, "+"), "-")
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func nullExampleNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
}
