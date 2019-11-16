package cli

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

func printStructElem(w *tabwriter.Writer, name string, field reflect.Value) {
	if field.Kind() == reflect.Struct {
		fieldType := field.Type()
		for i := 0; i < field.NumField(); i++ {
			printStructElem(w, name+"."+fieldType.Field(i).Name, field.Field(i))
		}
	} else {
		fmt.Fprintf(w, "%s\t%s\n", name, field.Interface())
	}
}

// printStruct prints the field and value of all elements of a structure using the provided
// header names. It supports only 2-level structures
func printStruct(fieldHdr string, valueHdr string, items interface{}) {

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	fmt.Fprintf(w, "%s\t%s\n", strings.ToUpper(fieldHdr), strings.ToUpper(valueHdr))

	elem := reflect.ValueOf(items).Elem()
	elemType := elem.Type()
	for i := 0; i < elem.NumField(); i++ {
		printStructElem(w, elemType.Field(i).Name, elem.Field(i))
	}
}

// printList prints a list (slice of structures) using the field names as the header
func printList(list interface{}) {

	if reflect.TypeOf(list).Kind() != reflect.Slice {
		panic("provided argument must be of the type: slice")
	}

	if reflect.TypeOf(list).Elem().Kind() != reflect.Struct {
		panic("provided argument must be of the type: slice of structures")
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	hdr := reflect.TypeOf(list).Elem()
	fmt.Fprintf(w, "%s", strings.ToUpper(hdr.Field(0).Name))
	for i := 1; i < hdr.NumField(); i++ {
		fmt.Fprintf(w, "\t%s", strings.ToUpper(hdr.Field(i).Name))
	}
	fmt.Fprintf(w, "\n")

	items := reflect.ValueOf(list)
	for i := 0; i < items.Len(); i++ {
		item := items.Index(i)
		fmt.Fprintf(w, "%s", item.Field(0).Interface())
		for j := 1; j < item.NumField(); j++ {
			fmt.Fprintf(w, "\t%s", item.Field(j).Interface())
		}
		fmt.Fprintf(w, "\n")
	}
}
