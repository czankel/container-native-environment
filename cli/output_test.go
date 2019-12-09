package cli

import (
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
	if len(l) != len(r) {
		return maxLen
	}
	for i := 0; i < maxLen; i++ {
		if l[i] != r[i] {
			return i
		}
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

	const expected = "NAME           VALUE\nPrefix.FieldA  ValueA\nPrefix.FieldAB ValueAB\n"
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

	const expected = "NAME           VALUE\nFieldA.FieldAA ValueAA\nFieldA.FieldAB ValueAB\n"
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
Prefix.KeyA.FieldA ValueAA
Prefix.KeyA.FieldB ValueAB
Prefix.KeyB.FieldA ValueBA
Prefix.KeyB.FieldB ValueBB
`
	errPos, out := compareFuncOutput(
		func() { printValue("NAME", "VALUE", "Prefix", &testMap) }, expected)
	if errPos != -1 {
		t.Errorf("Failed to print simple structure with prefix (pos %d)", errPos)
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
