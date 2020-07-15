package struc2frm

import (
	"fmt"
	"log"
	"net/http"
)

// FileUploadH is an http handler func for file upload
func FileUploadH(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")

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
	// if len(bts) > 0 && excelFileName != "" {
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

}
