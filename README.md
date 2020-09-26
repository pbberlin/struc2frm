# struc2frm

![./struc2frm.jpg](./struc2frm.jpg)

## Golang Struct to HTML Form

[![GoDoc](http://godoc.org/github.com/pbberlin/struc2frm?status.svg)](https://godoc.org/github.com/pbberlin/struc2frm) 
[![Travis Build](https://travis-ci.org/pbberlin/struc2frm.svg?branch=master)](https://travis-ci.org/pbberlin/struc2frm) 
[![codecov](https://codecov.io/gh/pbberlin/struc2frm/branch/master/graph/badge.svg)](https://codecov.io/gh/pbberlin/struc2frm) 




* Package struc2frm converts or transforms a  
golang `struct type` into an `HTML input form`.

* All your admin and backend forms generated directly from golang structs.

* HTML input field info is taken from the `form` struct tag.

* Decode() and DecodeMultipartForm() transform the HTTP request data 
back into an instance of the `struct type` used for the HTML code.

* Decode() and DecodeMultipartForm() also check the   
auto-generated form token against [CSRF attacks](https://en.wikipedia.org/wiki/Cross-site_request_forgery).  

* Use `Form()` to render an HTML form  

* Use `Card()` to render a read-only HTML card.

<img src="card-view.jpg" height="144px;" style="margin-left:20px;position: relative; top: -10px;" >

* Fully functional example-webserver in directory `systemtest`;  
compile and run, then  
[Main example](http://localhost:8085/)  
[File upload example](http://localhost:8085/file-upload)

## Example use

```golang
type entryForm struct {
    Department  string `json:"department,omitempty"    form:"subtype='select',accesskey='p',onchange='true',label='Department/Abteilung',title='loading items'"`
    Separator01 string `json:"separator01,omitempty"   form:"subtype='separator'"`
    HashKey     string `json:"hashkey,omitempty"       form:"maxlength='16',size='16',autocapitalize='off',suffix='salt&comma; changes randomness'"` // the &comma; instead of , prevents wrong parsing
    Groups      int    `json:"groups,omitempty"        form:"min=1,max='100',maxlength='3',size='3'"`
    Items       string   `json:"items,omitempty"         form:"subtype='textarea',cols='22',rows='4',maxlength='4000',label='Textarea of<br>line items',title='add times - delimited by newline (enter)'"`
    Items2      []string `json:"items2,omitempty"        form:"subtype='select',size='3',multiple='true',label='Multi<br>select<br>dropdown'"`
    Group01     string `json:"group01,omitempty"       form:"subtype='fieldset'"`
    Date        string `json:"date,omitempty"          form:"subtype='date',nobreak=true,min='1989-10-29',max='2030-10-29'"`
    Time        string `json:"time,omitempty"          form:"subtype='time',maxlength='12',inputmode='numeric',size='12'"`
    Group02     string `json:"group02,omitempty"       form:"subtype='fieldset'"`
    DateLayout  string `json:"date_layout,omitempty"   form:"accesskey='t',maxlength='16',size='16',pattern='[0-9\\.\\-/]{2&comma;10}',placeholder='2006/01/02 15:04',label='Layout of the date'"` // 2006-01-02 15:04
    CheckThis   bool   `json:"checkthis,omitempty"     form:"suffix='without consequence'"`

    // Requires distinct way of form parsing
    // Upload     []byte `json:"upload,omitempty"       form:"accesskey='u',accept='.xlsx'"`
}

// Validate checks whether form entries as a whole are "submittable";
// more than 'populated'
func (frm entryForm) Validate() bool {
    g1 := frm.Department != ""
    g2 := frm.CheckThis && frm.Items != ""
    return g1 && g2
}


// getting a converter
s2f := struc2frm.New()  // or clone existing one
s2f.ShowHeadline = true // set options
s2f.SetOptions("department", []string{"ub", "fm"}, []string{"UB", "FM"})

// init values
frm := entryForm{
    HashKey: time.Now().Format("2006-01-02"),
    Groups:  4,
    Date:    time.Now().Format("2006-01-02"),
    Time:    time.Now().Format("15:04"),
}

// pulling in values from http request
populated, err := Decode(req, &frm)
if populated && err != nil {
    s2f.AddError("global", fmt.Sprintf("cannot decode form: %v<br>\n <pre>%v</pre>", err, indentedDump(r.Form)))
    log.Printf("cannot decode form: %v<br>\n <pre>%v</pre>", err, indentedDump(r.Form))
}

if populated {
    valid := frm.Validate()
    if !valid {
        // business logic
    }
    // more business logic
}

// render to HTML
fmt.Fprint(w, s2f.Form(frm))
```

## Usage - specific field types

* Use `float64` or `int` to create number inputs - with attributes `min=1,max=100,step=2`.  
Notice that `step=2` defines maximum precision; uneven numbers become invalid.  
This is an [HTML5 restriction](https://stackoverflow.com/questions/14365348/).

* `string` supports attribute `placeholder='2006/01/02 15:04'` to show a pattern to the user (placeholder).

* `string` supports attribute `pattern='[0-9\\.\\-/]{10}'` to restrict the entry to a regular expression.

* Use attributes `maxlength='16'` and `size='16'`  
determine width and maximum content length respectively for `input` and `textarea`.  
Attribute `size` determines height for select/dropdown elements.

* Use `string` field with subtype `textarea` and attributes `cols='32',rows='22'`

* Use `string` field with subtype `date` and attributes  `min='1989-10-29'` or `max=...`

* Use `string` field with subtype `time`

* Use `bool` to create a checkbox

### Separator and fieldset

These are `dummmy` fields for formatting only

* Every `string` field with subtype `separator` is rendered into a horizontal line

* Every `string` field with subtype `fieldset` is rendered into grouping box with label

### Select / dropdown inputs

* Use `string | int | float64 | bool` field with subtype `select`

* Use `size=1` or `size=5` to determine the height

* Use `SetOptions()` to fill input[select] elements

* Use `DefaultOptionKey()` to pre-select an option other than the first on clean forms

* Use `onchange='true'` for onchange submit

* Use `multiple='true'` to enable the selection of __multiple items__  
  in conjunction with struct field type `[]string | []int | []float64 | []bool`

## Submit button

If your form only has `select` inputs with `onchange='this.form.submit()'`  
then no submit button is shown.

This can be overridden by setting `struc2frm.New().ShowSubmit` to true.

## File upload

* input[file] must have golang type `[]byte`

* input[file] should be named `upload`  
and _requires_ `ParseMultipartForm()` instead of `ParseForm()`

* `DecodeMultipartForm()` and `ExtractUploadedFile()` are helper funcs  
to extract file upload data

Example

```golang

type entryForm struct {
    TextField string `json:"text_field,omitempty"   form:"maxlength='16',size='16'"`
    // Requires distinct way of form parsing
    Upload []byte `json:"upload,omitempty"          form:"accesskey='u',accept='.txt',suffix='*.txt files'"`
}

s2f := struc2frm.New()  // or clone existing one
s2f.ShowHeadline = true // set options
s2f.Indent = 80


// init values
frm := entryForm{
    TextField: "some-init-text",
}

populated, err := DecodeMultipartForm(req, &frm)
if populated && err != nil {
    s2f.AddError("global", fmt.Sprintf("cannot decode multipart form: %v<br>\n <pre>%v</pre>", err, indentedDump(req.Form)))
    log.Printf("cannot decode multipart form: %v<br>\n <pre>%v</pre>", err, indentedDump(req.Form))
}

bts, excelFileName, err := ExtractUploadedFile(req)
if err != nil {
    fmt.Fprintf(w, "Cannot extract file from POST form: %v<br>\n", err)
}

fileMsg := ""
if populated {
    fileMsg = fmt.Sprintf("%v bytes read from excel file -%v- <br>\n", len(bts), excelFileName)
    fileMsg = fmt.Sprintf("%vFile content is --%v-- <br>\n", fileMsg, string(bts))
} else {
    fileMsg = "No upload filename - or empty file<br>\n"

}

fmt.Fprintf(
    w,
    defaultHTML,
    s2f.HTML(frm),
    fileMsg,
)
```

See `handler-file-upload_test.go` on how to programmatically POST a file and key-values.

## General field attributes

* Use `form:"-"` to exclude fields from being rendered  
neither in form view nor in card view

* Every field can have an attribute `label=...`,  
appearing before the input element,  
if not specified, json:"[name]..." is labelized and used

* Every field can have an attribute `suffix=...`,  
appearing after the input element

* Every field can have an attribute `title=...`  
for mouse-over tooltips

* Values inside of `label='...'`, `suffix='...'`, `title='...'`, `placeholder='...'`, `pattern='...'`  
need `&comma;` instead  of `,`

* Every field  can have an attribute `accesskey='t'`  
Accesskeys are not put into the label, but into the input tag

* Every field  can have an attribute `nobreak='true'`  
so that the next input remains on the same line

### Field attributes for mobile phones

* `inputmode="numeric"` opens the numbers keyboard on mobile phones

* `autocapitalize=off` switches off first letter upper casing

## CSS Styling

* Styling is done via CSS selectors  
and can be customized  
by changing or appending `struc2frm.New().CSS`

* If you already have good styles in your website,  
set `CSS = ""`

```CSS
div.struc2frm {
    padding: 4px;
}
div.struc2frm  input {
    margin:  4px;
}
```

The media query CSS block in default `struc2frm.New().CSS`  
can be used to change the label width depending on screen width.

Label width can also be changed by setting  
via `struc2frm.New().Indent` and `struc2frm.New().IndentAddenum`
to none-zero values.

```CSS
div.struc2frm-34323168 H3 {
    margin-left: 116px; /* programmatically set via s3f.Indent for each form */
}
```

```CSS
/* change specific inputs */
div.struc2frm label[for="time"] {
    min-width: 20px;
}
div.struc2frm select[name="department"] {
    background-color: darkkhaki;
}
```

## Technical stuff

Language | files | blank | comment | code
---      | ---   | ---   | ---     | ---
Go               |                5   |         123      |       68    |        672
Markdown         |                1   |          61      |        0    |        150
CSS              |                1   |          26      |        8    |        120
HTML             |                1   |           6      |        1    |         30

* Default CSS is init-loaded from an in-package file `default.css`,  
mostly to have syntax highlighting while editing it.

## TODO

* ListView() with labels from `form` tag and values from SetOptions().

* Support for focus() first input element and  
focus() on first input element having an error

* Can we use `0x2C` instead of `,` ?

* Low Prio: Add field type `option group`  
meanwhile use `select / dropdown`
