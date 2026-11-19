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
	"testing"

	"connectrpc.com/protoc-gen-connect-openapi/internal/converter/options"
	goa3 "github.com/google/gnostic/openapiv3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestPathItemWithMethodAnnotations_ParameterMerging(t *testing.T) {
	t.Run("merges parameters from gnostic annotations", func(t *testing.T) {
		// Create existing path item with operation
		item := &v3.PathItem{
			Post: &v3.Operation{
				Parameters: []*v3.Parameter{
					{
						Name:        "id",
						In:          "path",
						Description: "Original ID",
					},
					{
						Name: "filter",
						In:   "query",
					},
				},
			},
		}

		// Create method descriptor with gnostic operation annotation
		gnosticOp := &goa3.Operation{
			Parameters: []*goa3.ParameterOrReference{
				{
					Oneof: &goa3.ParameterOrReference_Parameter{
						Parameter: &goa3.Parameter{
							Name:        "id",
							In:          "path",
							Description: "ID from annotation",
							Required:    true,
						},
					},
				},
				{
					Oneof: &goa3.ParameterOrReference_Parameter{
						Parameter: &goa3.Parameter{
							Name: "limit",
							In:   "query",
						},
					},
				},
			},
		}

		methodOpts := &descriptorpb.MethodOptions{}
		proto.SetExtension(methodOpts, goa3.E_Operation, gnosticOp)

		md := &mockMethodDescriptor{
			options: methodOpts,
		}

		opts := options.Options{}
		result := PathItemWithMethodAnnotations(opts, item, md)

		// Verify parameters were merged
		require.NotNil(t, result.Post)
		assert.Len(t, result.Post.Parameters, 3, "should have 3 parameters after merge")

		// Check id parameter - should preserve original description
		idParam := result.Post.Parameters[0]
		assert.Equal(t, "id", idParam.Name)
		assert.Equal(t, "path", idParam.In)
		assert.Equal(t, "Original ID", idParam.Description, "should preserve original description")
		assert.True(t, *idParam.Required, "should merge required flag")

		// Check filter parameter - should be unchanged
		filterParam := result.Post.Parameters[1]
		assert.Equal(t, "filter", filterParam.Name)
		assert.Equal(t, "query", filterParam.In)

		// Check limit parameter - should be added
		limitParam := result.Post.Parameters[2]
		assert.Equal(t, "limit", limitParam.Name)
		assert.Equal(t, "query", limitParam.In)
	})

	t.Run("appends new parameters from annotations", func(t *testing.T) {
		item := &v3.PathItem{
			Get: &v3.Operation{
				Parameters: []*v3.Parameter{},
			},
		}

		gnosticOp := &goa3.Operation{
			Parameters: []*goa3.ParameterOrReference{
				{
					Oneof: &goa3.ParameterOrReference_Parameter{
						Parameter: &goa3.Parameter{
							Name:        "api_key",
							In:          "header",
							Description: "API Key",
							Required:    true,
						},
					},
				},
			},
		}

		methodOpts := &descriptorpb.MethodOptions{}
		proto.SetExtension(methodOpts, goa3.E_Operation, gnosticOp)

		md := &mockMethodDescriptor{
			options: methodOpts,
		}

		opts := options.Options{}
		result := PathItemWithMethodAnnotations(opts, item, md)

		require.NotNil(t, result.Get)
		assert.Len(t, result.Get.Parameters, 1)
		assert.Equal(t, "api_key", result.Get.Parameters[0].Name)
	})

	t.Run("returns unchanged item when no gnostic operation extension", func(t *testing.T) {
		item := &v3.PathItem{
			Post: &v3.Operation{
				Summary: "Original operation",
				Parameters: []*v3.Parameter{
					{Name: "id", In: "path"},
				},
			},
		}

		md := &mockMethodDescriptor{
			options: &descriptorpb.MethodOptions{},
		}

		opts := options.Options{}
		result := PathItemWithMethodAnnotations(opts, item, md)

		assert.Equal(t, item, result)
		assert.Len(t, result.Post.Parameters, 1)
		assert.Equal(t, "id", result.Post.Parameters[0].Name)
	})

	t.Run("merges parameters across multiple operations in path item", func(t *testing.T) {
		item := &v3.PathItem{
			Get: &v3.Operation{
				Parameters: []*v3.Parameter{
					{Name: "id", In: "path"},
				},
			},
			Post: &v3.Operation{
				Parameters: []*v3.Parameter{
					{Name: "id", In: "path"},
				},
			},
		}

		gnosticOp := &goa3.Operation{
			Parameters: []*goa3.ParameterOrReference{
				{
					Oneof: &goa3.ParameterOrReference_Parameter{
						Parameter: &goa3.Parameter{
							Name: "filter",
							In:   "query",
						},
					},
				},
			},
		}

		methodOpts := &descriptorpb.MethodOptions{}
		proto.SetExtension(methodOpts, goa3.E_Operation, gnosticOp)

		md := &mockMethodDescriptor{
			options: methodOpts,
		}

		opts := options.Options{}
		result := PathItemWithMethodAnnotations(opts, item, md)

		// Both GET and POST should have the filter parameter added
		require.NotNil(t, result.Get)
		assert.Len(t, result.Get.Parameters, 2)
		assert.Equal(t, "filter", result.Get.Parameters[1].Name)

		require.NotNil(t, result.Post)
		assert.Len(t, result.Post.Parameters, 2)
		assert.Equal(t, "filter", result.Post.Parameters[1].Name)
	})

	t.Run("preserves parameter schema details when merging", func(t *testing.T) {
		item := &v3.PathItem{
			Get: &v3.Operation{
				Parameters: []*v3.Parameter{
					{
						Name:        "page",
						In:          "query",
						Description: "Original page description",
						Schema: base.CreateSchemaProxy(&base.Schema{
							Type:    []string{"integer"},
							Default: &yaml.Node{Value: "1"},
						}),
					},
				},
			},
		}

		gnosticOp := &goa3.Operation{
			Parameters: []*goa3.ParameterOrReference{
				{
					Oneof: &goa3.ParameterOrReference_Parameter{
						Parameter: &goa3.Parameter{
							Name: "page",
							In:   "query",
							Schema: &goa3.SchemaOrReference{
								Oneof: &goa3.SchemaOrReference_Schema{
									Schema: &goa3.Schema{
										Type:   "integer",
										Format: "int32",
									},
								},
							},
						},
					},
				},
			},
		}

		methodOpts := &descriptorpb.MethodOptions{}
		proto.SetExtension(methodOpts, goa3.E_Operation, gnosticOp)

		md := &mockMethodDescriptor{
			options: methodOpts,
		}

		opts := options.Options{}
		result := PathItemWithMethodAnnotations(opts, item, md)

		require.NotNil(t, result.Get)
		assert.Len(t, result.Get.Parameters, 1)

		pageParam := result.Get.Parameters[0]
		assert.Equal(t, "page", pageParam.Name)
		assert.Equal(t, "Original page description", pageParam.Description, "should preserve original description")
		// The schema should have been merged with format from gnostic annotation
		assert.Equal(t, "int32", pageParam.Schema.Schema().Format)
	})
}

