package container

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/czankel/cne/errdefs"
)

// templates start with '{{' and end with '}}'
// They can reference a variable or contain a command
//
// Examples
//   {{.Context.User.Username}} -> returns the value for the username
//   {{if .Parameters.Rebuild == true}} -> includes the next fields

func getVar(name string, vars interface{}) (string, error) {

	path := strings.Split(strings.Trim(name, " ."), ".")
	elem := reflect.ValueOf(vars)

	for _, sel := range path {

		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		idx := -1
		i := strings.Index(sel, "[")
		if i != -1 {
			if sel[len(sel)-1] != ']' {
				return "", errdefs.InvalidArgument(
					"invalid template (malformed index): %s", name)
			}
			var err error
			idx, err = strconv.Atoi(sel[i+1 : len(sel)-1])
			if err != nil {
				return "", errdefs.InvalidArgument(
					"invalid template (malformed index): %s", name)
			}
			sel = sel[:i]
		}

		if elem.Kind() != reflect.Struct {
			return "", errdefs.InvalidArgument("invalid template (no struct): %s", name)
		}

		elem = elem.FieldByName(sel)
		if !elem.IsValid() {
			return "", errdefs.InvalidArgument(
				"invalid template (no such field): %s", name)
		}

		if idx != -1 {
			if elem.Kind() != reflect.Slice {
				return "", errdefs.InvalidArgument(
					"invalid template (not an array): %s", name)
			}

			elem = elem.Index(idx)
			if !elem.IsValid() {
				return "", errdefs.InvalidArgument(
					"invalid template (invalid array index): %d in %s", i, name)
			}
		}
	}

	kind := elem.Kind()
	if !elem.CanInterface() ||
		(kind != reflect.String &&
			kind != reflect.Bool &&
			kind != reflect.Int &&
			kind != reflect.Uint32) {
		return "", errdefs.InvalidArgument("invalid template (value not found): %s", name)
	}
	val := fmt.Sprintf("%v", elem.Interface())
	return val, nil
}

func expandLine(line []string, vars interface{}) ([]string, error) {

	cmds := []string{}
	skip := 0
	nestedIfs := 0

	for _, arg := range line {

		if arg[0] != '{' || arg[1] != '{' {
			if skip == 0 {
				cmds = append(cmds, arg)
			}
			continue
		}

		l := len(arg)
		if l < 4 || (arg[l-2] != '}' || arg[l-1] != '}') {
			return nil, errdefs.InvalidArgument("malformed template: %s", arg)
		}

		templ := strings.TrimSpace(arg[2 : l-2])

		switch {
		case strings.HasPrefix(templ, "if "):
			nestedIfs = nestedIfs + 1
			if skip > 0 {
				skip = skip + 1
				continue
			}

			cond, err := parseConditional(templ[3:], vars)
			if err != nil {
				return nil, err
			}
			if !cond {
				skip = skip + 1
			}
			continue

		case templ == "end":
			if nestedIfs == 0 {
				return nil, errdefs.InvalidArgument("erraneous extra '{{end}}'")
			}
			nestedIfs = nestedIfs - 1

			if skip > 0 {
				skip = skip - 1
			}

			continue
		}

		if skip == 0 && strings.HasPrefix(templ, ".") {
			val, err := getVar(templ, vars)
			if err != nil {
				return nil, err
			}
			cmds = append(cmds, val)
		}
	}

	if nestedIfs > 0 {
		return nil, errdefs.InvalidArgument("missing {{end}}")
	}

	return cmds, nil
}

// Simple parser
//  - all comparisons are done on strings.
//  - empty strings are false, all others are tru. "false" and "0" are converted to an empty string
//  - only boolean operators ('==' and '!=') are supported (i.e. not: '>', '<', '>=', '<=')
//  - unary not operator ('!') is supported
//  - provide 'in' <list>
//  - precedence: '(...)' (group expressions)  =>  '==' and '!='  =>  '&&'  =>  '||',

type condElem struct {
	token string // one of: "val", "op", "unary", "start", "end"
	value string // operator, or intermediate value or result
	pos   int    // text position
}

func parseErr(desc, line string, elem *condElem) error {
	return errdefs.InvalidArgument("invalid argument: %s at pos %d '%s >> %c << %s'",
		desc, elem.pos, line[:elem.pos], line[elem.pos], line[elem.pos+1:])
}

