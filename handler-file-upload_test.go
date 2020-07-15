package struc2frm

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
)

func TestFileUpload(t *testing.T) {

	fileName := "upload-file.txt"
	uploadBody := &bytes.Buffer{}

	if false { // normally we would provide the contents of a real file...
		filePath := path.Join(".", fileName)
		file, _ := os.Open(filePath)
		defer file.Close()
	}
	file := strings.NewReader("file content 123") // instead more easy

	mpWriter := multipart.NewWriter(uploadBody)
	part, err := mpWriter.CreateFormFile("upload", fileName)
	if err != nil {
		t.Fatalf("could not create form multi part: %v", err)
	}
	io.Copy(part, file)

	// adding normal fields...
	mpWriter.WriteField("text_field", "posted-text")
	mpWriter.WriteField("token", New().FormToken())
	mpWriter.Close()

	req, err := http.NewRequest("POST", "/file-upload", uploadBody) // <-- encoded payload
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}
	/*
		Content-Type is *not*
		   application/x-www-form-urlencoded
		but
		   multipart/form-data; boundary=...
	*/
	req.Header.Add("Content-Type", mpWriter.FormDataContentType())

	w := httptest.NewRecorder() //  satisfying http.ResponseWriter for recording
	handler := http.HandlerFunc(FileUploadH)

	handler.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("returned status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	expected1 := `<form  name='frmMain'  method='post'   enctype='multipart/form-data'>
	<input name='token'    type='hidden'   value='%v' />
	<label for='text_field' style='' >Text field</label>
	<input type='text' name='text_field' id='text_field' value='posted-text'  maxlength='16' size='16' />
	<div style='height:0.6rem'>&nbsp;</div>
	<label for='upload' style='' ><u>U</u>pload</label>
	<input type='file'   name='upload'     id='upload'     value='ignored.json'  accesskey='u' accept='.txt' /><span class='postlabel' >*.txt files</span>
	<div style='height:0.6rem'>&nbsp;</div>
	<button  type='submit' name='btnSubmit' value='1' accesskey='s'  ><b>S</b>ubmit</button>
	<div style='height:0.6rem'>&nbsp;</div>
</form>`

	expected1 = fmt.Sprintf(expected1, New().FormToken())

	expected2 := `16 bytes read from excel file -upload-file.txt- <br>
File content is --file content 123-- <br>`

	body := w.Body.String()

	if !strings.Contains(body, expected1) {
		t.Errorf("handler returned unexpected body")
		ioutil.WriteFile("tmp-test-fileupload1_want.html", []byte(expected1), 0777)
		ioutil.WriteFile("tmp-test-fileupload1_got.html", []byte(body), 0777)
	}
	if !strings.Contains(body, expected2) {
		t.Errorf("handler did not get the uploaded file name or data right")
		ioutil.WriteFile("tmp-test-fileupload2_want.html", []byte(expected2), 0777)
		ioutil.WriteFile("tmp-test-fileupload2_got.html", []byte(body), 0777)
	}
}
