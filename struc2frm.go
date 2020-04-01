// Package struc2frm creates an HTML input form
// for a given struct type;
// see README.md for details.
package struc2frm

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode"
)

var defaultCSS = ""

func init() {
	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath := path.Join(path.Dir(filename), "default.css")
	bts, err := ioutil.ReadFile(sourceDirPath)
	if err != nil {
		log.Printf("Could not load default CSS: %v", err)
	}
	defaultCSS = "<style>\n" + string(bts) + "\n</style>"
}

type option struct {
	Key, Val string
}

type options []option

// s2FT contains formatting options for converting a struct into a HTML form
type s2FT struct {
	Indent        int                // horizontal width of the labels column
	IndentAddenum int                // for h3-headline and submit button, depends on CSS paddings and margins of div and input
	ShowSubmit    bool               // show submit, despite having only auto-changing selects
	ShowHeadline  bool               // headline derived from struct name
	Method        string             // form method - default is POST
	SelectOptions map[string]options // select inputs get their options from here

	InstanceID string // to distinguish several instances of a website

	CSS string // general formatting - provided defaults can be replaced
}

// New converter
func New() *s2FT {
	s2f := s2FT{
		Indent:        0,           // non-zero values override the CSS
		IndentAddenum: 2 * (4 + 4), // horizontal padding and margin

		ShowSubmit:    false,
		ShowHeadline:  false,
		SelectOptions: map[string]options{},
		Method:        "POST",
	}
	s2f.InstanceID = fmt.Sprint(time.Now().UnixNano())
	s2f.InstanceID = s2f.InstanceID[len(s2f.InstanceID)-8:] // last 8 digits

	// meta.embedded.block.CSS
	// meta.embedded.block.javascript
	// CSS

	s2f.CSS = defaultCSS

	return &s2f
}

var defaultS2F = New()

func (s2f *s2FT) RenderCSS(w io.Writer) {

	fmt.Fprint(w, s2f.CSS) // generic CSS

	if s2f.Indent == 0 { // using additional generic specs - for instance with media query
		return
	}

	// instance specific
	specific := `
<style>
	/* instance specifics */
	div.struc2frm-  label {
		min-width: %vpx;
	}
	div.struc2frm-  h3 {
		margin-left: %vpx;
	}
	div.struc2frm-  button[type=submit],
	div.struc2frm-  input[type=submit]
	{
		margin-left: %vpx;
	}
</style>
`
	specific = fmt.Sprintf(
		specific,
		s2f.Indent,
		s2f.Indent+s2f.IndentAddenum,
		s2f.Indent+s2f.IndentAddenum,
	)
	specific = strings.ReplaceAll(specific, "div.struc2frm-", fmt.Sprintf("div.struc2frm-%v", s2f.InstanceID))

	fmt.Fprint(w, specific)
}

// AddOptions is used by the caller to prepare option key-values
// for the rendering into HTML()
func (s2f *s2FT) AddOptions(name string, keys, values []string) {
	if s2f.SelectOptions == nil {
		s2f.SelectOptions = map[string]options{}
	}
	for i, key := range keys {
		s2f.SelectOptions[name] = append(s2f.SelectOptions[name], option{key, values[i]})
	}
}

// DefaultOptionKey gives the value to be selected on form init
func (s2f *s2FT) DefaultOptionKey(name string) string {
	if s2f.SelectOptions == nil {
		return ""
	}
	if len(s2f.SelectOptions[name]) == 0 {
		return ""
	}
	return s2f.SelectOptions[name][0].Key
}

// rendering <option val='...'>...</option> tags
func (opts options) HTML(selected string) string {
	w := &bytes.Buffer{}
	for _, o := range opts {
		if o.Key == selected {
			fmt.Fprintf(w, "\t<option value='%v' selected >%v</option>\n", o.Key, o.Val)
		} else {
			fmt.Fprintf(w, "\t<option value='%v'          >%v</option>\n", o.Key, o.Val)
		}
	}
	return w.String()
}