func TestMergeRequestBody(t *testing.T) {
	t.Parallel()
	generatedSchema := base.CreateSchemaProxyRef("#/components/schemas/Generated")
	annotatedSchema := base.CreateSchemaProxyRef("#/components/schemas/Annotated")
	newGenerated := func() *v3.RequestBody {
		content := orderedmap.New[string, *v3.MediaType]()
		content.Set("application/json", &v3.MediaType{Schema: generatedSchema})
		content.Set("application/proto", &v3.MediaType{Schema: generatedSchema})
		required := true
		return &v3.RequestBody{Description: "Generated", Content: content, Required: &required}
	}
	newAnnotated := func(mediaTypes map[string]*v3.MediaType) *v3.RequestBody {
		content := orderedmap.New[string, *v3.MediaType]()
		for name, mediaType := range mediaTypes {
			content.Set(name, mediaType)
		}
		return &v3.RequestBody{Content: content}
	}

	t.Run("media type without a schema keeps the generated schema", func(t *testing.T) {
		t.Parallel()
		example := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "example"}
		merged := mergeRequestBody(newGenerated(), newAnnotated(map[string]*v3.MediaType{
			"application/json": {Example: example},
		}))

		// The annotation still decides everything else about the body.
		require.Equal(t, 1, merged.Content.Len())
		assert.Empty(t, merged.Description)
		assert.Nil(t, merged.Required)
		mediaType, ok := merged.Content.Get("application/json")
		require.True(t, ok)
		assert.Same(t, generatedSchema, mediaType.Schema)
		assert.Same(t, example, mediaType.Example)
	})

	t.Run("media type with a schema keeps its own", func(t *testing.T) {
		t.Parallel()
		merged := mergeRequestBody(newGenerated(), newAnnotated(map[string]*v3.MediaType{
			"application/json": {Schema: annotatedSchema},
		}))

		mediaType, ok := merged.Content.Get("application/json")
		require.True(t, ok)
		assert.Same(t, annotatedSchema, mediaType.Schema)
	})

	t.Run("media type that wasn't generated has no schema to keep", func(t *testing.T) {
		t.Parallel()
		merged := mergeRequestBody(newGenerated(), newAnnotated(map[string]*v3.MediaType{
			"text/csv": {},
		}))

		require.Equal(t, 1, merged.Content.Len())
		mediaType, ok := merged.Content.Get("text/csv")
		require.True(t, ok)
		assert.Nil(t, mediaType.Schema)
	})

	t.Run("body without content keeps the generated content", func(t *testing.T) {
		t.Parallel()
		generated := newGenerated()
		merged := mergeRequestBody(generated, &v3.RequestBody{Description: "Annotated"})

		assert.Equal(t, "Annotated", merged.Description)
		assert.Same(t, generated.Content, merged.Content)
		require.NotNil(t, merged.Required)
		assert.True(t, *merged.Required)
		// The generated body is left alone.
		assert.Equal(t, "Generated", generated.Description)
	})

	t.Run("body without content can make the generated body required", func(t *testing.T) {
		t.Parallel()
		generated := newGenerated()
		generated.Required = nil
		required := true
		merged := mergeRequestBody(generated, &v3.RequestBody{Required: &required})

		assert.Equal(t, "Generated", merged.Description)
		assert.Same(t, generated.Content, merged.Content)
		require.NotNil(t, merged.Required)
		assert.True(t, *merged.Required)
	})

	t.Run("referenced body replaces the generated body", func(t *testing.T) {
		t.Parallel()
		annotated := v3.CreateRequestBodyRef("#/components/requestBodies/Annotated")
		assert.Same(t, annotated, mergeRequestBody(newGenerated(), annotated))
	})

	t.Run("without a generated body the annotation is used as is", func(t *testing.T) {
		t.Parallel()
		annotated := &v3.RequestBody{Description: "Annotated"}
		assert.Same(t, annotated, mergeRequestBody(nil, annotated))
	})

	t.Run("empty annotation removes the generated body", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, mergeRequestBody(newGenerated(), nil))
	})
}

// Mock implementation of protoreflect.MethodDescriptor for testing
type mockMethodDescriptor struct {
	protoreflect.MethodDescriptor
	options *descriptorpb.MethodOptions
}

func (m *mockMethodDescriptor) Options() protoreflect.ProtoMessage {
	return m.options
}
