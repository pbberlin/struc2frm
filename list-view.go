package struc2frm

import (
	"bytes"
	"fmt"
	"html/template"
	"reflect"
	"strings"
)

// Completable structs can be rendered as HTML list
type Completable interface {
	Complete() bool
}

// List creates an HTML list view - instead of an HTML form
func (s2f *s2FT) List(intf Completable) template.HTML {

	v := reflect.ValueOf(intf)
	typeOfS := v.Type()

	labels := make([]string, 0, 10)
	values := make([]string, 0, 10)
	statusMsg := ""

	for i := 0; i < v.NumField(); i++ {
		fn := typeOfS.Field(i).Name
		if strings.HasPrefix(fn, "Separator") {
			continue
		}
		if fn == "Status" || fn == "Msg" {
			val := v.Field(i).Interface()
			if valStr, ok := val.(string); ok {
				if statusMsg != "" {
					statusMsg += " - "
				}
				statusMsg += valStr
			}
			continue
		}

		val := v.Field(i).Interface()
		if val == "" {
			continue
		}

		labels = append(labels, fn)
		if valStr, ok := val.(string); ok {
			values = append(values, strings.Title(valStr))
		} else if valBool, ok := val.(bool); ok {
			values = append(values, fmt.Sprintf("%v", valBool))
		}
	}

	w := &bytes.Buffer{}

	fmt.Fprintf(w, "<ul>\n")

	if intf.Complete() {
		for idx, label := range labels {
			fmt.Fprintf(w, "\t<li>\n")
			fmt.Fprintf(w, "\t<span style='display: inline-block; width: 40%%;' >%v:</span>", label)
			fmt.Fprintf(w, "  %v  \n", values[idx])
			fmt.Fprintf(w, "\t</li>\n")
		}
	} else {
		fmt.Fprintf(w, "\t<li>\n")
		fmt.Fprintf(w, "\t  Struct is incomplete: %v\n", statusMsg)
		fmt.Fprintf(w, "\t</li>\n")
	}

	fmt.Fprintf(w, "</ul>\n")

	return template.HTML(w.String())
}
