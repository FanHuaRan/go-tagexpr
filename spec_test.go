// Copyright 2019 Bytedance Inc. All Rights Reserved.
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

package tagexpr

import (
	"fmt"
	"reflect"
	"testing"
)

func TestReadPairedSymbol(t *testing.T) {
	var cases = []struct {
		expr         string
		val          string
		lastExprNode string
		left, right  rune
	}{
		{expr: "'true '+'a'", val: "true ", lastExprNode: "+'a'", left: '\'', right: '\''},
		{expr: "((0+1)/(2-1)*9)%2", val: "(0+1)/(2-1)*9", lastExprNode: "%2", left: '(', right: ')'},
	}
	for _, c := range cases {
		t.Log(c.expr)
		expr := c.expr
		got := readPairedSymbol(&expr, c.left, c.right)
		if *got != c.val || expr != c.lastExprNode {
			t.Fatalf("expr: %q, got: %q, %q, want: %q, %q", c.expr, *got, expr, c.val, c.lastExprNode)
		}
	}
}

func TestReadBoolExprNode(t *testing.T) {
	var cases = []struct {
		expr         string
		val          bool
		lastExprNode string
	}{
		{expr: "false", val: false, lastExprNode: ""},
		{expr: "true", val: true, lastExprNode: ""},
		{expr: "true ", val: true, lastExprNode: " "},
		{expr: "!true&", val: false, lastExprNode: "&"},
		{expr: "!false|", val: true, lastExprNode: "|"},
		{expr: "!!!!false =", val: !!!!false, lastExprNode: " ="},
	}
	for _, c := range cases {
		t.Log(c.expr)
		expr := c.expr
		e := readBoolExprNode(&expr)
		got := e.Run("", nil).(bool)
		if got != c.val || expr != c.lastExprNode {
			t.Fatalf("expr: %s, got: %v, %s, want: %v, %s", c.expr, got, expr, c.val, c.lastExprNode)
		}
	}
}

func TestReadDigitalExprNode(t *testing.T) {
	var cases = []struct {
		expr         string
		val          float64
		lastExprNode string
	}{
		{expr: "0.1 +1", val: 0.1, lastExprNode: " +1"},
		{expr: "-1\\1", val: -1, lastExprNode: "\\1"},
		{expr: "1a", val: 0, lastExprNode: ""},
		{expr: "1", val: 1, lastExprNode: ""},
		{expr: "1.1", val: 1.1, lastExprNode: ""},
		{expr: "1.1/", val: 1.1, lastExprNode: "/"},
	}
	for _, c := range cases {
		expr := c.expr
		e := readDigitalExprNode(&expr)
		if c.expr == "1a" {
			if e != nil {
				t.Fatalf("expr: %s, got:%v, want:%v", c.expr, e.Run("", nil), nil)
			}
			continue
		}
		got := e.Run("", nil).(float64)
		if got != c.val || expr != c.lastExprNode {
			t.Fatalf("expr: %s, got: %f, %s, want: %f, %s", c.expr, got, expr, c.val, c.lastExprNode)
		}
	}
}

func TestFindSelector(t *testing.T) {
	var cases = []struct {
		expr          string
		field         string
		name          string
		subSelector   []string
		boolOpposite  bool
		floatOpposite bool
		found         bool
		last          string
	}{
		{expr: "$", name: "$", found: true},
		{expr: "!!$", name: "$", found: true},
		{expr: "!$", name: "$", boolOpposite: true, found: true},
		{expr: "+$", name: "$", found: true},
		{expr: "--$", name: "$", found: true},
		{expr: "-$", name: "$", floatOpposite: true, found: true},
		{expr: "---$", name: "$", floatOpposite: true, found: true},
		{expr: "()$", last: "()$"},
		{expr: "(0)$", last: "(0)$"},
		{expr: "(A)$", field: "A", name: "$", found: true},
		{expr: "+(A)$", field: "A", name: "$", found: true},
		{expr: "++(A)$", field: "A", name: "$", found: true},
		{expr: "!(A)$", field: "A", name: "$", boolOpposite: true, found: true},
		{expr: "-(A)$", field: "A", name: "$", floatOpposite: true, found: true},
		{expr: "(A0)$", field: "A0", name: "$", found: true},
		{expr: "!!(A0)$", field: "A0", name: "$", found: true},
		{expr: "--(A0)$", field: "A0", name: "$", found: true},
		{expr: "(A0)$(A1)$", last: "(A0)$(A1)$"},
		{expr: "(A0)$ $(A1)$", field: "A0", name: "$", found: true, last: " $(A1)$"},
		{expr: "$a", last: "$a"},
		{expr: "$[1]['a']", name: "$", subSelector: []string{"1", "'a'"}, found: true},
		{expr: "$[1][]", last: "$[1][]"},
		{expr: "$[[]]", last: "$[[]]"},
		{expr: "$[[[]]]", last: "$[[[]]]"},
		{expr: "$[(A)$[1]]", name: "$", subSelector: []string{"(A)$[1]"}, found: true},
		{expr: "$>0&&$<10", name: "$", found: true, last: ">0&&$<10"},
	}
	for _, c := range cases {
		last := c.expr
		field, name, subSelector, boolOpposite, floatOpposite, found := findSelector(&last)
		if found != c.found {
			t.Fatalf("%q found: got: %v, want: %v", c.expr, found, c.found)
		}
		if c.boolOpposite && (boolOpposite == nil || !*boolOpposite) {
			t.Fatalf("%q boolOpposite: got: %v, want: %v", c.expr, boolOpposite, c.boolOpposite)
		}
		if floatOpposite != c.floatOpposite {
			t.Fatalf("%q floatOpposite: got: %v, want: %v", c.expr, floatOpposite, c.floatOpposite)
		}
		if field != c.field {
			t.Fatalf("%q field: got: %q, want: %q", c.expr, field, c.field)
		}
		if name != c.name {
			t.Fatalf("%q name: got: %q, want: %q", c.expr, name, c.name)
		}
		if !reflect.DeepEqual(subSelector, c.subSelector) {
			t.Fatalf("%q subSelector: got: %v, want: %v", c.expr, subSelector, c.subSelector)
		}
		if last != c.last {
			t.Fatalf("%q last: got: %q, want: %q", c.expr, last, c.last)
		}
	}
}
func printBoolPtr(b *bool) string {
	var v interface{} = b
	if b != nil {
		v = *b
	}
	return fmt.Sprint(v)
}