// golang type and 'form' struct tag 'subtype' => html input type
func toInputType(t, attrs string) string {

	switch t {
	case "string":
		switch structTag(attrs, "subtype") { // various possibilities - distinguish by subtype
		case "separator":
			return "separator"
		case "fieldset":
			return "fieldset"
		case "date":
			return "date"
		case "time":
			return "time"
		case "textarea":
			return "textarea"
		case "select":
			return "select"
		}
		return "text"
	case "int", "float64":
		return "number"
	case "bool":
		return "checkbox"
	case "[]uint8":
		return "file"
	}
	return "text"
}

// retrieving some special 'form' struct tag from the struct
func structTag(tags, key string) string {
	tagss := strings.Split(tags, ",")
	for _, a := range tagss {
		aLow := strings.ToLower(a)
		if strings.HasPrefix(aLow, key) {
			kv := strings.Split(a, "=")
			if len(kv) == 2 {
				return strings.Trim(kv[1], "'")
			}
		}
	}
	return ""
}

// convert all 'form' struct tags to  html input attributes
func structTagsToAttrs(tags string) string {
	tagss := strings.Split(tags, ",")
	ret := ""
	for _, t := range tagss {
		t = strings.TrimSpace(t)
		tl := strings.ToLower(t) // tag lower
		switch {
		case strings.HasPrefix(tl, "subtype"): // string - [date,textarea,select]
			ret += " " + t
		case strings.HasPrefix(tl, "size="): // visible width of input field
			ret += " " + t
		case strings.HasPrefix(tl, "maxlength="): // digits of input data
			ret += " " + t
		case strings.HasPrefix(tl, "max="): // for input number
			ret += " " + t
		case strings.HasPrefix(tl, "min="): // for input number
			ret += " " + t
		case strings.HasPrefix(tl, "step="): // for input number - special value 'any'
			ret += " " + t
		case strings.HasPrefix(tl, "pattern="): // client side validation; i.e. date layout [0-9\\.\\-/]{10}
			ret += " " + t
		case strings.HasPrefix(tl, "placeholder="): // a watermark showing expected input; i.e. 2006/01/02 15:04
			ret += " " + t
		case strings.HasPrefix(tl, "rows="): // for texarea
			ret += " " + t
		case strings.HasPrefix(tl, "cols="): // for texarea
			ret += " " + t
		case strings.HasPrefix(tl, "accept="): // file upload extension
			ret += " " + t
		case strings.HasPrefix(tl, "onchange"): // file upload extension
			ret += " " + "onchange='javascript:this.form.submit();'"
		case strings.HasPrefix(tl, "accesskey"): // goes into input, not into label
			ret += " " + t
		case strings.HasPrefix(tl, "title="): // mouse over tooltip - alt
			ret += " " + t
		default:
			// suffix    is not converted into an attribute
			// nobreak   is not converted into an attribute
		}

	}
	return ret
}

// for example 'Date layout' with accesskey 't' becomes 'Da<u>t</u>e layout'
func accessKeyify(s, attrs string) string {
	ak := structTag(attrs, "accesskey")
	if ak == "" {
		return s
	}
	akr := rune(ak[0])
	akrUp := unicode.ToUpper(akr)

	s2 := []rune{}
	found := false
	// log.Printf("-%s- -%s-", s, ak)
	for _, ru := range s {
		// log.Printf("\tcomparing %#U to %#U - %#U", ru, akr, akrUp)
		if (ru == akr || ru == akrUp) && !found {
			s2 = append(s2, '<', 'u', '>')
			s2 = append(s2, ru)
			s2 = append(s2, '<', '/', 'u', '>')
			found = true
			continue
		}
		s2 = append(s2, ru)
	}
	return string(s2)
}

// labelize converts struct field names and json field names
// to human readable format:
// bond_fund => Bond fund
// bondFund  => Bond fund
// bondFUND  => Bond fund
//
// notice rare edge case: BONDFund would be converted to 'BONDF und'
func labelize(s string) string {
	rs := make([]rune, 0, len(s))
	previousUpper := false
	for i, char := range s {
		if i == 0 {
			rs = append(rs, unicode.ToUpper(char))
			previousUpper = true
		} else {
			if char == '_' {
				char = ' '
			}
			if unicode.ToUpper(char) == char {
				if !previousUpper {
					rs = append(rs, ' ')
					rs = append(rs, unicode.ToLower(char))
				} else {
					rs = append(rs, unicode.ToLower(char))
				}
				previousUpper = true
			} else {
				rs = append(rs, char)
				previousUpper = false
			}
		}
	}
	return string(rs)
}

