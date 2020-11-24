package struc2frm

import (
	"bytes"
	"fmt"
	"html/template"
	"reflect"
	"strings"
)

// Card creates an HTML list view - instead of an HTML form;
// TODO: render fieldsets
func (s2f *s2FT) Card(intf interface{}) template.HTML {

	v := reflect.ValueOf(intf) // ifVal
	typeOfS := v.Type()
	// v = v.Elem()            // dereference

	if v.Kind().String() != "struct" {
		return template.HTML(fmt.Sprintf("struct2form.Card() - arg1 must be struct - is %v", v.Kind()))
	}

	labels := make([]string, 0, v.NumField())
	values := make([]string, 0, v.NumField())
	sfxs := make([]string, 0, v.NumField())
	statusMsg := ""

	for i := 0; i < v.NumField(); i++ {

		fn := typeOfS.Field(i).Name // i.e. Name, Birthdate

		if fn[0:1] != strings.ToUpper(fn[0:1]) {
			continue // skip unexported
		}

		inpName := typeOfS.Field(i).Tag.Get("json") // i.e. date_layout
		inpName = strings.Replace(inpName, ",omitempty", "", -1)
		frmLabel := labelize(inpName)

		attrs := typeOfS.Field(i).Tag.Get("form") // i.e. form:"maxlength='42',size='28',suffix='optional'"

		if structTag(attrs, "label") != "" {
			frmLabel = structTag(attrs, "label")
		}

		if strings.Contains(attrs, ", ") || strings.Contains(attrs, ", ") {
			return template.HTML(fmt.Sprintf("struct2form.Card() - field %v: tag 'form' cannot contain ', ' or ' ,' ", fn))
		}

		if commaInsideQuotes(attrs) {
			return template.HTML(fmt.Sprintf("struct2form.Card() - field %v: tag 'form' - use &comma; instead of ',' inside of single quotes values", fn))
		}

		if attrs == "-" {
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

		if s2f.SkipEmpty {
			if val == "" && !strings.HasPrefix(fn, "Separator") {
				continue
			}
		}

		labels = append(labels, frmLabel)
		if valBool, ok := val.(bool); ok {
			values = append(values, fmt.Sprintf("%v", valBool))
		} else {
			values = append(values, fmt.Sprintf("%v", val)) // covers string, float, int ...
		}

		// Replace <select...> keys with values
		idx := len(values) - 1
		if values[idx] != "" {
			if opts, ok := s2f.selectOptions[inpName]; ok {
				for _, opt := range opts {
					// log.Printf("For %12v: Comparing %5v to %5v  %5v", inpName, values[idx], opt.Key, opt.Val)
					if values[idx] == opt.Key {
						values[idx] = opt.Val
					}
				}
			}
		}

		sfx := structTag(attrs, "suffix")
		sfxs = append(sfxs, sfx)

	}

	w := &bytes.Buffer{}

	s2f.RenderCSS(w)

	// one class selector for general - one for specific instance
	fmt.Fprintf(w, "<div class='struc2frm struc2frm-%v'>\n", s2f.InstanceID)

	if s2f.ShowHeadline {
		fmt.Fprintf(w, "<h3>%v</h3>\n", labelize(typeOfS.Name()))
	}

	fmt.Fprintf(w, "<ul>\n")

	// fieldsetOpen := false

	valid := true // default
	errs := map[string]string{}
	if vldr, ok := intf.(Validator); ok { // if validator interface is implemented...
		errs, valid = vldr.Validate() // ...check for validity
	}
	if valid {
		for idx, label := range labels {
			if strings.HasPrefix(label, "Separator") {
				fmt.Fprint(w, "\t<div class='separator'></div>\n")
				continue
			}
			fmt.Fprintf(w, "\t<li>\n")
			fmt.Fprintf(w, "\t<span style='display: inline-block; width: 40%%;' >%v:</span>", label)
			fmt.Fprintf(w, "  %v  \n", values[idx])
			if sfxs[idx] != "" {
				fmt.Fprintf(w, "<span class='postlabel' >%s</span>", sfxs[idx])
			}
			fmt.Fprintf(w, "\t</li>\n")
		}
	} else {
		fmt.Fprintf(w, "\t<li>\n")
		fmt.Fprintf(w, "\t  Struct content is invalid: %v\n", statusMsg)
		for fld, msg := range errs {
			fmt.Fprintf(w, "\t  Field: %v - %v\n", fld, msg)
		}
		fmt.Fprintf(w, "\t</li>\n")
	}

	fmt.Fprintf(w, "</ul>\n")

	fmt.Fprint(w, "</div><!-- </div class='struc2frm'... -->\n")

	// global replacements
	ret := strings.ReplaceAll(w.String(), "&comma;", ",")

	return template.HTML(ret)
}
