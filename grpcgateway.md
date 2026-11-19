# gRPC-Gateway Support
protoc-gen-connect-openapi documents the HTTP routes defined with [`google.api.http`](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto) rules. These are the rules used by [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway), [Vanguard](https://github.com/connectrpc/vanguard-go), and Envoy's gRPC-JSON transcoder.

This support is on by default. It's controlled by the `features` option: with `features=google.api.http`, only methods with a rule are documented, and with `features=connectrpc`, rules are ignored. See the [README](README.md#features) for details.

Here's an example:

```protobuf
syntax = "proto3";

package library.v1;

import "google/api/annotations.proto";

service LibraryService {
  rpc GetBook(GetBookRequest) returns (Book) {
    option (google.api.http) = {get: "/v1/{name=shelves/*/books/*}"};
  }
  rpc UpdateBook(UpdateBookRequest) returns (Book) {
    option (google.api.http) = {
      patch: "/v1/{book.name=shelves/*/books/*}"
      body: "book"
    };
  }
}

message GetBookRequest {
  string name = 1;
  string view = 2;
}

message UpdateBookRequest {
  Book book = 1;
  bool allow_missing = 2;
}

message Book {
  string name = 1;
  string title = 2;
}
```

Result (abridged):
```yaml
paths:
  /v1/shelves/{shelf}/books/{book}:
    get:
      operationId: library.v1.LibraryService.GetBook
      parameters:
        - name: shelf
          in: path
          required: true
          schema:
            type: string
        - name: book
          in: path
          required: true
          schema:
            type: string
        - name: view
          in: query
          schema:
            type: string
            title: view
      responses:
        default:
          description: An unexpected error response.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/google.rpc.Status'
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/library.v1.Book'
    patch:
      operationId: library.v1.LibraryService.UpdateBook
      parameters:
        # shelf and book path parameters, as above
        - name: allowMissing
          in: query
          schema:
            type: boolean
            title: allow_missing
      requestBody:
        content:
          application/json:
            schema:
              title: book
              $ref: '#/components/schemas/library.v1.Book'
```

## How rules are rendered
- **Methods:** `get`, `put`, `post`, `delete`, and `patch` become operations of that method. A `custom` rule becomes an operation if its kind is `HEAD`, `OPTIONS`, or `TRACE`. Other kinds have no OpenAPI operation, so the rule is skipped with a warning.
- **Path parameters:** `{field}` and nested `{parent.field}` become path parameters. A template such as `{name=shelves/*/books/*}` is split into one parameter per `*`, named after the collection (`shelf`, `book`).
- **Request body:** `body: "*"` sends the request message without the path parameters. `body: "<field>"` sends only that field.
- **Query parameters:** request fields that aren't in the path or the body become query parameters. Message fields are flattened into dotted names.
- **Response body:** the response message, or only the `response_body` field when it's set.
- **Errors:** the default response is [`google.rpc.Status`](https://github.com/googleapis/googleapis/blob/master/google/rpc/status.proto). Use `with-google-error-detail` to type its `details`.
- **Additional bindings:** each one becomes another operation. Operation IDs get a number suffix, starting at 2, to stay unique.

A gnostic `request_body` annotation changes the generated request body; see [Gnostic Support](gnostic.md#method-options).

For more about writing these rules, see the [`google.api.http` reference](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto) and gRPC-Gateway's [adding annotations](https://grpc-ecosystem.github.io/grpc-gateway/docs/tutorials/adding_annotations/) tutorial. If something you need isn't supported, please [create an issue](https://github.com/connectrpc/protoc-gen-connect-openapi/issues/new).
