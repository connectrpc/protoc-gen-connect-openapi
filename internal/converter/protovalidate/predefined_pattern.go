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
	"strings"
	"sync"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// celParseEnv returns a shared CEL environment used only for parsing. Parsing
// does not need variable declarations, so the environment is the bare default.
var celParseEnv = sync.OnceValues(func() (*cel.Env, error) {
	return cel.NewEnv()
})

// extractMatchesPattern recognizes CEL expressions of the form
// `this.matches('<regex>')`, `matches(this, '<regex>')`, and either of these
// guarded as `!rule || <matches>`. guarded reports the latter, meaning the
// match only applies when the predefined rule's own value is true.
//
// Anything else (for example combined with `&&`, a non-literal regex, or other
// functions) is not recognized, because it can't be represented as a single
// pattern without changing its meaning.
func extractMatchesPattern(expression string) (pattern string, guarded, ok bool) {
	// Cheap pre-check: most predefined rules are not regex matches, so skip the
	// parser for them.
	if !strings.Contains(expression, overloads.Matches) {
		return "", false, false
	}
	root, ok := parseExpression(expression)
	if !ok {
		return "", false, false
	}

	if args, ok := callArgs(root, operators.LogicalOr, 2); ok && isNegatedRule(args[0]) {
		guarded, root = true, args[1]
	}
	pattern, ok = matchesLiteral(root)
	return pattern, guarded && ok, ok
}

// parseExpression parses a CEL expression without type-checking it.
func parseExpression(expression string) (ast.Expr, bool) {
	env, err := celParseEnv()
	if err != nil {
		return nil, false
	}
	parsed, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return nil, false
	}
	return parsed.NativeRep().Expr(), true
}

// passesWhenRuleIsFalse reports whether an expression has `!rule` as an
// operand of a top-level `||`, so it always passes when the rule's value is
// false. Built-in rules write the same guard as `!rules.<name>`, where name is
// the rule's field name.
func passesWhenRuleIsFalse(expression, name string) bool {
	// Cheap pre-check, as in extractMatchesPattern.
	if !strings.Contains(expression, "rule") {
		return false
	}
	root, ok := parseExpression(expression)
	if !ok {
		return false
	}
	return slices.ContainsFunc(orOperands(root), func(operand ast.Expr) bool {
		if isNegatedRule(operand) {
			return true
		}
		args, ok := callArgs(operand, operators.LogicalNot, 1)
		if !ok || args[0].Kind() != ast.SelectKind {
			return false
		}
		selected := args[0].AsSelect()
		return isIdent(selected.Operand(), "rules") && selected.FieldName() == name
	})
}

// orOperands returns the operands of e's top-level `||` chain, or e itself
// if it isn't one.
func orOperands(e ast.Expr) []ast.Expr {
	args, ok := callArgs(e, operators.LogicalOr, 2)
	if !ok {
		return []ast.Expr{e}
	}
	return append(orOperands(args[0]), orOperands(args[1])...)
}

// callArgs returns the arguments of e if it is a call of function with
// exactly n arguments (the target of a member call counts as the first one).
func callArgs(e ast.Expr, function string, n int) ([]ast.Expr, bool) {
	if e.Kind() != ast.CallKind {
		return nil, false
	}
	call := e.AsCall()
	if call.FunctionName() != function {
		return nil, false
	}
	args := call.Args()
	if call.IsMemberFunction() {
		args = append([]ast.Expr{call.Target()}, args...)
	}
	return args, len(args) == n
}

// isIdent reports whether e is the identifier name.
func isIdent(e ast.Expr, name string) bool {
	return e.Kind() == ast.IdentKind && e.AsIdent() == name
}

// isNegatedRule reports whether e is `!rule`.
func isNegatedRule(e ast.Expr) bool {
	args, ok := callArgs(e, operators.LogicalNot, 1)
	return ok && isIdent(args[0], "rule")
}

// matchesLiteral returns the regex if e is `this.matches('<literal>')` or
// `matches(this, '<literal>')`.
func matchesLiteral(e ast.Expr) (string, bool) {
	args, ok := callArgs(e, overloads.Matches, 2)
	if !ok || !isIdent(args[0], "this") || args[1].Kind() != ast.LiteralKind {
		return "", false
	}
	pattern, ok := args[1].AsLiteral().Value().(string)
	return pattern, ok && pattern != ""
}

// updateSchemaWithPredefinedPattern sets schema.Pattern from the first
// predefined rule whose CEL expression is a plain regex match. Rules are tried
// in order of id. An existing pattern (for example from the standard
// `string.pattern` rule) is kept. ruleOn is the value of a boolean predefined
// rule; it controls expressions guarded with `!rule ||`. The rule's message is
// added to the description separately, by updateWithCEL.
func updateSchemaWithPredefinedPattern(schema *base.Schema, rules []*validate.Rule, ruleOn bool) {
	if schema.Pattern != "" {
		return
	}
	slices.SortStableFunc(rules, func(a, b *validate.Rule) int {
		return strings.Compare(a.GetId(), b.GetId())
	})
	for _, rule := range rules {
		pattern, guarded, ok := extractMatchesPattern(rule.GetExpression())
		if !ok || (guarded && !ruleOn) {
			continue
		}
		schema.Pattern = pattern
		return
	}
}

// enabledPredefinedRules returns the rules that can fail for the rule's value.
// A boolean rule set to false leaves out the rules it guards with `!rule ||`,
// since they always pass.
func enabledPredefinedRules(rules []*validate.Rule, val protoreflect.Value, fieldDesc protoreflect.FieldDescriptor) []*validate.Rule {
	if fieldDesc.Kind() != protoreflect.BoolKind || fieldDesc.IsList() || val.Bool() {
		return rules
	}
	return slices.DeleteFunc(slices.Clone(rules), func(rule *validate.Rule) bool {
		return passesWhenRuleIsFalse(rule.GetExpression(), string(fieldDesc.Name()))
	})
}

// ruleEnabled reports whether a predefined rule is a boolean set to true.
// Non-boolean rule values can't be used in a `!rule` guard, so they are
// treated as not enabled.
func ruleEnabled(val protoreflect.Value, fieldDesc protoreflect.FieldDescriptor) bool {
	if fieldDesc.Kind() != protoreflect.BoolKind || fieldDesc.IsList() || fieldDesc.IsMap() {
		return false
	}
	return val.Bool()
}
