package cli

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// sizeToSIString converts the provide integer value to a SI size string from the 10^3x exponent
func sizeToSIString(sz int64) string {
	const unit = 1000
	b := sz
	if b < 0 {
		b = -b
	}
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%cB", float64(sz)/float64(div), "kMGTPE"[exp])
}

// timeToAgoString converts the timespan from the provided time to the current time to a string
// in the formwat "T {year|month|hour}[s] ago". Future dates will return 'future'
func timeToAgoString(t time.Time) string {

	now := time.Now()
	if now.Before(t) {
		return "future"
	}

	diff := now.Sub(t)
	hours := diff.Hours()
	years := int(hours / 365 / 24)

	if years == 1 {
		return "one year ago"
	} else if years > 1 {
		return strconv.Itoa(years) + " years ago"
	}

	months := int(hours / 30.5)
	if months == 1 {
		return "one month ago"
	} else if months >= 1 {
		return strconv.Itoa(months) + " months ago"
	}

	if int(hours) == 1 {
		return "one hour ago"
	} else if hours > 1 {
		return strconv.Itoa(int(hours)) + " hours ago"
	}

	mins := diff.Minutes()
	if int(mins) == 1 {
		return "one minute ago"
	} else if mins > 1 {
		return strconv.Itoa(int(mins)) + " minutes ago"
	}

	return "seconds ago"
}

// printValue prints the provided value as two columns for name and value content.
// Struct  Each element is printed as a single row with the provided prefix for the name field
//         For nested structures, the field names of each substructure are concatenated by '.'
//         Use the output:"-" tag to omit a field.
// Map     Similar to Struct, but using the keys as the prefix instead of the structure elements.
// <other> Printed as two columns using the prefix as the name for the value content.
func printValueElem(w *tabwriter.Writer, prefix string, elem reflect.Value) {

	kind := elem.Kind()

	if prefix != "" && (kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice) {
		prefix = prefix + "/"
	}

	if kind == reflect.Struct {
		elemType := elem.Type()

		if elemType == reflect.TypeOf(time.Time{}) {
			t := elem.Interface().(time.Time).Format(time.RFC3339Nano)
			fmt.Fprintf(w, "%s\t%v\n", prefix, t)
			return
		}

		for i := 0; i < elem.NumField(); i++ {
			if !elem.Field(i).CanInterface() {
				break
			}
			if elemType.Field(i).Tag.Get("output") != "-" {
				printValueElem(w, prefix+elemType.Field(i).Name, elem.Field(i))
			}
		}
	} else if kind == reflect.Map {
		m := elem.MapKeys()
		keys := make([]string, len(m))
		for i := 0; i < len(m); i++ {
			keys[i] = m[i].String()
		}
		sort.Strings(keys)
		for _, k := range keys {
			printValueElem(w, prefix+k, elem.MapIndex(reflect.ValueOf(k)))
		}
	} else if kind == reflect.Slice {
		for i := 0; i < elem.Len(); i++ {
			printValueElem(w, prefix+strconv.Itoa(i), elem.Index(i))
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

	format := "%s"
	hdr := reflect.TypeOf(list).Elem()
	for i := 0; i < hdr.NumField(); i++ {
		if hdr.Field(i).Tag.Get("output") != "-" {
			fmt.Fprintf(w, format, strings.ToUpper(hdr.Field(i).Name))
			format = "\t%s"
		}
	}
	fmt.Fprintf(w, "\n")

	items := reflect.ValueOf(list)
	for i := 0; i < items.Len(); i++ {
		format = "%s"
		item := items.Index(i)
		for j := 0; j < item.NumField(); j++ {
			if hdr.Field(j).Tag.Get("output") == "-" {
				continue
			}
			fmt.Fprintf(w, format, item.Field(j).Interface())
			format = "\t%s"
		}
		fmt.Fprintf(w, "\n")
	}
}
