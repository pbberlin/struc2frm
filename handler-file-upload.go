package struc2frm

import (
	"fmt"
	"net/http"

	"github.com/go-playground/form"
)

// FileUploadH is an http handler func for file upload
func FileUploadH(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	err := ParseMultipartForm(req)
	if err != nil {
		fmt.Fprintf(w, "Cannot parse multi part form: %v<br>\n", err)
		return
	}

	// bts2, _ := json.MarshalIndent(req.Form, " ", "\t")
	// fmt.Fprintf(w, "<br><br>Form was: <pre>%v</pre> <br>\n", string(bts2))

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

}
