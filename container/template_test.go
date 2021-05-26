package container

import (
	"testing"
)

type TestVariables struct {
	String string
	Int    int
	Uint32 uint32
}

type TestVariablesStruct struct {
	Vars TestVariables
}

func TestTemplateVars(t *testing.T) {

	vars := struct {
		Vars    TestVariables
		VarsPtr *TestVariables
	}{
		TestVariables{
			String: "some-string",
			Int:    42,
			Uint32: 0,
		},
		&TestVariables{
			String: "some-string-ptr",
			Int:    24,
			Uint32: 1,
		},
	}

	type testcase struct {
		name string
		req  []string
		res  []string
	}
	testcases := []testcase{
		{"malformed", []string{"{{"}, []string{"E"}},
		{"empty", []string{}, []string{}},
		{"normal text", []string{"normal", "text"}, []string{"normal", "text"}},
		{"text with spaces", []string{"text", "with spaces"}, []string{"text", "with spaces"}},
		{"templ only space", []string{"{{        }}"}, []string{}},
		{"short var-path", []string{"{{.Vars}}"}, []string{"E"}},
		{"extra var-path", []string{"{{.Vars.String.Other}}"}, []string{"E"}},
		{"Vars.String", []string{"{{.Vars.String}}"}, []string{"some-string"}},
		{"Vars.Int", []string{"{{.Vars.Int}}"}, []string{"42"}},
		{"Vars.Uint32", []string{"{{.Vars.Uint32}}"}, []string{"0"}},
		{"VarsPtr.String", []string{"{{.VarsPtr.String}}"}, []string{"some-string-ptr"}},
		{"VarsPtr.Int", []string{"{{.VarsPtr.Int}}"}, []string{"24"}},
		{"VarsPtr.Uint32", []string{"{{.VarsPtr.Uint32}}"}, []string{"1"}},
	}

	for _, tc := range testcases {
		res, err := expandLine(tc.req, vars)
		if err != nil {
			if len(tc.res) != 1 || tc.res[0] != "E" {
				t.Errorf("testcase: '%s' failed with error %v", tc.name, err)
			}
			continue
		} else if len(tc.res) > 0 && tc.res[0] == "E" {
			t.Errorf("testcase: '%s' failed, expected error", tc.name)
			continue
		}

		if len(res) != len(tc.res) {
			t.Errorf("testcase: '%s' failed, expected %d args, got %d",
				tc.name, len(tc.res), len(res))
			continue
		}

		for i, r := range res {
			if r != tc.res[i] {
				t.Errorf("testcase: '%s' failed in arg %d %v %v",
					tc.name, i, r, tc.req[i])
			}
		}
	}
}

func TestTemplateConditional(t *testing.T) {

	vars := struct {
		Vars TestVariables
	}{
		TestVariables{
			String: "some-string",
			Int:    42,
		},
	}

	type testcase struct {
		name string
		req  []string
		res  string
	}

	testcases := []testcase{
		{"malformed if - no cond", []string{"{{if }}", "ok", "{{end}}"}, "E"},
		{"malformed if - no end", []string{"{{if false}}", "ok"}, "E"},
		{"simple true", []string{"{{if true}}", "ok", "{{end}}"}, "ok"},
		{"simple false", []string{"{{if false}}", "false", "{{end}}"}, "nil"},
		{"simple string compare",
			[]string{"{{if .Vars.String == \"some-string\"}}", "ok", "{{end}}"},
			"ok"},
		{"simple int compare equal",
			[]string{"{{if .Vars.Int == 42}}", "ok", "{{end}}"},
			"ok"},
		{"simple int compare not-equal",
			[]string{"{{if .Vars.Int!= 42}}", "ok", "{{end}}"},
			"nil"},
		{"simple unary not",
			[]string{"{{if !false}}", "ok", "{{end}}"},
			"ok"},
		{"unary not on var",
			[]string{"{{if !.Vars.Uint32}}", "ok", "{{end}}"},
			"ok"},
		// <, >, <=, >= not yet supported
		{"simple int compare less",
			[]string{"{{if .Vars.Int < 43}}", "ok", "{{end}}"},
			"E"},
		{"simple int compare less-equal",
			[]string{"{{if .Vars.Int <= 42}}", "ok", "{{end}}"},
			"E"},
		{"simple int compare greater",
			[]string{"{{if .Vars.Int > 41}}", "ok", "{{end}}"},
			"E"},
		{"simple int compare greater-equal",
			[]string{"{{if .Vars.Int >= 42}}", "ok", "{{end}}"},
			"E"},
		{"only braces",
			[]string{"{{if (((true)))}}", "ok", "{{end}}"},
			"ok"},
		{"misaligned braces too many closers",
			[]string{"{{if (((true))))}}", "ok", "{{end}}"},
			"E"},
		{"misaligned braces too many openers",
			[]string{"{{if ((((true)))}}", "ok", "{{end}}"},
			"E"},
		{"operator preferences",
			[]string{"{{if (1 || 0) || (0 || 1) == 0}}", "ok", "{{end}}"},
			"ok"},
		{"boolean order",
			[]string{"{{if true || true && false}}", "ok", "{{end}}"},
			"ok"},
		{"boolean order with blocks",
			[]string{"{{if (true || true) && false}}", "ok", "{{end}}"},
			"nil"},
		{"boolean order with many blocks",
			[]string{"{{if ((true || false) && (false || true)) != " +
				"((true || false) && (false || false))}}", "ok", "{{end}}"},
			"ok"},
		{"compare empty strings",
			[]string{"{{if \"\" == \"\"}}", "ok", "{{end}}"},
			"ok"},
		{"simple string test",
			[]string{"{{if a == a}}", "ok", "{{end}}"},
			"ok"},
		{"empty list",
			[]string{"{{if \"a\" in []}}", "ok", "{{end}}"},
			"E"},
		{"malformed list left",
			[]string{"{{if a in ]}}", "ok", "{{end}}"},
			"E"},
		{"malformed list right",
			[]string{"{{if a in [}}", "ok", "{{end}}"},
			"E"},
		{"malformed list no sep",
			[]string{"{{if a in [a, b c]}}", "E", "{{end}}"},
			"E"},
		{"single-entry list found",
			[]string{"{{if \"a\" in [a]}}", "ok", "{{end}}"},
			"ok"},
		{"multi-entry list found",
			[]string{"{{if \"a\" in [b, c, a]}}", "ok", "{{end}}"},
			"ok"},
	}

	for _, tc := range testcases {
		res, err := expandLine(tc.req, vars)
		if err != nil {
			if len(tc.res) != 1 || tc.res != "E" {
				t.Errorf("testcase: '%s' failed with error %v", tc.name, err)
			}
			continue
		} else if len(tc.res) > 0 && tc.res == "E" {
			t.Errorf("testcase: '%s' failed, expected error", tc.name)
			continue
		}

		if len(res) == 0 && tc.res != "nil" {
			t.Errorf("testcase: '%s' failed, expected a return value", tc.name)
		} else if len(res) != 0 && tc.res == "nil" {
			t.Errorf("testcase: '%s' failed, expected no return value", tc.name)
		} else if len(res) > 0 && res[0] != tc.res {
			t.Errorf("testcase: '%s' failed, expected '%s', got '%s'",
				tc.name, tc.res, res[0])
		}
	}
}
