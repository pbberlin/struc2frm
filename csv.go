package struc2frm

import (
	"fmt"
	"reflect"
	"strings"
)

// CSVLine renders intf to CSV format
func (s2f *s2FT) CSVLine(intf interface{}) string {

	v := reflect.ValueOf(intf) // ifVal
	typeOfS := v.Type()
	// v = v.Elem()            // dereference

	if v.Kind().String() != "struct" {
		return fmt.Sprintf("struct2form.Card() - arg1 must be struct - is %v", v.Kind())
	}

	values := make([]string, 0, v.NumField())

	for i := 0; i < v.NumField(); i++ {

		fn := typeOfS.Field(i).Name // i.e. Name, Birthdate
		if fn[0:1] != strings.ToUpper(fn[0:1]) {
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

	valid := true // default
	errs := map[string]string{}
	if vldr, ok := intf.(Validator); ok { // if validator interface is implemented...
		errs, valid = vldr.Validate() // ...check for validity
	}
	if valid {
		for idx := range values {
			fmt.Fprintf(w, "%v;", values[idx])
		}
		fmt.Fprintf(w, "\n")
	} else {
		fmt.Fprintf(w, "Struct content is invalid: ")
		for fld, msg := range errs {
			fmt.Fprintf(w, "\t  Field: %v - %v", fld, msg)
		}
		fmt.Fprintf(w, "\n")
	}
	return w.String()
}
