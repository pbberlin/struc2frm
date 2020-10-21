// Package struc2frm creates an HTML input form
// for a given struct type;
// see README.md for details.
package struc2frm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/go-playground/form"
	"github.com/pkg/errors"
)

var defaultHTML = ""

func init() {
	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath := path.Join(path.Dir(filename), "tpl-main.html")
	bts, err := ioutil.ReadFile(sourceDirPath)
	if err != nil {
		log.Printf("Could not load main template: %v", err)
		defaultHTML = staticTplMainHTML
		log.Printf("Loaded %v chars from static.go instead", len(staticTplMainHTML))
		return
	}
	defaultHTML = string(bts)
}

var defaultCSS = ""

func init() {
	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath := path.Join(path.Dir(filename), "default.css")
	bts, err := ioutil.ReadFile(sourceDirPath)
	if err != nil {
		log.Printf("Could not load default CSS: %v", err)
		defaultCSS = "<style>\n" + staticDefaultCSS + "\n</style>"
		log.Printf("Loaded %v chars from static.go instead", len(staticDefaultCSS))
		return
	}
	defaultCSS = "<style>\n" + string(bts) + "\n</style>"
}

type option struct {
	Key, Val string
}

type options []option

// s2FT contains formatting options for converting a struct into a HTML form
type s2FT struct {
	ShowHeadline bool   // show headline derived from struct name
	FormTag      bool   // include <form...> and </form>
	Name         string // form name; default 'frmMain'; if distinct names are needed, application may change the values
	Method       string // form method - default is POST
	InstanceID   string // to distinguish several instances on same website

	Salt        string // generated from MAC address - see below
	FormTimeout int    // hours until a form post is rejected - CSRF token

	FocusFirstError bool // setfocus(); takes precedence over focus attribute
	ForceSubmit     bool // show submit, despite having only auto-changing selects

	Indent         int     // horizontal width of the labels column
	IndentAddenum  int     // for h3-headline and submit button, depends on CSS paddings and margins of div and input
	VerticalSpacer float64 // in CSS REM

	CSS string // general formatting - provided defaults can be replaced

	// Card View options
	SkipEmpty bool // Fields with value "" are not rendered

	selectOptions map[string]options // select inputs get their options from here
	errors        map[string]string  // validation errors by json name of input

}

var addressMAC = ""

func init() {
	// MAC address as salt
	// run only once at init() time to save time
	ifs, _ := net.Interfaces()
	for _, v := range ifs {
		h := v.HardwareAddr.String()
		if len(h) == 0 {
			continue
		}
		addressMAC = h
		break
	}
}

// New converter
func New() *s2FT {
	s2f := s2FT{
		ShowHeadline: false,
		FormTag:      true,
		// Name - see below
		Method: "POST",

		Salt:        addressMAC,
		FormTimeout: 2,

		selectOptions: map[string]options{},
		errors:        map[string]string{},

		FocusFirstError: true,
		ForceSubmit:     false,

		Indent:         0,           // non-zero values override the CSS
		IndentAddenum:  2 * (4 + 4), // horizontal padding and margin
		VerticalSpacer: 0.6,

		CSS: defaultCSS,
	}
	s2f.InstanceID = fmt.Sprint(time.Now().UnixNano())
	s2f.InstanceID = s2f.InstanceID[len(s2f.InstanceID)-8:] // use the last 8 digits
	s2f.Name = fmt.Sprintf("frmMain_%s", s2f.InstanceID)
	s2f.Name = "frmMain" // form name can be changed by application

	return &s2f
}

// CloneForRequest takes a package instance of s2FT
// and clones it for safe usage in parallel http requests
func (s2f *s2FT) CloneForRequest() *s2FT {
	clone := *s2f
	clone.errors = map[string]string{}
	clone.InstanceID = fmt.Sprint(time.Now().UnixNano())
	clone.InstanceID = clone.InstanceID[len(clone.InstanceID)-8:] // last 8 digits
	return &clone
}

var defaultS2F = New()

/*
	.+      any chars
	'       boundary one
	(.+?)   any chars, ? means not greedy
	'       boundary two
	.+      any chars

	this leads nowhere, since opening and closing boundary are the same
*/
var comma = regexp.MustCompile(`.*'(.*,.*?)'.*`)

