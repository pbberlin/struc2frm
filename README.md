
# struc2frm

![./struc2frm.jpg](./struc2frm.jpg)

## Golang Struct to HTML Form

[![Travis Build](https://travis-ci.org/pbberlin/struc2frm.svg?branch=master)](https://travis-ci.org/pbberlin/struc2frm)        [![codecov](https://codecov.io/gh/pbberlin/struc2frm/branch/master/graph/badge.svg)](https://codecov.io/gh/pbberlin/struc2frm)

* Package struc2frm converts or transforms a  
golang `struct` into an `HTML input form`.

* Tired of the boilerplate?  
All your admin and backend forms generated directly from golang structs.

* Field info is taken from the `json` struct tag.

* Additional attributes are taken from the `form` struct tag

* Package does not provide parsing request forms into a struct type.  
For this, we recommend `github.com/go-playground/form`  
since it accepts json tags despite containing `,omitempty`.  

* `github.com/go-playground/form` also tolerates superfluous request fields -  
thus the submit button does not cause an error.

Example

```golang
type entryForm struct {
    Department  string `json:"department,omitempty"    form:"subtype='select',accesskey='p',onchange='true',title='loading items'"`
    Separator01 string `json:"separator01,omitempty"   form:"subtype='separator'"`
    HashKey     string `json:"hashkey,omitempty"       form:"maxlength='16',size='16',suffix='changes randomness'"`
    Groups      int    `json:"groups,omitempty"        form:"min=1,max='100',maxlength='3',size='3'"`
    Items       string `json:"items,omitempty"         form:"subtype='textarea',cols='22',rows='12',maxlength='4000',title='add times - delimited by newline (enter)'"`
    Group01     string `json:"group01,omitempty"       form:"subtype='fieldset'"`
    Date        string `json:"date,omitempty"          form:"subtype='date',nobreak=true,min='1989-10-29',max='2030-10-29'"`
    Time        string `json:"time,omitempty"          form:"subtype='time',maxlength='12',size='12'"`
    Group02     string `json:"group02,omitempty"       form:"subtype='fieldset'"`
    DateLayout  string `json:"date_layout,omitempty"   form:"accesskey='t',maxlength='16',size='16',pattern='[0-9\\.\\-/]{10}',placeholder='2006/01/02 15:04'"` // 2006-01-02 15:04
    CheckThis   bool   `json:"checkthis,omitempty"     form:"suffix='without consequence'"`

    // Requires distinct way of form parsing
    // Upload     []byte `json:"upload,omitempty"       form:"accesskey='u',accept='.xlsx'"`
}

// getting a converter
s2f := struc2frm.New()
s2f.ShowHeadline = true
s2f.AddOptions("department", []string{"ub", "fm"}, []string{"UB", "FM"})

// init values
frm := entryForm{
    HashKey: time.Now().Format("2006-01-02"),
    Groups:  4,
    Date:    time.Now().Format("2006-01-02"),
    Time:    time.Now().Format("15:04"),
}

// pulling in values from http request
dec := form.NewDecoder()
dec.SetTagName("json") // recognizes and ignores ,omitempty
err = dec.Decode(&frm, req.Form)
if err != nil {
    fmt.Fprintf(w, "Could not decode form: %v <br>\n", err)
}

// render to HTML
fmt.Fprint(w, s2f.HTML(frm))
```

Fully functional example code in `directory systemtest`

## Select / dropdown inputs

* Use `string` field with subtype `select`

* Use `AddOptions()` to fill input[select] elements

* Use `onchange='true'`

## Other field specifics

* Use `float64` or `int` to create number inputs - with attributes `min=1,max=100,step=2`.  
Notice that `step=2` defines maximum precision; uneven number become invalid.  
This is an [HTML5 restriction](https://stackoverflow.com/questions/14365348/).

* `string`, `textarea`, `float64` and `int` fields have the attributes `maxlength='16',size='16'`

* `string` supports attributes `placeholder='2006/01/02 15:04',pattern='[0-9\\.\\-/]{10}'`  
to show a pattern to the user (placeholder)  
and to restrict the entry to a regular expression

* Use `string` field with subtype `textarea` and attributes `cols='32',rows='22'`

* Use `string` field with subtype `date` and attributes  `min='1989-10-29'` or `max=...`

* Use `string` field with subtype `time`

* Use `bool` to create a checkbox

* Every `string` field with subtype `separator` is rendered into a horizontal line

* Every `string` field with subtype `fieldset` is rendered into grouping box with label

## General

* Every field can have an attribute `suffix=...`

* Every field can have an attribute `title=...`  
for mouse-over tooltips

* Every field  can have an attribute `accesskey='t'`  
Accesskeys are not put into the label, but into the input tag

* Every field  can have an attribute `nobreak='true'`  
so that the next input remains on the same line

## File upload

* input[file] must have golang type `[]byte`

* input[file] should be named `upload`  
and _requires_ `ParseMultipartForm()` instead of `ParseForm()`

* `ParseMultipartForm()` and `ExtractUploadedFile()` are helper funcs  
to extract file upload data

Example

```golang
err := ParseMultipartForm(req)
if err != nil {
    fmt.Fprintf(w, "Cannot parse multi part form: %v<br>\n", err)
    return
}

type entryForm struct {
    TextField string `json:"text_field,omitempty"   form:"maxlength='16',size='16'"`
    // Requires distinct way of form parsing
    Upload []byte `json:"upload,omitempty"          form:"accesskey='u',accept='.txt',suffix='*.txt files'"`
}

s2f := New()
s2f.ShowHeadline = true
s2f.Indent = 80

// init values
frm := entryForm{
    TextField: "some-init-text",
}

dec := form.NewDecoder()
dec.SetTagName("json") // recognizes and ignores ,omitempty
err = dec.Decode(&frm, req.Form)
if err != nil {
    fmt.Fprintf(w, "Could not decode form: %v <br>\n", err)
}

bts, excelFileName, err := ExtractUploadedFile(req)
if err != nil {
    fmt.Fprintf(w, "Cannot extract file from POST form: %v<br>\n", err)
}

fileMsg := ""
if len(bts) > 0 && excelFileName != "" {
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

## CSS Styling

Styling is done via CSS selectors  
and can be customized  
by changing or appending `struc2frm.New().CSS`

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

## Submit button

If your form only has `select` inputs with `onchange='this.form.submit()'`  
then no submit button is shown.

This can be overridden by setting `struc2frm.New().ShowSubmit` to true.

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

* Low Prio: Add field type `option group`  
meanwhile use `select / dropdown`
