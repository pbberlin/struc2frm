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
	FormTag     bool   // include <form...> and </form>
	Name        string // form name
	Method      string // form method - default is POST
	InstanceID  string // to distinguish several instances on same website
	FormTimeout int    // hours until a form post is rejected
	Salt        string

	SelectOptions map[string]options // select inputs get their options from here
	Errors        map[string]string  // validation errors by json name of input

	Indent         int     // horizontal width of the labels column
	IndentAddenum  int     // for h3-headline and submit button, depends on CSS paddings and margins of div and input
	ForceSubmit    bool    // show submit, despite having only auto-changing selects
	ShowHeadline   bool    // headline derived from struct name
	VerticalSpacer float64 // in CSS REM

	CSS string // general formatting - provided defaults can be replaced

}

// New converter
func New() *s2FT {
	s2f := s2FT{
		FormTag:       true,
		Name:          "frmMain",
		Method:        "POST",
		FormTimeout:   2,
		SelectOptions: map[string]options{},

		Indent:         0,           // non-zero values override the CSS
		IndentAddenum:  2 * (4 + 4), // horizontal padding and margin
		ForceSubmit:    false,
		ShowHeadline:   false,
		VerticalSpacer: 0.6,

		CSS: defaultCSS,
	}
	s2f.InstanceID = fmt.Sprint(time.Now().UnixNano())
	s2f.InstanceID = s2f.InstanceID[len(s2f.InstanceID)-8:] // last 8 digits

	// MAC address as salt
	ifs, _ := net.Interfaces()
	for _, v := range ifs {
		h := v.HardwareAddr.String()
		if len(h) == 0 {
			continue
		}
		s2f.Salt = h
		break
	}

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

func (s2f *s2FT) verticalSpacer() string {
	return fmt.Sprintf("\t<div style='height:%3.1frem'>&nbsp;</div>", s2f.VerticalSpacer)
}

// AddOptions is used by the caller to prepare option key-labels
// for the rendering into HTML()
func (s2f *s2FT) AddOptions(nameJSON string, keys, labels []string) {
	if s2f.SelectOptions == nil {
		s2f.SelectOptions = map[string]options{}
	}
	for i, key := range keys {
		s2f.SelectOptions[nameJSON] = append(s2f.SelectOptions[nameJSON], option{key, labels[i]})
	}
}

// AddError adds validations messages;
// key 'global' writes msg on top of form.
func (s2f *s2FT) AddError(nameJSON string, msg string) {
	if s2f.Errors == nil {
		s2f.Errors = map[string]string{}
	}
	s2f.Errors[nameJSON] = msg
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
			fmt.Fprintf(w, "\t\t<option value='%v' selected >%v</option>\n", o.Key, o.Val)
		} else {
			fmt.Fprintf(w, "\t\t<option value='%v'          >%v</option>\n", o.Key, o.Val)
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

// parsing the content of some special 'form' struct tag from the struct
// i.e. "maxlength='42',size='28',suffix='optional'"
func structTag(tags, key string) string {
	tagss := strings.Split(tags, ",")
	for _, a := range tagss {
		aLow := strings.ToLower(a)
		if strings.HasPrefix(aLow, key) {
			kv := strings.Split(a, "=")
			if len(kv) == 2 {
				// kv[1] = strings.ReplaceAll(kv[1], "&comma;", ",") // done at the end for the entire string
				return strings.Trim(kv[1], "'")
			}
		}
	}
	return ""
}

// convert the content of some special 'form' struct tag to html input attributes
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
		case strings.HasPrefix(tl, "accesskey="): // goes into input, not into label
			ret += " " + t
		case strings.HasPrefix(tl, "title="): // mouse over tooltip - alt
			ret += " " + t
		default:
			// suffix    is not converted into an attribute
			// nobreak   is not converted into an attribute
		}

	}
	// ret = strings.ReplaceAll(ret, "&comma;", ",") // done at the end for the entire string
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

	if s2f.FormTag {
		if uploadPostForm {
			fmt.Fprintf(w, "<form  name='%v'  method='post'   enctype='multipart/form-data'>\n", s2f.Name)
		} else {
			fmt.Fprintf(w, "<form name='%v'  method='%v' >\n", s2f.Name, s2f.Method)
		}
	}

	if errMsg, ok := s2f.Errors["global"]; ok {
		fmt.Fprintf(w, "\t<p class='error-block' >%v</p>\n", errMsg)
	}

	fmt.Fprintf(w, "\t<input name='token'    type='hidden'   value='%v' />\n", s2f.FormToken())

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

		attrs := ifVal.Type().Field(i).Tag.Get("form") // i.e. form:"maxlength='42',size='28',suffix='optional'"

		if strings.Contains(attrs, ", ") || strings.Contains(attrs, ", ") {
			return template.HTML(fmt.Sprintf("struct2form.HTML() - field %v: tag 'form' cannot contain ', ' or ' ,' ", fldName))
		}

		if attrs == "-" {
			continue
		}

		val := ifVal.Field(i)
		tp := ifVal.Field(i).Type().Name() // primitive type name: string, int
		if ifVal.Type().Field(i).Type.Kind() == reflect.Slice {
			tp = "[]" + ifVal.Type().Field(i).Type.Elem().Name() // []byte => []uint8
		}

		if errMsg, ok := s2f.Errors[inpName]; ok {
			fmt.Fprintf(w, "\t<p class='error-block' >%v</p>\n", errMsg)
		}

		// label
		specialVAlign := ""
		if toInputType(tp, attrs) == "textarea" {
			specialVAlign = "vertical-align: top;"
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
			fmt.Fprint(w, "\t<div class='select-arrow'>\n")
			fmt.Fprintf(w, "\t<select name='%v' id='%v' %v />\n", inpName, inpName, structTagsToAttrs(attrs))
			fmt.Fprint(w, s2f.SelectOptions[inpName].HTML(val.String()))
			fmt.Fprint(w, "\t</select>\n")
			fmt.Fprint(w, "\t</div>")
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
	fmt.Fprint(w, "</div><!-- </div class='struc2frm'... -->\n") //

	// global replacements
	ret := strings.ReplaceAll(w.String(), "&comma;", ",")

	return template.HTML(ret)
}

// HTML takes a struct instance
// and uses the default formatter
// to turns it into an HTML form.
func HTML(intf interface{}) template.HTML {
	return defaultS2F.HTML(intf)
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

// Decode decodes the form into an instance of stuct
// and checks the
func Decode(r *http.Request, ptr2Struct interface{}) (populated bool, err error) {

	err = r.ParseForm()
	if err != nil {
		return false, errors.Wrapf(err, "cannot parse form: %v<br>\n <pre>%v</pre>", err, indentedDump(r.Form))
	}

	_, ok := r.Form["token"]
	ln := len(map[string][]string(r.Form))
	if ln < 1 || !ok {
		// empty request form or missing validation token
		return false, nil
	}

	err = New().ValidateFormToken(r.Form.Get("token"))
	if err != nil {
		return false, errors.Wrap(err, "invalid form token")
	}

	dec := form.NewDecoder()
	dec.SetTagName("json")
	err = dec.Decode(ptr2Struct, r.Form)
	if err != nil {
		return false, errors.Wrapf(err, "cannot decode form: %v<br>\n <pre>%v</pre>", err, indentedDump(r.Form))
	}

	return true, nil

}
