package cli

import (
	"strings"
	"testing"

	"bytes"
	"io"
	"os"
)

// compareString compares the provided strings and returns -1 if they match, or the position
// where they mismatch. Note that this will return the length of the shorter string if their
// length differs.
func compareStrings(l, r string) int {
	maxLen := len(l)
	if len(r) < maxLen {
		maxLen = len(r)
	}
	for i := 0; i < maxLen; i++ {
		if l[i] != r[i] {
			return i
		}
	}
	if len(l) != len(r) {
		return maxLen
	}
	return -1
}

// compareFuncOutput compares the output printed to stdout from the provided function with the
// provided expected string. It returns -1 if the strings match or the position where they
// mismatch, and also returns the generated string from the function.
func compareFuncOutput(printFunc func(), expected string) (int, string) {

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	printFunc()

	w.Sync()
	w.Close()
	os.Stdout = oldStdout // restoring the real stdout
	out := <-outC

	return compareStrings(out, expected), out
}

// TestPrintValueSimpleString tests printValue for a simple string value
func TestPrintValueSimpleString(t *testing.T) {

	testString := "TestString"
	const expected = "NAME    VALUE\nVarName TestString\n"

	errPos, out := compareFuncOutput(
		func() { printValue("NAME", "VALUE", "VarName", &testString) }, expected)
	if errPos != -1 {
		t.Errorf("Failed to print simple structure (pos %d)", errPos)
		t.Errorf("\n" + out)
	}
}

// TestPrintValueSimpleStruct tests printValue for a simple (non-nested) structure
func TestPrintValueSimpleStruct(t *testing.T) {

	testStruct := struct {
		FieldA   string
		FieldAB  string
		fieldABC string
	}{
		FieldA:   "ValueA",
		FieldAB:  "ValueAB",
		fieldABC: "ValueABC",
	}

	const expected = "NAME    VALUE\nFieldA  ValueA\nFieldAB ValueAB\n"

	errPos, out := compareFuncOutput(
		func() { printValue("NAME", "VALUE", "", &testStruct) }, expected)
	if errPos != -1 {
		t.Errorf("Failed to print simple structure (pos %d)", errPos)
		t.Errorf("\n" + out)
	}
}

// TestPrintValueSimpleStructWithPrefix tests printValue for a simple structure with
// a provided prefix string
func TestPrintValueSimpleStructWithPrefix(t *testing.T) {

	testStruct := struct {
		FieldA   string
		FieldAB  string
		fieldABC string
	}{
		FieldA:   "ValueA",
		FieldAB:  "ValueAB",
		fieldABC: "ValueABC",
	}

	const expected = "NAME           VALUE\nPrefix/FieldA  ValueA\nPrefix/FieldAB ValueAB\n"
	errPos, out := compareFuncOutput(
		func() { printValue("NAME", "VALUE", "Prefix", &testStruct) }, expected)
	if errPos != -1 {
		t.Errorf("Failed to print simple structure with prefix (pos %d)", errPos)
		t.Errorf("\n" + out)
	}

}

// TestPrintValueNestedStruct tests printValue for a nested structure
func TestPrintValueNestedStruct(t *testing.T) {

	type testSubStruct struct {
		FieldAA string
		FieldAB string
	}
	testStruct := struct {
		FieldA testSubStruct
	}{
		FieldA: testSubStruct{
			FieldAA: "ValueAA",
			FieldAB: "ValueAB",
		},
	}

	const expected = "NAME           VALUE\nFieldA/FieldAA ValueAA\nFieldA/FieldAB ValueAB\n"
	errPos, out := compareFuncOutput(
		func() { printValue("NAME", "VALUE", "", &testStruct) }, expected)
	if errPos != -1 {
		t.Errorf("Failed to print nested structure (pos %d)", errPos)
		t.Errorf("\n" + out)
	}
}