func parseConditionalLexer(line string, vars interface{}) ([]condElem, error) {

	elems := []condElem{}
	//list := []
	l := len(line)
	for i := 0; i < l; i++ {

		if line[i] == ' ' {
			continue
		}

		elems = append(elems, condElem{pos: i})
		cur := &elems[len(elems)-1]

		if line[i] == '(' {
			cur.token = "start"
		} else if line[i] == ')' {
			cur.token = "end"
		} else if l-i > 1 && line[i] == '!' && line[i+1] == '=' {
			cur.token = "op"
			cur.value = "!="
			i++
		} else if l-i > 1 && line[i] == '=' && line[i+1] == '=' {
			cur.token = "op"
			cur.value = "=="
			i++
		} else if l-i > 1 && line[i] == '!' {
			cur.token = "unary"
			cur.value = "!"
		} else if l-i > 1 && line[i] == '|' && line[i+1] == '|' {
			cur.token = "op"
			cur.value = "||"
			i++
		} else if l-i > 1 && line[i] == '&' && line[i+1] == '&' {
			cur.token = "op"
			cur.value = "&&"
			i++
		} else if line[i] == '.' {
			cur.token = "val"
			e := i + 1
			for ; e < l; e++ {
				ch := line[e]
				if ((ch|0x20) < 'a' || (ch|0x20) > 'z') &&
					((ch-'0' < 0 || ch-'0' > 9) && e > i+1) &&
					ch != '.' && ch != '[' && ch != ']' {
					break
				}
			}
			val, err := getVar(line[i:e], vars)
			if err != nil {
				return elems, err
			}
			if val == "false" || val == "0" {
				val = ""
			}
			cur.value = val
			i = e - 1
		} else if line[i] == '"' {
			cur.token = "val"
			e := i + 1
			for ; e < l; e++ {
				if line[e] == '"' {
					break
				}
			}
			if e == l {
				return elems, parseErr("invalid string", line, cur)
			}
			cur.value = line[i+1 : e]
			i = e
		} else if l-i > 1 && line[i] == 'i' && line[i+1] == 'n' {
			cur.token = "in"
			cur.value = ""
			i++
		} else if line[i] == '[' {
			cur.token = "list-start"
		} else if line[i] == ']' {
			cur.token = "list-end"
		} else if line[i] == ',' {
			cur.token = ","
		} else {
			cur.token = "val"
			val := ""
			e := i
			for ; e < l; e++ {
				if line[e] == '\\' {
					e = e + 1
					if e == l {
						return elems, parseErr("EOF", line, cur)
					}
					val = val + string(line[e])
					continue
				}
				ch := line[e] | 0x20
				if ch >= 'a' && ch <= 'z' {
					val = val + string(line[e])
					continue
				}
				v := line[e] - '0'
				if v >= 0 && v <= 9 {
					val = val + string(line[e])
					continue
				}
				break
			}
			if i == e {
				return elems, parseErr("invalid character", line, cur)
			}
			cur.value = line[i:e]
			if cur.value == "false" || cur.value == "0" {
				cur.value = ""
			}
			i = e - 1
		}

	}

	return elems, nil
}

func parseConditionalParser(line string, elems []condElem) (bool, error) {

	idx := 1
	pass := 0
	again := false
	for len(elems) != 1 && pass < 3 {

		if idx >= len(elems) {
			idx = 1
			pass = pass + 1
			if again {
				pass = 0
				again = false
			}
			continue
		}

		prev := elems[idx-1]
		this := elems[idx]
		next := condElem{}
		if idx+1 < len(elems) {
			next = elems[idx+1]
		}

		if prev.token == "start" && this.token == "end" ||
			prev.token == "end" && this.token == "start" {
			return false, parseErr("misplaced parentheses", line, &this)
		}

		// reduce defines the number of elements that were reduced
		reduce := 0
		if prev.token == "unary" && this.token == "val" {
			val := ""
			if this.value == "" {
				val = "true"
			}
			this.value = val
			reduce = 1
		} else if prev.token == "start" && this.token == "val" && next.token == "end" {
			reduce = 2
		} else if prev.token == "val" && this.token == "op" && next.token == "val" {
			res := false
			switch {
			case pass >= 0 && this.value == "==":
				res = prev.value == next.value
				reduce = 2
			case pass >= 0 && this.value == "!=":
				res = prev.value != next.value
				reduce = 2
			case pass >= 1 && this.value == "&&":
				res = prev.value != "" && next.value != ""
				reduce = 2
			case pass >= 2 && this.value == "||":
				res = prev.value != "" || next.value != ""
				reduce = 2
			}

			if reduce > 0 {
				this.token = "val"
				this.value = ""
				if res {
					this.value = "true"
				}
			}
		} else if prev.token == "val" && this.token == "in" {
			if next.token != "list-start" {
				return false, parseErr("no list after 'in' found", line, &elems[0])
			}
			this.value = prev.value
			reduce = 2
		} else if prev.token == "in" || prev.token == "in-found" {
			if next.token != "list-end" && next.token != "," || this.token != "val" {
				return false, parseErr("malformed list", line, &elems[0])
			}

			if prev.token == "in" && prev.value == this.value {
				this.token = "in-found"
				this.value = "true"
			} else {
				this.token = prev.token
				this.value = prev.value
			}

			if next.token == "list-end" {
				this.token = "val"
			}
			reduce = 2
		}

		if reduce > 0 {
			elems = append(append(elems[:idx-1], this), elems[idx+reduce:]...)

			// optimization: continue and delay starting over
			again = true
		} else {
			idx++
		}
	}

	if len(elems) > 1 || elems[0].token != "val" {
		return false, parseErr(
			"failed to reduce condition (misplaced parentheses, multiple operands, etc.)",
			line, &elems[0])
	}
	return elems[0].value != "", nil
}

func parseConditional(line string, vars interface{}) (bool, error) {

	elems, err := parseConditionalLexer(line, vars)
	if err != nil {
		return false, err
	}
	if len(elems) == 0 {
		return false, errdefs.InvalidArgument("no condition provided")
	}

	return parseConditionalParser(line, elems)
}
