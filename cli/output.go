package cli

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

// printValue prints the provided value as two columns for name and value content.
// Struct  Each element is printed as a single row with the provided prefix for the name field
//         For nested structures, the field names of each substructure are concatenated by '.'
// Map     Similar to Struct, but using the keys as the prefix instead of the structure elements.
// <other> Printed as two columns using the prefix as the name for the value content.
func printValueElem(w *tabwriter.Writer, prefix string, elem reflect.Value) {

	kind := elem.Kind()

	if prefix != "" && (kind == reflect.Struct || kind == reflect.Map) {
		prefix = prefix + "/"
	}

	if kind == reflect.Struct {
		elemType := elem.Type()
		for i := 0; i < elem.NumField(); i++ {
			printValueElem(w, prefix+elemType.Field(i).Name, elem.Field(i))
		}
	} else if kind == reflect.Map {
		iter := elem.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			printValueElem(w, prefix+k.String(), v)
		}
	} else if kind == reflect.Ptr {
		printValueElem(w, prefix, elem.Elem())
	} else if elem.CanInterface() {
		fmt.Fprintf(w, "%s\t%v\n", prefix, elem.Interface())
	}
}

// printValue prints the content of the provided value in two columns.
//  struct: field name, value
//  map:    key, value
//  slice:  index, value
//  <type>: prefix, value
func printValue(fieldHdr string, valueHdr string, prefix string, value interface{}) {

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 0, 1, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "%s\t%s\n", strings.ToUpper(fieldHdr), strings.ToUpper(valueHdr))
	printValueElem(w, prefix, reflect.ValueOf(value))
}

// printList prints a slice of structures using the field names as the header
func printList(list interface{}) {

	if reflect.TypeOf(list).Kind() != reflect.Slice {
		panic("provided argument must be of the type: slice")
	}

	if reflect.TypeOf(list).Elem().Kind() != reflect.Struct {
		panic("provided argument must be of the type: slice of structures")
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 0, 1, ' ', 0)
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
