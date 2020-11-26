package struc2frm

import (
	"fmt"
	"reflect"
	"strings"
)

// CSVLine renders intf into a line of CSV formatted data; not double quotes
func (s2f *s2FT) CSVLine(intf interface{}, sep string) string {

	v := reflect.ValueOf(intf) // ifVal
	typeOfS := v.Type()
	// v = v.Elem()            // dereference

	if v.Kind().String() != "struct" {
		return fmt.Sprintf("struct2form.CSVLine() - arg1 must be struct - is %v", v.Kind())
	}

	values := make([]string, 0, v.NumField())

	for i := 0; i < v.NumField(); i++ {

		// struct field name; i.e. Name, Birthdate
		fn := typeOfS.Field(i).Name
		if fn[0:1] != strings.ToUpper(fn[0:1]) { // only used to find unexported fields; otherwise json tag name is used
			continue // skip unexported
		}
		if strings.HasPrefix(fn, "Separator") {
			continue
		}

		val := v.Field(i).Interface()
		if valBool, ok := val.(bool); ok {
			values = append(values, fmt.Sprintf("%v", valBool))
		} else {
			values = append(values, fmt.Sprintf("%v", val)) // covers string, float, int ...
		}
		// not implemented: replacing <select...> keys with values
	}

	w := &strings.Builder{}
	for idx := range values {
		fmt.Fprintf(w, "%v%v", values[idx], sep)
	}

	valid := true // default
	errs := map[string]string{}
	if vldr, ok := intf.(Validator); ok { // if validator interface is implemented...
		errs, valid = vldr.Validate() // ...check for validity
	}
	if !valid {
		fmt.Fprintf(w, "struct content is invalid, ")
		for fld, msg := range errs {
			fmt.Fprintf(w, "field '%v' has error '%v', ", fld, msg)
		}
	}

	fmt.Fprintf(w, "\n")
	return w.String()
}
