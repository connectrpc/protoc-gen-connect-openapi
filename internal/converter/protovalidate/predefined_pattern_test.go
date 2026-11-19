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
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestExtractMatchesPattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		expression  string
		wantPattern string
		wantGuarded bool
		wantOK      bool
	}{
		{
			name:        "member call",
			expression:  `this.matches('^[a-z2-7]{25}[aeimquy4]$')`,
			wantPattern: `^[a-z2-7]{25}[aeimquy4]$`,
			wantOK:      true,
		},
		{
			name:        "global call",
			expression:  `matches(this, '^a$')`,
			wantPattern: `^a$`,
			wantOK:      true,
		},
		{
			name:        "guarded by rule",
			expression:  `!rule || this.matches('^a+$')`,
			wantPattern: `^a+$`,
			wantGuarded: true,
			wantOK:      true,
		},
		{
			name:        "escaped backslash and quotes",
			expression:  `this.matches("^\\d+\"x\"$")`,
			wantPattern: `^\d+"x"$`,
			wantOK:      true,
		},
		{
			name:        "raw string",
			expression:  `this.matches(r'^\d+$')`,
			wantPattern: `^\d+$`,
			wantOK:      true,
		},
		{name: "no matches call", expression: `this.size() > 0`},
		{name: "non literal regex", expression: `this.matches(other)`},
		{name: "empty regex", expression: `this.matches('')`},
		{name: "other subject", expression: `other.matches('x')`},
		{name: "conjunction", expression: `this.size() > 0 && this.matches('x')`},
		{name: "guard without negation", expression: `rule || this.matches('x')`},
		{name: "unrelated guard", expression: `!other || this.matches('x')`},
		{name: "negated match", expression: `!this.matches('x')`},
		{name: "invalid CEL", expression: `this.matches(`},
		{name: "empty", expression: ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pattern, guarded, ok := extractMatchesPattern(tt.expression)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantPattern, pattern)
			assert.Equal(t, tt.wantGuarded, guarded)
		})
	}
}

func TestUpdateSchemaWithPredefinedPattern(t *testing.T) {
	t.Parallel()

	rule := func(id, expression string) *validate.Rule {
		r := &validate.Rule{}
		r.SetId(id)
		r.SetExpression(expression)
		return r
	}
	tests := []struct {
		name    string
		initial string
		rules   []*validate.Rule
		ruleOn  bool
		want    string
	}{
		{
			name:   "unguarded sets pattern even if rule is off",
			rules:  []*validate.Rule{rule("a", `this.matches('^x$')`)},
			ruleOn: false,
			want:   "^x$",
		},
		{
			name:   "guarded with rule on sets pattern",
			rules:  []*validate.Rule{rule("a", `!rule || this.matches('^x$')`)},
			ruleOn: true,
			want:   "^x$",
		},
		{
			name:   "guarded with rule off is skipped",
			rules:  []*validate.Rule{rule("a", `!rule || this.matches('^x$')`)},
			ruleOn: false,
		},
		{
			name:    "existing pattern is kept",
			initial: "^std$",
			rules:   []*validate.Rule{rule("a", `this.matches('^x$')`)},
			ruleOn:  true,
			want:    "^std$",
		},
		{
			name: "first recognized rule by id wins regardless of input order",
			rules: []*validate.Rule{
				rule("c", `this.matches('^c$')`),
				rule("b", `this.matches('^b$')`),
				rule("a", `this.size() > 0`),
			},
			ruleOn: true,
			want:   "^b$",
		},
		{
			name:   "rule without expression is ignored",
			rules:  []*validate.Rule{rule("a", "")},
			ruleOn: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			schema := &base.Schema{Pattern: tt.initial}
			updateSchemaWithPredefinedPattern(schema, tt.rules, tt.ruleOn)
			assert.Equal(t, tt.want, schema.Pattern)
		})
	}
}

func TestRuleEnabled(t *testing.T) {
	t.Parallel()
	fields := (&validate.RepeatedRules{}).ProtoReflect().Descriptor().Fields()
	unique := fields.ByName("unique")      // bool
	minItems := fields.ByName("min_items") // uint64
	assert.True(t, ruleEnabled(protoreflect.ValueOfBool(true), unique))
	assert.False(t, ruleEnabled(protoreflect.ValueOfBool(false), unique))
	assert.False(t, ruleEnabled(protoreflect.ValueOfUint64(1), minItems))
}