func commaInsideQuotesBAD(s string) bool {
	matches := comma.FindAllStringSubmatch(s, -1) // -1 returns all matches
	log.Printf("%v matches for \n\t%v\n\t%+v\n\n", len(matches), s, matches)
	return comma.MatchString(s)
}

func commaInsideQuotes(s string) bool {
	parts := strings.Split(s, "'")
	for idx, pt := range parts {
		if idx%2 == 0 {
			continue
		}
		// log.Printf("checking part %v - %v", idx, pt)
		if strings.Contains(pt, ",") {
			return true
		}
	}
	return false
}

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

func (s2f *s2FT) verticalSpacer() string {
	return fmt.Sprintf("\t<div style='height:%3.1frem'>&nbsp;</div>", s2f.VerticalSpacer)
}

// SetOptions to prepare dropdown/select options - with keys and labels
// for rendering in Form()
func (s2f *s2FT) SetOptions(nameJSON string, keys, labels []string) {
	if s2f.selectOptions == nil {
		s2f.selectOptions = map[string]options{}
	}
	s2f.selectOptions[nameJSON] = options{} // always reset options to prevent accumulation of options on clones

	if len(keys) != len(labels) {
		s2f.selectOptions[nameJSON] = append(s2f.selectOptions[nameJSON], option{"key", "keys and labels length does not match"})
	} else {
		for i, key := range keys {
			s2f.selectOptions[nameJSON] = append(s2f.selectOptions[nameJSON], option{key, labels[i]})
		}
	}
}

// AddError adds a validation message;
// key 'global' writes msg on top of form.
func (s2f *s2FT) AddError(nameJSON string, msg string) {
	if s2f.errors == nil {
		s2f.errors = map[string]string{}
	}
	if _, ok := s2f.errors[nameJSON]; ok {
		s2f.errors[nameJSON] += "<br>\n"
	}
	s2f.errors[nameJSON] += msg
}

// AddErrors adds validation messages;
// key 'global' writes msg on top of form.
func (s2f *s2FT) AddErrors(errs map[string]string) {
	if s2f.errors == nil {
		s2f.errors = map[string]string{}
	}
	for nameJSON, msg := range errs {
		if _, ok := s2f.errors[nameJSON]; ok {
			s2f.errors[nameJSON] += "<br>\n"
		}
		s2f.errors[nameJSON] += msg
	}
}

// DefaultOptionKey gives the value to be selected on form init
func (s2f *s2FT) DefaultOptionKey(name string) string {
	if s2f.selectOptions == nil {
		return ""
	}
	if len(s2f.selectOptions[name]) == 0 {
		return ""
	}
	return s2f.selectOptions[name][0].Key
}

// rendering <option val='...'>...</option> tags
func (opts options) HTML(selecteds []string) string {
	w := &bytes.Buffer{}
	// log.Printf("select options - selecteds %v", selecteds)
	for _, o := range opts {
		found := false
		for _, selected := range selecteds {
			if o.Key == selected {
				found = true
				// log.Printf("found %v", o.Key)
			}
		}
		if found {
			fmt.Fprintf(w, "\t\t<option value='%v' selected >%v</option>\n", o.Key, o.Val)
		} else {
			fmt.Fprintf(w, "\t\t<option value='%v'          >%v</option>\n", o.Key, o.Val)
		}
	}
	return w.String()
}

/*ValToString converts reflect.Value to string.

go-playground/form.Decode nicely converts all kins of request.Form strings
into the desired struct types.

But to render the form to HTML, we have to convert those types back to string.

	val.String() of a   bool yields "<bool Value>"
	val.String() of an   int yields "<int Value>"
	val.String() of a  float yields "<float64 Value>"
*/
func ValToString(val reflect.Value) string {

	tp := val.Kind()

	valStr := val.String() // trivial case
	if tp == reflect.Bool {
		valStr = fmt.Sprint(val.Bool())
	} else if tp == reflect.Int {
		valStr = fmt.Sprint(val.Int())
	} else if tp == reflect.Float64 {
		valStr = fmt.Sprint(val.Float())
	}

	return valStr

}

