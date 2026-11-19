# Protovalidate Support
protoc-gen-connect-openapi has support for many [Protovalidate](https://github.com/bufbuild/protovalidate) annotations. Note that not every Protovalidate constraint translates clearly to OpenAPI.

Unconstrained `float` and `double` fields accept numbers and the ProtoJSON strings `"NaN"`, `"Infinity"`, and `"-Infinity"`. Fields with numeric bounds, `const`, `in`, `not_in`, or `finite: true` keep their numeric schema; special values under these constraints are not modeled. Explicit Gnostic property annotations also retain their previous schema behavior.

Your protobuf that looks like this:
```protobuf
syntax = "proto3";

import "buf/validate/validate.proto";

message User {
  int32 age = 1 [(buf.validate.field).int32.gte = 18];
}
```

... results in an OpenAPI file that contains the gte constraint as OpenAPI's "minimum" property:
```yaml
components:
  schemas:
    User:
      type: object
      properties:
        age:
          type: integer
          title: age
          minimum: 18
          format: int32
      title: User
      additionalProperties: false
```

Rules without a matching OpenAPI keyword are usually added to the description instead. For a custom CEL rule, this is its `message` (or its `id` if there is no message). The CEL expression itself is not rendered.
```protobuf
syntax = "proto3";

import "buf/validate/validate.proto";

package custom;

message User {
  int32 age = 1 [(buf.validate.field).cel = {
    id: "user.age",
    message: "The user can't be a minor (younger than 18 years old)",
    expression: "this < 18 ? 'User must be at least 18 years old': ''"
  }];
}
```

Result:
```yaml
components:
  schemas:
    custom.User:
      type: object
      properties:
        age:
          type: integer
          title: age
          format: int32
          description: |
            The user can't be a minor (younger than 18 years old)
      title: User
      additionalProperties: false
```

## Predefined Rules
[Predefined rules](https://buf.build/docs/protovalidate/schemas/predefined-rules/) are custom extensions of the standard rule messages, such as `buf.validate.StringRules`. Their `message` is added to the description (or the rule `id` if there is no message); the CEL expression itself is not rendered.

A predefined rule whose expression is a plain regular expression match on the value is also rendered as the `pattern` keyword. These forms are recognized:

* `this.matches('<regex>')`
* `matches(this, '<regex>')`
* `!rule || this.matches('<regex>')`, which only applies when the rule is set to `true`

```protobuf
extend buf.validate.StringRules {
  optional bool key = 80048955 [(buf.validate.predefined).cel = {
    id: "string.key"
    message: "must be a key"
    expression: "this.matches('^[a-z2-7]{25}[aeimquy4]$')"
  }];
}

message Item {
  string id = 1 [(buf.validate.field).string.(key) = true];
}
```

Result:
```yaml
components:
  schemas:
    Item:
      properties:
        id:
          type: string
          title: id
          pattern: ^[a-z2-7]{25}[aeimquy4]$
          description: |
            must be a key
```

A boolean rule that is set to `false` is left out when its expression is guarded with `!rule ||`, since it then always passes.

Any other expression (for example one combined with `&&`, or a regular expression that isn't a string literal) is not converted. If the field already has a `pattern`, such as one from `(buf.validate.field).string.pattern`, it is kept. If several predefined rules have a pattern, the first one is used: extensions are ordered by name, and the rules of one extension by `id`.

## Supported Options
Each option is marked with how it is rendered:

* ✅ Rendered as schema keywords.
* 🟠 Only added to the description.
* ❌ Not rendered.

### Message Options
| Option | Supported? | Notes |
|---|---|---|
| (buf.validate.message).cel | 🟠 | |
| (buf.validate.message).cel_expression | ❌ | |
| (buf.validate.message).oneof | ✅ | |

### Field Options
| Option | Supported? | Notes |
|---|---|---|
| (buf.validate.field).cel | 🟠 | |
| (buf.validate.field).cel_expression | ❌ | |
| (buf.validate.field).ignore | ✅ | `IGNORE_IF_ZERO_VALUE` also accepts the zero value. `IGNORE_ALWAYS` turns off the field's rules |
| (buf.validate.field).required | ✅ | |
| (buf.validate.field).any.in | ✅ | Set on the `@type` property |
| (buf.validate.field).any.not_in | ✅ | Set on the `@type` property |
| (buf.validate.field).bool.const | ✅ | |
| (buf.validate.field).bool.example | ✅ | |
| (buf.validate.field).bytes.const | ✅ | |
| (buf.validate.field).bytes.contains | 🟠 | |
| (buf.validate.field).bytes.in | ✅ | |
| (buf.validate.field).bytes.ip | 🟠 | |
| (buf.validate.field).bytes.ipv4 | 🟠 | |
| (buf.validate.field).bytes.ipv6 | 🟠 | |
| (buf.validate.field).bytes.len | 🟠 | |
| (buf.validate.field).bytes.max_len | 🟠 | |
| (buf.validate.field).bytes.min_len | 🟠 | |
| (buf.validate.field).bytes.not_in | ✅ | |
| (buf.validate.field).bytes.pattern | 🟠 | |
| (buf.validate.field).bytes.prefix | 🟠 | |
| (buf.validate.field).bytes.suffix | 🟠 | |
| (buf.validate.field).bytes.uuid | 🟠 | |
| (buf.validate.field).bytes.example | ✅ | |
| (buf.validate.field).double.const | ✅ | |
| (buf.validate.field).double.finite | ✅ | Only numbers are accepted |
| (buf.validate.field).double.gt | ✅ | |
| (buf.validate.field).double.gte | ✅ | |
| (buf.validate.field).double.in | ✅ | |
| (buf.validate.field).double.lt | ✅ | |
| (buf.validate.field).double.lte | ✅ | |
| (buf.validate.field).double.not_in | ✅ | |
| (buf.validate.field).double.example | ✅ | |
| (buf.validate.field).duration.const | ✅ | |
| (buf.validate.field).duration.gt | 🟠 | |
| (buf.validate.field).duration.gte | 🟠 | |
| (buf.validate.field).duration.in | ✅ | |
| (buf.validate.field).duration.lt | 🟠 | |
| (buf.validate.field).duration.lte | 🟠 | |
| (buf.validate.field).duration.not_in | ✅ | |
| (buf.validate.field).duration.example | ✅ | |
| (buf.validate.field).enum.const | ✅ | |
| (buf.validate.field).enum.defined_only | ✅ | The enum's schema already lists only the defined values |
| (buf.validate.field).enum.in | ✅ | |
| (buf.validate.field).enum.not_in | ✅ | |
| (buf.validate.field).enum.example | ✅ | |
| (buf.validate.field).field_mask.const | ✅ | |
| (buf.validate.field).field_mask.in | 🟠 | |
| (buf.validate.field).field_mask.not_in | 🟠 | |
| (buf.validate.field).field_mask.example | ✅ | |
| (buf.validate.field).fixed32.const | ✅ | |
| (buf.validate.field).fixed32.gt | ✅ | |
| (buf.validate.field).fixed32.gte | ✅ | |
| (buf.validate.field).fixed32.in | ✅ | |
| (buf.validate.field).fixed32.lt | ✅ | |
| (buf.validate.field).fixed32.lte | ✅ | |
| (buf.validate.field).fixed32.not_in | ✅ | |
| (buf.validate.field).fixed32.example | ✅ | |
| (buf.validate.field).fixed64.const | ✅ | |
| (buf.validate.field).fixed64.gt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).fixed64.gte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).fixed64.in | ✅ | |
| (buf.validate.field).fixed64.lt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).fixed64.lte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).fixed64.not_in | ✅ | |
| (buf.validate.field).fixed64.example | ✅ | |
| (buf.validate.field).float.const | ✅ | |
| (buf.validate.field).float.finite | ✅ | Only numbers are accepted |
| (buf.validate.field).float.gt | ✅ | |
| (buf.validate.field).float.gte | ✅ | |
| (buf.validate.field).float.in | ✅ | |
| (buf.validate.field).float.lt | ✅ | |
| (buf.validate.field).float.lte | ✅ | |
| (buf.validate.field).float.not_in | ✅ | |
| (buf.validate.field).float.example | ✅ | |
| (buf.validate.field).int32.const | ✅ | |
| (buf.validate.field).int32.gt | ✅ | |
| (buf.validate.field).int32.gte | ✅ | |
| (buf.validate.field).int32.in | ✅ | |
| (buf.validate.field).int32.lt | ✅ | |
| (buf.validate.field).int32.lte | ✅ | |
| (buf.validate.field).int32.not_in | ✅ | |
| (buf.validate.field).int32.example | ✅ | |
| (buf.validate.field).int64.const | ✅ | |
| (buf.validate.field).int64.gt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).int64.gte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).int64.in | ✅ | |
| (buf.validate.field).int64.lt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).int64.lte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).int64.not_in | ✅ | |
| (buf.validate.field).int64.example | ✅ | |
| (buf.validate.field).map.keys | ✅ | String keys only, as `propertyNames` |
| (buf.validate.field).map.max_pairs | ✅ | |
| (buf.validate.field).map.min_pairs | ✅ | |
| (buf.validate.field).map.values | ✅ | |
| (buf.validate.field).repeated.items | ✅ | |
| (buf.validate.field).repeated.max_items | ✅ | |
| (buf.validate.field).repeated.min_items | ✅ | |
| (buf.validate.field).repeated.unique | ✅ | |
| (buf.validate.field).sfixed32.const | ✅ | |
| (buf.validate.field).sfixed32.gt | ✅ | |
| (buf.validate.field).sfixed32.gte | ✅ | |
| (buf.validate.field).sfixed32.in | ✅ | |
| (buf.validate.field).sfixed32.lt | ✅ | |
| (buf.validate.field).sfixed32.lte | ✅ | |
| (buf.validate.field).sfixed32.not_in | ✅ | |
| (buf.validate.field).sfixed32.example | ✅ | |
| (buf.validate.field).sfixed64.const | ✅ | |
| (buf.validate.field).sfixed64.gt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).sfixed64.gte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).sfixed64.in | ✅ | |
| (buf.validate.field).sfixed64.lt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).sfixed64.lte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).sfixed64.not_in | ✅ | |
| (buf.validate.field).sfixed64.example | ✅ | |
| (buf.validate.field).sint32.const | ✅ | |
| (buf.validate.field).sint32.gt | ✅ | |
| (buf.validate.field).sint32.gte | ✅ | |
| (buf.validate.field).sint32.in | ✅ | |
| (buf.validate.field).sint32.lt | ✅ | |
| (buf.validate.field).sint32.lte | ✅ | |
| (buf.validate.field).sint32.not_in | ✅ | |
| (buf.validate.field).sint32.example | ✅ | |
| (buf.validate.field).sint64.const | ✅ | |
| (buf.validate.field).sint64.gt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).sint64.gte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).sint64.in | ✅ | |
| (buf.validate.field).sint64.lt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).sint64.lte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).sint64.not_in | ✅ | |
| (buf.validate.field).sint64.example | ✅ | |
| (buf.validate.field).string.address | ✅ | |
| (buf.validate.field).string.const | ✅ | |
| (buf.validate.field).string.contains | 🟠 | |
| (buf.validate.field).string.email | ✅ | |
| (buf.validate.field).string.host_and_port | 🟠 | |
| (buf.validate.field).string.hostname | ✅ | |
| (buf.validate.field).string.in | ✅ | |
| (buf.validate.field).string.ip | ✅ | |
| (buf.validate.field).string.ip_prefix | 🟠 | |
| (buf.validate.field).string.ip_with_prefixlen | 🟠 | |
| (buf.validate.field).string.ipv4 | ✅ | |
| (buf.validate.field).string.ipv4_prefix | 🟠 | |
| (buf.validate.field).string.ipv4_with_prefixlen | 🟠 | |
| (buf.validate.field).string.ipv6 | ✅ | |
| (buf.validate.field).string.ipv6_prefix | 🟠 | |
| (buf.validate.field).string.ipv6_with_prefixlen | 🟠 | |
| (buf.validate.field).string.len | ✅ | |
| (buf.validate.field).string.len_bytes | 🟠 | |
| (buf.validate.field).string.max_bytes | 🟠 | |
| (buf.validate.field).string.max_len | ✅ | |
| (buf.validate.field).string.min_bytes | 🟠 | |
| (buf.validate.field).string.min_len | ✅ | |
| (buf.validate.field).string.not_contains | 🟠 | |
| (buf.validate.field).string.not_in | ✅ | |
| (buf.validate.field).string.pattern | ✅ | |
| (buf.validate.field).string.prefix | 🟠 | |
| (buf.validate.field).string.protobuf_dot_fqn | 🟠 | |
| (buf.validate.field).string.protobuf_fqn | 🟠 | |
| (buf.validate.field).string.strict | ❌ | Only changes how `well_known_regex` is checked |
| (buf.validate.field).string.suffix | 🟠 | |
| (buf.validate.field).string.tuuid | 🟠 | |
| (buf.validate.field).string.ulid | 🟠 | |
| (buf.validate.field).string.uri | ✅ | |
| (buf.validate.field).string.uri_ref | ✅ | |
| (buf.validate.field).string.uuid | ✅ | |
| (buf.validate.field).string.well_known_regex | 🟠 | |
| (buf.validate.field).string.example | ✅ | |
| (buf.validate.field).timestamp.const | ✅ | |
| (buf.validate.field).timestamp.gt | 🟠 | |
| (buf.validate.field).timestamp.gt_now | 🟠 | |
| (buf.validate.field).timestamp.gte | 🟠 | |
| (buf.validate.field).timestamp.lt | 🟠 | |
| (buf.validate.field).timestamp.lt_now | 🟠 | |
| (buf.validate.field).timestamp.lte | 🟠 | |
| (buf.validate.field).timestamp.within | 🟠 | |
| (buf.validate.field).timestamp.example | ✅ | |
| (buf.validate.field).uint32.const | ✅ | |
| (buf.validate.field).uint32.gt | ✅ | |
| (buf.validate.field).uint32.gte | ✅ | |
| (buf.validate.field).uint32.in | ✅ | |
| (buf.validate.field).uint32.lt | ✅ | |
| (buf.validate.field).uint32.lte | ✅ | |
| (buf.validate.field).uint32.not_in | ✅ | |
| (buf.validate.field).uint32.example | ✅ | |
| (buf.validate.field).uint64.const | ✅ | |
| (buf.validate.field).uint64.gt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).uint64.gte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).uint64.in | ✅ | |
| (buf.validate.field).uint64.lt | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).uint64.lte | 🟠 | 64-bit integers are JSON strings |
| (buf.validate.field).uint64.not_in | ✅ | |
| (buf.validate.field).uint64.example | ✅ | |

### OneOf Options
| Option | Supported? | Notes |
|---|---|---|
| (buf.validate.oneof).required | ✅ | |
