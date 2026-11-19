# Remote Plugin Usage

`protoc-gen-connect-openapi` is available as a [remote plugin](https://buf.build/docs/bsr/remote-plugins/usage/) on the Buf Schema Registry, so you can generate OpenAPI files with `buf generate` without installing anything:

```yaml
version: v2
plugins:
  - remote: buf.build/connectrpc/openapi:v0.30.0
    out: gen
```

Remote plugins run on the BSR, so they can't read files from your machine.

## Disabled Options

Because remote plugins can't read your files, these options are disabled:

| Option   | Description                                                                         |
|----------|-------------------------------------------------------------------------------------|
| `base`     | The path to a base OpenAPI file to populate fields that this tool doesn't populate. |
| `override` | The path to an override OpenAPI file to override schema components.                 |

Using either option with the remote plugin fails with an error.

## Alternative: Gnostic Annotations

Instead of `base` and `override` files, you can put OpenAPI details in your `.proto` files with the OpenAPI v3 annotations from the gnostic project. They work with the remote plugin because they don't need any files. See [Gnostic Support](gnostic.md) for every supported annotation.

### Example

This sets the document's `info`, which is a common reason to use a `base` file:

```protobuf
syntax = "proto3";

package my.service.v1;

import "gnostic/openapi/v3/annotations.proto";

option (gnostic.openapi.v3.document) = {
  info: {
    title: "My Awesome API";
    version: "1.0.0";
    description: "This is a sample server for a pet store.";
    contact: {
      name: "API Support";
      url: "http://www.example.com/support";
      email: "support@example.com";
    }
    license: {
      name: "Apache 2.0";
      url: "http://www.apache.org/licenses/LICENSE-2.0.html";
    }
  }
};

service MyService {
  // ...
}
```

Annotations cover most of what `base` and `override` files do. They can't change messages you don't own, such as well-known types or messages from dependencies, which an `override` file can replace.

## Alternative: Using a local plugin

If you need `base` or `override`, use a local plugin. See the [README](README.md#install) for how to install it.