// golang type and 'form' struct tag 'subtype' => html input type
func toInputType(t, attrs string) string {

	switch t {
	case "string", "[]string":
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
	case "int", "float64", "[]int", "[]float64":
		switch structTag(attrs, "subtype") { // might want dropdown, for instance for list of years
		case "select":
			return "select"
		}
		return "number"
	case "bool", "[]bool":
		switch structTag(attrs, "subtype") { // not always checkbox, but sometimes dropdown
		case "select":
			return "select"
		}
		return "checkbox"
	case "[]uint8":
		return "file"
	}
	return "text"
}

// parsing the struct tag 'form';
// returning a *single* value for argument key;
// i.e. "maxlength='42',size='28',suffix='optional'"
//       key=size
//       returns 28
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

// convert the struct tag 'form' to html input attributes;
// mostly replacing comma with single space;
// i.e. "maxlength='42',size='28',suffix='optional'"
func structTagsToAttrs(tags string) string {
	tagss := strings.Split(tags, ",")
	ret := ""
	for _, t := range tagss {
		t = strings.TrimSpace(t)
		tl := strings.ToLower(t) // tag lower
		switch {
		case strings.HasPrefix(tl, "subtype="): // string - [date,textarea,select]
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
		case strings.HasPrefix(tl, "wildcardselect"): // show extra input next to select - to select options
			ret += " " + t
		case strings.HasPrefix(tl, "accesskey="): // goes into input, not into label
			ret += " " + t
		case strings.HasPrefix(tl, "title="): // mouse over tooltip - alt
			ret += " " + t
		case strings.HasPrefix(tl, "autocapitalize="): // 'off' prevents upper case for first word on mobile phones
			ret += " " + t
		case strings.HasPrefix(tl, "inputmode="): // 'numeric' shows only numbers keysboard on mobile phones
			ret += " " + t
		case strings.HasPrefix(tl, "multiple"): // dropdown/select - select multiple items; no value
			ret += " " + "multiple" // only the attribute; no value
		case strings.HasPrefix(tl, "autofocus"):
			ret += " " + "autofocus" // only the attribute; no value
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
// edge case: BONDFund would be converted to 'Bondfund'
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
				if !previousUpper && char != ' ' {
					rs = append(rs, ' ')
				}
				rs = append(rs, unicode.ToLower(char))
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

// Form takes a struct instance
// and turns it into an Form form.
func (s2f *s2FT) Form(intf interface{}) template.HTML {

	v := reflect.ValueOf(intf) // interface val
	typeOfS := v.Type()
	// v = v.Elem() // de reference

	if v.Kind().String() != "struct" {
		return template.HTML(fmt.Sprintf("struct2form.HTML() - arg1 must be struct - is %v", v.Kind()))
	}

	w := &bytes.Buffer{}

	needSubmit := false // only select with onchange:submit() ?

	// collect fields with initial focus and fields with errors
	inputWithFocus := ""      // first input having an autofocus attribute
	firstInputWithError := "" // first input having an error message
	if s2f.FocusFirstError {
		for i := 0; i < v.NumField(); i++ {
			inpName := typeOfS.Field(i).Tag.Get("json") // i.e. date_layout
			inpName = strings.Replace(inpName, ",omitempty", "", -1)
			_, hasError := s2f.errors[inpName]
			if hasError {
				firstInputWithError = inpName
				break
			}
		}
	}
	// error focus takes precedence over init focus
	if firstInputWithError != "" {
		inputWithFocus = firstInputWithError
	} else {
		for i := 0; i < v.NumField(); i++ {
			inpName := typeOfS.Field(i).Tag.Get("json") // i.e. date_layout
			inpName = strings.Replace(inpName, ",omitempty", "", -1)
			attrs := typeOfS.Field(i).Tag.Get("form") // i.e. form:"maxlength='42',size='28'"
			if structTag(attrs, "autofocus") != "" {
				inputWithFocus = inpName
			}
		}
	}

	s2f.RenderCSS(w)

	// one class selector for general - one for specific instance
	fmt.Fprintf(w, "<div class='struc2frm struc2frm-%v'>\n", s2f.InstanceID)

	if s2f.ShowHeadline {
		fmt.Fprintf(w, "<h3>%v</h3>\n", labelize(typeOfS.Name()))
	}

	// file upload requires distinct form attribute
	uploadPostForm := false
	for i := 0; i < v.NumField(); i++ {
		tp := v.Field(i).Type().Name() // primitive type name: string, int
		if typeOfS.Field(i).Type.Kind() == reflect.Slice {
			tp = "[]" + typeOfS.Field(i).Type.Elem().Name()
		}
		if toInputType(tp, "") == "file" {
			uploadPostForm = true
			break
		}
	}

	if s2f.FormTag {
		if uploadPostForm {
			fmt.Fprintf(w, "<form  name='%v'  method='post'   enctype='multipart/form-data'>\n", s2f.Name)
		} else {
			// browser default encoding for post is "application/x-www-form-urlencoded"
			fmt.Fprintf(w, "<form name='%v'  method='%v' >\n", s2f.Name, s2f.Method)
		}
	}

	if errMsg, ok := s2f.errors["global"]; ok {
		fmt.Fprintf(w, "\t<p class='error-block' >%v</p>\n", errMsg)
	}

	fmt.Fprintf(w, "\t<input name='token'    type='hidden'   value='%v' />\n", s2f.FormToken())

	fieldsetOpen := false

	// Render fields
	for i := 0; i < v.NumField(); i++ {

		fn := typeOfS.Field(i).Name // i.e. Name, Birthdate

		if fn[0:1] != strings.ToUpper(fn[0:1]) {
			continue // skip unexported
		}

		inpName := typeOfS.Field(i).Tag.Get("json") // i.e. date_layout
		inpName = strings.Replace(inpName, ",omitempty", "", -1)
		frmLabel := labelize(inpName)

		attrs := typeOfS.Field(i).Tag.Get("form") // i.e. form:"maxlength='42',size='28'"

		if structTag(attrs, "label") != "" {
			frmLabel = structTag(attrs, "label")
		}

		if strings.Contains(attrs, ", ") || strings.Contains(attrs, ", ") {
			return template.HTML(fmt.Sprintf("struct2form.HTML() - field %v: tag 'form' cannot contain ', ' or ' ,' ", fn))
		}

		if commaInsideQuotes(attrs) {
			return template.HTML(fmt.Sprintf("struct2form.HTML() - field %v: tag 'form' - use &comma; instead of ',' inside of single quotes values", fn))
		}

		if attrs == "-" {
			continue
		}

		// getting the value and the type of the iterated struct field
		val := v.Field(i)
		if false {
			// if our entry form struct would contain pointer fields...
			val = reflect.Indirect(val) // pointer converted to value
			val = reflect.Indirect(val) // idempotent
			val = val.Elem()            // what is the difference?
		}

		tp := v.Field(i).Type().Name() // primitive type name: string, int
		if typeOfS.Field(i).Type.Kind() == reflect.Slice {
			tp = "[]" + v.Type().Field(i).Type.Elem().Name() // []byte => []uint8
		}

		valStr := ValToString(val)
		valStrs := []string{valStr} // for select multiple='false'

		// for select multiple='true'
		// 		if tp == []string or []int or []float64 ...
		// 		unpack slice from checkbox arrays or select/dropdown multiple
		if typeOfS.Field(i).Type.Kind() == reflect.Slice {

			// valSlice := reflect.MakeSlice(val.Type(), val.Cap(), val.Len())
			// valSlice := val.Slice(0, val.Len())
			valSlice := val // same as above

			// log.Printf(
			// 	"kind of type %v - elem type %v - capacity %v - length %v - %v",
			// 	valSlice.Kind(), val.Type(), valSlice.Cap(), valSlice.Len(), valSlice,
			// )

			valStrs = []string{} // reset
			for i := 0; i < valSlice.Len(); i++ {
				// see package fmt/print.go - printValue()::865
				// vx := valSlice.Slice(i, i+1) // this woudl be a subslice
				valElem := valSlice.Index(i) // this is an element of a slice
				// log.Printf("Elem is %v", ValToString(valElem))
				valStrs = append(valStrs, ValToString(valElem))
			}

			// select multiple:
			// having extracted valStrs, we want prevent additive request into form parsing;
			// but CanSet() yields false;
			// better setting init values *conditionally* *after* parsing
			if valSlice.CanSet() && false {
				log.Printf("%v -  %v - %v - trying slice reset", fn, tp, val.Type())
				valueSlice := reflect.MakeSlice(val.Type(), 0, 5)
				valueSlice.Set(valueSlice)
			}
		}

		errMsg, hasError := s2f.errors[inpName]
		if hasError {
			fmt.Fprintf(w, "\t<p class='error-block' >%v</p>\n", errMsg)
		}

		// label positioning for tall inputs
		specialVAlign := ""
		if toInputType(tp, attrs) == "textarea" {
			specialVAlign = "vertical-align: top;"
		}
		if toInputType(tp, attrs) == "select" {
			if structTag(attrs, "multiple") != "" {
				specialVAlign = "vertical-align: top;"
			}
		}
		if toInputType(tp, attrs) != "separator" &&
			toInputType(tp, attrs) != "fieldset" {
			fmt.Fprintf(w,
				"\t<label for='%s' style='%v' >%v</label>\n", // no whitespace - input immediately afterwards
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
			fmt.Fprintf(w, "\t<input type='%v' name='%v' id='%v' value='%v' %v %v />\n", toInputType(tp, attrs), inpName, inpName, "true", checked, structTagsToAttrs(attrs))
			fmt.Fprintf(w, "\t<input type='hidden' name='%v' value='false' />", inpName)
		case "file":
			needSubmit = true
			//              <input type="file" name="upload" id="upload" value="ignored.json" accept=".json" >
			fmt.Fprintf(w, "\t<input type='%v'   name='%v'     id='%v'     value='%v' %v />",
				toInputType(tp, attrs), inpName, inpName, "ignored.json", structTagsToAttrs(attrs),
			)
		case "date", "time":
			needSubmit = true
			//              <input type="date" name="myDate" max="1989-10-29"  min="2001-01-02">
			fmt.Fprintf(w, "\t<input type='%v'   name='%v'     id='%v'     value='%v' %v />",
				toInputType(tp, attrs), inpName, inpName, val, structTagsToAttrs(attrs),
			)
		case "textarea":
			needSubmit = true
			fmt.Fprintf(w, "\t<textarea name='%v' id='%v' %v />",
				inpName, inpName, structTagsToAttrs(attrs),
			)
			fmt.Fprint(w, val)
			fmt.Fprintf(w, "</textarea>")
		case "select":
			if structTag(attrs, "onchange") == "" {
				needSubmit = true // select without auto submit => needs submit button
			}
			fmt.Fprint(w, "\t<div class='select-arrow'>\n")
			fmt.Fprintf(w, "\t<select name='%v' id='%v' %v />\n", inpName, inpName, structTagsToAttrs(attrs))
			fmt.Fprint(w, s2f.selectOptions[inpName].HTML(valStrs))
			fmt.Fprint(w, "\t</select>\n")
			fmt.Fprint(w, "\t</div>")
			if structTag(attrs, "wildcardselect") != "" {
				fmt.Fprint(w, "\t\t<div class='wildcardselect'>\n")
				// onchange only triggers on blur
				// onkeydown makes too much noise
				// oninput is just perfect
				fmt.Fprintf(w, `		  <input type='text' name='%v' id='%v' value='%v'
					title='case sensitive | multiple patterns with * | separated by ; | ! negates'
					oninput='javascript:selectOptions(this);'
					maxlength='40'
					xxtabindex=-1
					placeholder='a*;b*'
					/>`,
					inpName+"_so",
					inpName+"_so",
					"",
				)
				fmt.Fprint(w, "\n\t\t</div>")
				/*
					JS function is printed repeatedly for multiple selects
					and multiple forms per request.
					The complexity of keeping track would be even more ugly.
				*/
				fmt.Fprintf(w, `
				<script type="text/javascript">

				var wildcardselectDebug = false;

				function matchRule(str, rule) {
					// define an arrow function with =>
					// creating the func escapeRegex()
					// escape all regex control characters; i.e. [ with \[
					// this could be moved out into a plain JS function
					var escapeRegex = (strArg) => strArg.replace(/([.*+?^=!:${}()|\[\]\/\\])/g, "\\$1");

					// split by *
					// escape regex chars of the parts
					// join  by .*
					// "."  matches single character, except newline or line terminator
					// ".*" matches any string containing zero or more characters
					rule = rule.split("*").map(escapeRegex).join(".*");

					// "^" is expression start
					// "$" is expression end
					rule = "^" + rule + "$"

					if (wildcardselectDebug) {
						console.log("     testing rule '" + rule + "' on str '" + str + "'");
					}

					// create a regular expression object for matching string
					var regex = new RegExp(rule);

					//Returns true if it finds a match, otherwise it returns false
					return regex.test(str);
				}

				function selectOptions(src) {
					// console.log(src)
					if (src) {
						var myName = src.getAttribute("name");
						// console.log("on input " + myName);
						var selectName = myName.substring(0, myName.length - 3);
						// console.log("  corresponding select is " + selectName);

						var select = document.getElementById(selectName);
						if (select) {
							var wildcards = src.value;
							var wildcardsArray = wildcards.split(";");
							for (idx = 0; idx < wildcardsArray.length; ++idx) {
								var wildcard = wildcardsArray[idx];
								var negate = false;
								if (wildcard.charAt(0) === "!") {
									wildcard = wildcard.substring(1);
									var negate = true;
								}
								for (var i = 0, l = select.options.length, o; i < l; i++) {
									o = select.options[i];
									var doesMatch = matchRule(o.text, wildcard);
									// if (negate) {
									// 	doesMatch = !doesMatch;
									// }
									if (doesMatch && !negate) {
										o.selected = true;
										if (wildcardselectDebug) {
											console.log("   selected     " + o.text + " - wildcard '" + wildcard + "' - negation " + negate);
										}
									} else if (doesMatch && negate) {
										o.selected = false;
										if (wildcardselectDebug) {
											console.log(" unselected     " + o.text + " - wildcard '" + wildcard + "' - negation " + negate);
										}
									} else {
										if (wildcardselectDebug) {
											console.log("   no match     " + o.text + " - wildcard '" + wildcard + "' - negation " + negate);
										}
									}
								}

							}

						}
					}
				}

				</script>

				`)
			}

		case "separator":
			fmt.Fprint(w, "\t<div class='separator'></div>")
		case "fieldset":
			if fieldsetOpen {
				fmt.Fprint(w, "</fieldset>\n")
			}
			fmt.Fprint(w, "<fieldset>")
			fmt.Fprintf(w, "\t<legend>&nbsp;%v&nbsp;</legend>", frmLabel)
			fieldsetOpen = true
		default:
			// plain vanilla input
			needSubmit = true
			fmt.Fprintf(w, "\t<input type='%v' name='%v' id='%v' value='%v' %v />", toInputType(tp, attrs), inpName, inpName, val, structTagsToAttrs(attrs))

		}

		sfx := structTag(attrs, "suffix")
		if sfx != "" {
			fmt.Fprintf(w, "<span class='postlabel' >%s</span>", sfx)
		}

		if toInputType(tp, attrs) != "separator" &&
			toInputType(tp, attrs) != "fieldset" &&
			structTag(attrs, "nobreak") == "" {
			fmt.Fprintf(w, "\n")
			fmt.Fprintf(w, s2f.verticalSpacer())
		}

		// close input with newline
		fmt.Fprintf(w, "\n")

	}

	if fieldsetOpen {
		fmt.Fprint(w, "</fieldset>\n")
	}

	if needSubmit || s2f.ForceSubmit {
		// name should *not* be 'submit'
		// avoiding error on this.form.submit()
		// 'submit is not a function' stackoverflow.com/questions/833032/
		fmt.Fprintf(w, "\t<button  type='submit' name='btnSubmit' value='1' accesskey='s'  ><b>S</b>ubmit</button>\n%v\n", s2f.verticalSpacer())
	} else {
		fmt.Fprintf(w, "\t<input   type='hidden' name='btnSubmit' value='1'\n")
	}

	if s2f.FormTag {
		fmt.Fprint(w, "</form>\n")
	}
	fmt.Fprint(w, "</div><!-- </div class='struc2frm'... -->\n")

	if inputWithFocus != "" {
		fmt.Fprintf(w, `
			<script type="text/javascript">

			var frm;
			var forms = document.getElementsByName("%v");
			for (var i1 = 0; i1 < forms.length; i1++) {
				if (forms[i1].tagName == "FORM") {
					frm = forms[i1];
					break;
				}
			}

			var elements = frm.elements;
			for (var i1 = 0; i1 < elements.length; i1++) {
				var name = elements[i1].getAttribute("name");
				if ( name === "%v") {
					if (elements[i1].type !== "hidden") {
						console.log("element to set focus", name);
						elements[i1].focus();
					}
				}
			}

			</script>
			`, s2f.Name, inputWithFocus)
	}

	// global replacements
	ret := strings.ReplaceAll(w.String(), "&comma;", ",")

	return template.HTML(ret)
}

// HTML takes a struct instance
// and uses the default formatter
// to turns it into an HTML form.
func HTML(intf interface{}) template.HTML {
	return defaultS2F.Form(intf)
}

func indentedDump(v interface{}) string {

	firstColLeftMostPrefix := " "
	byts, err := json.MarshalIndent(v, firstColLeftMostPrefix, "\t")
	if err != nil {
		s := fmt.Sprintf("error indent: %v\n", err)
		return s
	}
	// byts = bytes.Replace(byts, []byte(`\u003c`), []byte("<"), -1)
	// byts = bytes.Replace(byts, []byte(`\u003e`), []byte(">"), -1)
	// byts = bytes.Replace(byts, []byte(`\n`), []byte("\n"), -1)
	return string(byts)
}

// Decode the http request form into ptr2Struct;
// validating the CSRF token (https://en.wikipedia.org/wiki/Cross-site_request_forgery);
// deriving the 'populated' return value from the existence of the CSRF token.
// We *could* call Validate() on ptr2Struct if implemented;
// but valid is *more* than just populated.
func Decode(r *http.Request, ptr2Struct interface{}) (populated bool, err error) {
	err = r.ParseForm()
	if err != nil {
		return false, errors.Wrapf(err, "cannot parse form: %v<br>\n <pre>%v</pre>", err, indentedDump(r.Form))
	}
	return decode(r, ptr2Struct)
}

// DecodeMultipartForm decodes the form into an instance of struct
// and checks the token against CSRF attacks (https://en.wikipedia.org/wiki/Cross-site_request_forgery)
func DecodeMultipartForm(r *http.Request, ptr2Struct interface{}) (populated bool, err error) {
	err = ParseMultipartForm(r)
	if err != nil {
		return false, errors.Wrapf(err, "cannot parse multi part form: %v<br>\n <pre>%v</pre>", err, indentedDump(r.Form))
	}
	return decode(r, ptr2Struct)
}

func decode(r *http.Request, ptr2Struct interface{}) (populated bool, err error) {

	//
	// check for empty requests
	_, hasToken := r.Form["token"] // missing validation token
	ln := len(r.Form)              // request form is empty
	// sm := r.FormValue("btnSubmit") != ""  // submit btn would not be present in single dropdown forms with onclick
	if ln > 0 && !hasToken {
		log.Printf("warning: request params ignored, due to missing validation token")
	}
	if ln < 1 || !hasToken {
		return false, nil
	}

	err = New().ValidateFormToken(r.Form.Get("token"))
	if err != nil {
		return true, errors.Wrap(err, "form token exists; but invalid")
	}

	dec := form.NewDecoder()
	dec.SetTagName("json")
	err = dec.Decode(ptr2Struct, r.Form)
	if err != nil {
		return true, errors.Wrapf(err, "cannot decode form: %v<br>\n <pre>%v</pre>", err, indentedDump(r.Form))
	}

	// this belongs outside of the library into application side
	if false {
		if vldr, ok := ptr2Struct.(Validator); ok {
			_, valid := vldr.Validate()
			if !valid {
				return false, nil
			}
		}
	}

	return true, nil

}
