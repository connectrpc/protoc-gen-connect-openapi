# Gnostic Support
protoc-gen-connect-openapi has support for the [OpenAPI v3 annotations](https://github.com/google/gnostic/blob/main/openapiv3/annotations.proto) provided by the [google/gnostic project](https://github.com/google/gnostic). Here's an example of what this looks like in a protobuf file:

```protobuf
syntax = "proto3";

package example_with_gnostic;

import "gnostic/openapi/v3/annotations.proto";

option (gnostic.openapi.v3.document) = {
  info: {
    title: "Title from annotation";
    version: "Version from annotation";
    description: "Description from annotation";
    contact: {
      name: "Contact Name";
      url: "https://github.com/connectrpc/protoc-gen-connect-openapi";
      email: "support@example.com";
    }
    license: {
      name: "Apache-2.0";
      url: "https://github.com/connectrpc/protoc-gen-connect-openapi/blob/main/LICENSE";
    }
  }
  components: {
    security_schemes: {
      additional_properties: [
        {
          name: "BasicAuth";
          value: {
            security_scheme: {
              type: "http";
              scheme: "basic";
            }
          }
        }
      ]
    }
  }
};

// The greeting service definition.
service Greeter {
  // Sends a greeting
  rpc SayHello(HelloRequest) returns (HelloReply) {
    option idempotency_level = NO_SIDE_EFFECTS;
    option (gnostic.openapi.v3.operation) = {
      deprecated: true,
      security: [
        {
          additional_properties: [
            {
              name: "BasicAuth";
              value: {
                value: []
              }
            }
          ]
        }
      ]
    };
  }
}

// The request message containing the user's name.
message HelloRequest {
  option (gnostic.openapi.v3.schema) = {title: "Custom title for a message"};

  string name = 1 [(gnostic.openapi.v3.property) = {title: "Custom title for a field"}];
}

// The response message containing the greetings
message HelloReply {
  string message = 1;
}

```

#### File Options
| Option | Supported? | Notes |
|---|---|---|
| (gnostic.openapi.v3.document).openapi | ❌ | The plugin sets the OpenAPI version |
| (gnostic.openapi.v3.document).info | ✅ | |
| (gnostic.openapi.v3.document).servers | ✅ | |
| (gnostic.openapi.v3.document).paths | ✅ | |
| (gnostic.openapi.v3.document).components | ✅ | |
| (gnostic.openapi.v3.document).security | ✅ | |
| (gnostic.openapi.v3.document).tags | ✅ | |
| (gnostic.openapi.v3.document).external_docs | ✅ | |
| (gnostic.openapi.v3.document).specification_extension | ✅ | |

#### Method Options
| Option | Supported? |
|---|---|
| (gnostic.openapi.v3.operation).tags | ✅ |
| (gnostic.openapi.v3.operation).summary | ✅ |
| (gnostic.openapi.v3.operation).description | ✅ |
| (gnostic.openapi.v3.operation).external_docs | ✅ |
| (gnostic.openapi.v3.operation).operation_id | ✅ |
| (gnostic.openapi.v3.operation).parameters | ✅ |
| (gnostic.openapi.v3.operation).request_body | ✅ |
| (gnostic.openapi.v3.operation).responses | ✅ |
| (gnostic.openapi.v3.operation).callbacks | ✅ |
| (gnostic.openapi.v3.operation).deprecated  | ✅ |
| (gnostic.openapi.v3.operation).security  | ✅ |
| (gnostic.openapi.v3.operation).servers  | ✅ |
| (gnostic.openapi.v3.operation).specification_extension | ✅ |

A `request_body` replaces the request body generated from the method's input. There are two exceptions:

- If a media type has no `schema`, the generated schema for that media type is used. For example, setting only an `example` on `application/json` keeps the generated schema and adds the example.
- If the request body has no `content`, the generated content and `required` are kept. Use this to set only a `description`.

Media types that aren't listed are removed. A `reference` replaces the whole body.

#### Message Options
| Option | Supported? | Notes |
|---|---|---|
| (gnostic.openapi.v3.schema).nullable | ✅ | Adds `null` to `type`. Only `true` can be set |
| (gnostic.openapi.v3.schema).discriminator | ✅ | |
| (gnostic.openapi.v3.schema).read_only | ✅ | Only `true` can be set |
| (gnostic.openapi.v3.schema).write_only | ✅ | Only `true` can be set |
| (gnostic.openapi.v3.schema).xml | ✅ | |
| (gnostic.openapi.v3.schema).external_docs | ✅ | |
| (gnostic.openapi.v3.schema).example | ✅ | YAML only; a protobuf `Any` value is ignored |
| (gnostic.openapi.v3.schema).deprecated | ✅ | Only `true` can be set |
| (gnostic.openapi.v3.schema).title | ✅ | |
| (gnostic.openapi.v3.schema).multiple_of | ✅ | |
| (gnostic.openapi.v3.schema).maximum | ✅ | `0` can't be set |
| (gnostic.openapi.v3.schema).exclusive_maximum | ✅ | Needs a nonzero `maximum` |
| (gnostic.openapi.v3.schema).minimum | ✅ | `0` can't be set |
| (gnostic.openapi.v3.schema).exclusive_minimum | ✅ | Needs a nonzero `minimum` |
| (gnostic.openapi.v3.schema).max_length | ✅ | |
| (gnostic.openapi.v3.schema).min_length | ✅ | |
| (gnostic.openapi.v3.schema).pattern | ✅ | |
| (gnostic.openapi.v3.schema).max_items | ✅ | Only for an `array` type |
| (gnostic.openapi.v3.schema).min_items | ✅ | Only for an `array` type |
| (gnostic.openapi.v3.schema).unique_items | ✅ | Only for an `array` type |
| (gnostic.openapi.v3.schema).max_properties | ✅ | Only for an `object` type |
| (gnostic.openapi.v3.schema).min_properties | ✅ | Only for an `object` type |
| (gnostic.openapi.v3.schema).required | ✅ | |
| (gnostic.openapi.v3.schema).enum | ✅ | |
| (gnostic.openapi.v3.schema).type | ✅ | Keeps `null` if the field is nullable |
| (gnostic.openapi.v3.schema).all_of | ✅ | |
| (gnostic.openapi.v3.schema).one_of | ✅ | |
| (gnostic.openapi.v3.schema).any_of | ✅ | |
| (gnostic.openapi.v3.schema).not | ✅ | |
| (gnostic.openapi.v3.schema).items | ✅ | |
| (gnostic.openapi.v3.schema).properties | ✅ | |
| (gnostic.openapi.v3.schema).additional_properties | ✅ | |
| (gnostic.openapi.v3.schema).default | ✅ | |
| (gnostic.openapi.v3.schema).description | ✅ | |
| (gnostic.openapi.v3.schema).format | ✅ | |
| (gnostic.openapi.v3.schema).specification_extension | ✅ | |

#### Field Options
| Option | Supported? | Notes |
|---|---|---|
| (gnostic.openapi.v3.property).nullable | ✅ | Adds `null` to `type`. Only `true` can be set |
| (gnostic.openapi.v3.property).discriminator | ✅ | |
| (gnostic.openapi.v3.property).read_only | ✅ | Only `true` can be set |
| (gnostic.openapi.v3.property).write_only | ✅ | Only `true` can be set |
| (gnostic.openapi.v3.property).xml | ✅ | |
| (gnostic.openapi.v3.property).external_docs | ✅ | |
| (gnostic.openapi.v3.property).example | ✅ | YAML only; a protobuf `Any` value is ignored |
| (gnostic.openapi.v3.property).deprecated | ✅ | Only `true` can be set |
| (gnostic.openapi.v3.property).title | ✅ | |
| (gnostic.openapi.v3.property).multiple_of | ✅ | |
| (gnostic.openapi.v3.property).maximum | ✅ | `0` can't be set |
| (gnostic.openapi.v3.property).exclusive_maximum | ✅ | Needs a nonzero `maximum` |
| (gnostic.openapi.v3.property).minimum | ✅ | `0` can't be set |
| (gnostic.openapi.v3.property).exclusive_minimum | ✅ | Needs a nonzero `minimum` |
| (gnostic.openapi.v3.property).max_length | ✅ | |
| (gnostic.openapi.v3.property).min_length | ✅ | |
| (gnostic.openapi.v3.property).pattern | ✅ | |
| (gnostic.openapi.v3.property).max_items | ✅ | Only for an `array` type |
| (gnostic.openapi.v3.property).min_items | ✅ | Only for an `array` type |
| (gnostic.openapi.v3.property).unique_items | ✅ | Only for an `array` type |
| (gnostic.openapi.v3.property).max_properties | ✅ | Only for an `object` type |
| (gnostic.openapi.v3.property).min_properties | ✅ | Only for an `object` type |
| (gnostic.openapi.v3.property).required | ✅ | |
| (gnostic.openapi.v3.property).enum | ✅ | |
| (gnostic.openapi.v3.property).type | ✅ | Keeps `null` if the field is nullable |
| (gnostic.openapi.v3.property).all_of | ✅ | |
| (gnostic.openapi.v3.property).one_of | ✅ | |
| (gnostic.openapi.v3.property).any_of | ✅ | |
| (gnostic.openapi.v3.property).not | ✅ | |
| (gnostic.openapi.v3.property).items | ✅ | |
| (gnostic.openapi.v3.property).properties | ✅ | |
| (gnostic.openapi.v3.property).additional_properties | ✅ | |
| (gnostic.openapi.v3.property).default | ✅ | |
| (gnostic.openapi.v3.property).description | ✅ | |
| (gnostic.openapi.v3.property).format | ✅ | |
| (gnostic.openapi.v3.property).specification_extension | ✅ | |

For more information on how to use each option in your Protobuf file, you can reference [the gnostic.openapi.v3 module documentation](https://buf.build/gnostic/gnostic/docs/main:gnostic.openapi.v3) and the [google/gnostic repo](https://github.com/google/gnostic). If you need something that isn't supported, please [create an issue](https://github.com/connectrpc/protoc-gen-connect-openapi/issues/new).