// ParseMultipartForm parses an HTTP request form
// with file attachments
func ParseMultipartForm(r *http.Request) error {

	if r.Method == "GET" {
		return nil
	}

	const _24K = (1 << 20) * 24
	err := r.ParseMultipartForm(_24K)
	if err != nil {
		log.Printf("Parse multipart form error: %v\n", err)
		return err
	}
	return nil
}

// ExtractUploadedFile extracts a file from an HTTP POST request.
// It needs the request form to be prepared with ParseMultipartForm.
func ExtractUploadedFile(r *http.Request, names ...string) (bts []byte, fname string, err error) {

	if r.Method == "GET" {
		return
	}

	name := "upload"
	if len(names) > 0 {
		name = names[0]
	}

	_, fheader, err := r.FormFile(name)
	if err != nil {
		log.Printf("Error unpacking upload bytes from post request: %v\n", err)
		return
	}

	fname = fheader.Filename
	log.Printf("Uploaded filename = %+v", fname)

	rdr, err := fheader.Open()
	if err != nil {
		log.Printf("Error opening uploaded file: %v\n", err)
		return
	}
	defer rdr.Close()

	bts, err = ioutil.ReadAll(rdr)
	if err != nil {
		log.Printf("Error reading uploaded file: %v\n", err)
		return
	}

	log.Printf("Extracted %v bytes from uploaded file", len(bts))
	return

}