// TestPrintValueStructMap tests printValue for a map of a structure
func TestPrintValueStructMap(t *testing.T) {

	type testStruct struct {
		FieldA string
		FieldB string
	}

	testMap := map[string]testStruct{
		"KeyA": testStruct{
			FieldA: "ValueAA",
			FieldB: "ValueAB",
		},
		"KeyB": testStruct{
			FieldA: "ValueBA",
			FieldB: "ValueBB",
		},
	}

	const expected = `NAME               VALUE
Prefix/KeyA/FieldA ValueAA
Prefix/KeyA/FieldB ValueAB
Prefix/KeyB/FieldA ValueBA
Prefix/KeyB/FieldB ValueBB
`
	errPos, out := compareFuncOutput(
		func() { printValue("NAME", "VALUE", "Prefix", &testMap) }, expected)
	if errPos != -1 {
		t.Errorf("Failed to print map (pos %d)", errPos)
		t.Errorf("\n" + out)
	}
}

// TestPrintList tests printList for a slice of a simpel structure
func TestPrintValueList(t *testing.T) {

	type testStruct struct {
		FieldA   string
		FieldBB  string
		FieldCCC string
	}

	testList := []testStruct{
		testStruct{
			FieldA:   "ValueAA",
			FieldBB:  "ValueAB",
			FieldCCC: "ValueAC",
		},
		testStruct{
			FieldA:   "ValueBA",
			FieldBB:  "ValueBB",
			FieldCCC: "ValueBC",
		},
	}

	const expected = `FIELDA  FIELDBB FIELDCCC
ValueAA ValueAB ValueAC
ValueBA ValueBB ValueBC
`
	errPos, out := compareFuncOutput(
		func() { printList(testList) }, expected)
	if errPos != -1 {
		t.Errorf("Failed to print simple structure with prefix (pos %d)", errPos)
		t.Errorf("\n" + out)
	}
}

func TestPrintValueSlice(t *testing.T) {

	type testStruct struct {
		FieldA  string
		FieldBB string
	}

	testSlice := []testStruct{
		testStruct{
			FieldA:  "ValueAA",
			FieldBB: "ValueABB",
		},
		testStruct{
			FieldA:  "ValueBA",
			FieldBB: "ValueXYZ",
		},
	}

	const expected = `FIELD     VALUE
0/FieldA  ValueAA
0/FieldBB ValueABB
1/FieldA  ValueBA
1/FieldBB ValueXYZ
`
	errPos, out := compareFuncOutput(
		func() { printValue("Field", "Value", "", testSlice) }, expected)
	if errPos != -1 {
		t.Errorf("Failed to print simple structure with prefix (pos %d)", errPos)
		t.Errorf("\n" + out)
	}
}

// TODO: implement splitting a command line into slices of arguments
func compareCommands(t *testing.T, desc string, line string, exp [][]string) bool {

	res := scanLine(line)

	if len(res) != len(exp) {
		t.Errorf("Test '%s' failed: different length %d, should be %d %s",
			desc, len(res), len(exp), strings.Join(res[0], ":"))
		return false
	}
	for i := range res {
		if len(res[i]) != len(exp[i]) {
			t.Errorf("Test '%s' failed in line %d: number of arguments mismatch",
				desc, i)
		}
		for j := range res[i] {
			if res[i][j] != exp[i][j] {
				t.Errorf("Test '%s' failed in line %d, index %d: '%s' (exp: '%s')",
					desc, i, j, res[i][j], exp[i][j])
				return false
			}
		}
	}
	return true
}

func TestCliScanArgs(t *testing.T) {

	testLine := ""
	testCmds := [][]string{}
	compareCommands(t, "empty line", testLine, testCmds)

	testLine = "cmd1 arg11 arg12"
	testCmds = [][]string{{"cmd1 arg11 arg12"}}
	compareCommands(t, "single line", testLine, testCmds)

	testLine = " cmd1   arg11   arg12  "
	testCmds = [][]string{{"cmd1   arg11   arg12"}}
	compareCommands(t, "single line, extra spaces", testLine, testCmds)

	testLine = "cmd1 arg11, cmd2 arg21"
	testCmds = [][]string{{"cmd1 arg11"}, {"cmd2 arg21"}}
	compareCommands(t, "multi line, attached delim", testLine, testCmds)

	testLine = "cmd1 arg11 , cmd2 arg21"
	testCmds = [][]string{{"cmd1 arg11"}, {"cmd2 arg21"}}
	compareCommands(t, "multi line", testLine, testCmds)

	testLine = "cmd1 arg11 ,  ,,, cmd2 arg21"
	testCmds = [][]string{{"cmd1 arg11"}, {"cmd2 arg21"}}
	compareCommands(t, "multi line, multi delims", testLine, testCmds)
}