// HTML takes a struct instance
// and turns it into an HTML form.
func (s2f *s2FT) HTML(intf interface{}) template.HTML {

	needSubmit := false // only select with onchange:submit() ?

	ifVal := reflect.ValueOf(intf)
	// ifVal = ifVal.Elem() // de reference
	if ifVal.Kind().String() != "struct" {
		return template.HTML(fmt.Sprintf("struct2form.HTML() - first arg must be struct - is %v", ifVal.Kind()))
	}

	w := &bytes.Buffer{}

	s2f.RenderCSS(w)

	// one class selector for general - one for specific instance
	fmt.Fprintf(w, "<div class='struc2frm struc2frm-%v'>\n", s2f.InstanceID)

	if s2f.ShowHeadline {
		fmt.Fprintf(w, "<h3>%v</h3>\n", labelize(ifVal.Type().Name()))
	}

	//
	uploadPostForm := false
	for i := 0; i < ifVal.NumField(); i++ {
		tp := ifVal.Field(i).Type().Name() // primitive type name: string, int
		if ifVal.Type().Field(i).Type.Kind() == reflect.Slice {
			tp = "[]" + ifVal.Type().Field(i).Type.Elem().Name()
		}
		if toInputType(tp, "") == "file" {
			uploadPostForm = true
			break
		}
	}

	if uploadPostForm {
		fmt.Fprint(w, "<form      method='post'   enctype='multipart/form-data'>\n")
	} else {
		fmt.Fprintf(w, "<form  method='%v' >\n", s2f.Method)
	}

	fieldsetOpen := false

	// Render fields
	for i := 0; i < ifVal.NumField(); i++ {

		fldName := ifVal.Type().Field(i).Name // i.e. Name, Birthdate

		if fldName[0:1] != strings.ToUpper(fldName[0:1]) {
			continue // skip unexported
		}

		inpName := ifVal.Type().Field(i).Tag.Get("json") // i.e. date_layout
		inpName = strings.Replace(inpName, ",omitempty", "", -1)
		frmLabel := labelize(inpName)

		attrs := ifVal.Type().Field(i).Tag.Get("form")

		if attrs == "-" {
			continue
		}

		val := ifVal.Field(i)
		tp := ifVal.Field(i).Type().Name() // primitive type name: string, int
		if ifVal.Type().Field(i).Type.Kind() == reflect.Slice {
			tp = "[]" + ifVal.Type().Field(i).Type.Elem().Name() // []byte => []uint8
		}

		// label
		specialVAlign := ""
		if toInputType(tp, attrs) == "textarea" {
			specialVAlign = "vertical-align: top;"
		}
		if toInputType(tp, attrs) != "separator" &&
			toInputType(tp, attrs) != "fieldset" {
			fmt.Fprintf(w,
				"<label for='%s' style='%v' >%v</label>", // no whitespace - input immediately afterwards
				inpName, specialVAlign, accessKeyify(frmLabel, attrs),
			)
		}

		// various inputs
		switch toInputType(tp, attrs) {
		case "checkbox":
			needSubmit = true
			checked := ""
			if val.Bool() {
				checked = "checked"

			}
			fmt.Fprintf(w, "<input type='%v' name='%v' id='%v' value='%v' %v %v />\n", toInputType(tp, attrs), inpName, inpName, "true", checked, structTagsToAttrs(attrs))
			fmt.Fprintf(w, "<input type='hidden' name='%v' value='false' />\n", inpName)
		case "file":
			needSubmit = true
			//              <input type="file" name="upload" id="upload" value="ignored.json" accept=".json" >
			fmt.Fprintf(w, "<input type='%v'   name='%v'     id='%v'     value='%v' %v />",
				toInputType(tp, attrs), inpName, inpName, "ignored.json", structTagsToAttrs(attrs),
			)
		case "date", "time":
			needSubmit = true
			//              <input type="date" name="myDate" max="1989-10-29"  min="2001-01-02">
			fmt.Fprintf(w, "<input type='%v'   name='%v'     id='%v'     value='%v' %v />",
				toInputType(tp, attrs), inpName, inpName, val, structTagsToAttrs(attrs),
			)
		case "textarea":
			needSubmit = true
			fmt.Fprintf(w, "<textarea name='%v' id='%v' %v />",
				inpName, inpName, structTagsToAttrs(attrs),
			)
			fmt.Fprint(w, val)
			fmt.Fprintf(w, "</textarea>")
		case "select":
			fmt.Fprintf(w, "<select name='%v' id='%v' %v />\n", inpName, inpName, structTagsToAttrs(attrs))
			fmt.Fprint(w, s2f.SelectOptions[inpName].HTML(val.String()))
			fmt.Fprint(w, "</select>")
		case "separator":
			fmt.Fprint(w, "<div class='separator'></div>")
		case "fieldset":
			if fieldsetOpen {
				fmt.Fprint(w, "</fieldset>")
			}
			fmt.Fprint(w, "<fieldset>")
			fmt.Fprintf(w, "\t<legend>&nbsp;%v&nbsp;</legend>", frmLabel)
			fieldsetOpen = true
		default:
			// plain vanilla input
			needSubmit = true
			fmt.Fprintf(w, "<input type='%v' name='%v' id='%v' value='%v' %v />", toInputType(tp, attrs), inpName, inpName, val, structTagsToAttrs(attrs))

		}

		sfx := structTag(attrs, "suffix")
		if sfx != "" {
			fmt.Fprintf(w, "<span class='suffix' >%s</span>", sfx)
		}

		if toInputType(tp, attrs) != "separator" &&
			toInputType(tp, attrs) != "fieldset" &&
			structTag(attrs, "nobreak") == "" {
			fmt.Fprintf(w, "<br>")
		}
		fmt.Fprintf(w, "\n")

	}

	if fieldsetOpen {
		fmt.Fprint(w, "</fieldset>")
	}

	if needSubmit || s2f.ShowSubmit {
		// name should *not* be 'submit'
		// avoiding error on this.form.submit()
		// 'submit is not a function' stackoverflow.com/questions/833032/
		fmt.Fprintf(w, "<button  type='submit' name='btnSubmit' value='1' accesskey='s'  ><b>S</b>ubmit</button><br>\n")
	} else {
		fmt.Fprintf(w, "<input   type='hidden' name='btnSubmit' value='1'\n")
	}

	fmt.Fprint(w, "</form>\n")
	fmt.Fprint(w, "</div>\n")

	return template.HTML(w.String())
}

// HTML takes a struct instance
// and uses the default formatter
// to turns it into an HTML form.
func HTML(intf interface{}) template.HTML {
	return defaultS2F.HTML(intf)
}
