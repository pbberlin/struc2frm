package struc2frm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMainH(t *testing.T) {

	req, err := http.NewRequest("GET", "/", nil) // create request without query params
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder() //  satisfying http.ResponseWriter for recording
	handler := http.HandlerFunc(FormH)

	handler.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("returned status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	expected := `<h3>Entry form</h3>
<form  method='POST' >
	<input name='token'    type='hidden'   value='%v' />
	<label for='department' style='' >De<u>p</u>artment</label>
	<div class='select-arrow'>
	<select name='department' id='department'  subtype='select' accesskey='p' onchange='javascript:this.form.submit();' title='loading items' />
		<option value='ub'          >UB</option>
		<option value='fm'          >FM</option>
	</select>
	</div>
	<div style='height:0.6rem'>&nbsp;</div>
	<div class='separator'></div>
	<label for='hashkey' style='' >Hashkey</label>
	<input type='text' name='hashkey' id='hashkey' value='%v'  maxlength='16' size='16' /><span class='postlabel' >salt, changes randomness</span>
	<div style='height:0.6rem'>&nbsp;</div>
	<label for='groups' style='' >Groups</label>
	<input type='number' name='groups' id='groups' value='4'  min=1 max='100' maxlength='3' size='3' />
	<div style='height:0.6rem'>&nbsp;</div>
	<label for='items' style='vertical-align: top;' >Items</label>
	<textarea name='items' id='items'  subtype='textarea' cols='22' rows='12' maxlength='4000' title='add times - delimited by newline (enter)' />Brutsyum, Zusoh
Dovosuke, Udsyuke
Fyrkros, Loekyo
Gyaffsydu, Loekusde
Heyos, Ysyr
Rtoynbsonnos, Tars</textarea>
	<div style='height:0.6rem'>&nbsp;</div>
<fieldset>	<legend>&nbsp;Group 01&nbsp;</legend>
	<label for='date' style='' >Date</label>
	<input type='date'   name='date'     id='date'     value='%v'  subtype='date' min='1989-10-29' max='2030-10-29' />
	<label for='time' style='' >Time</label>
	<input type='time'   name='time'     id='time'     value='%v'  subtype='time' maxlength='12' size='12' />
	<div style='height:0.6rem'>&nbsp;</div>
</fieldset>
<fieldset>	<legend>&nbsp;Group 02&nbsp;</legend>
	<label for='date_layout' style='' >Da<u>t</u>e layout</label>
	<input type='text' name='date_layout' id='date_layout' value=''  accesskey='t' maxlength='16' size='16' pattern='[0-9\.\-/]{2,10}' placeholder='2006/01/02 15:04' />
	<div style='height:0.6rem'>&nbsp;</div>
	<label for='checkthis' style='' >Checkthis</label>
	<input type='checkbox' name='checkthis' id='checkthis' value='true'   />
	<input type='hidden' name='checkthis' value='false' /><span class='postlabel' >without consequence</span>
	<div style='height:0.6rem'>&nbsp;</div>
</fieldset>
	<button  type='submit' name='btnSubmit' value='1' accesskey='s'  ><b>S</b>ubmit</button>
	<div style='height:0.6rem'>&nbsp;</div>
</form>
</div>`

	expected = fmt.Sprintf(
		expected,
		New().FormToken(),
		time.Now().Format("2006-01-02"),
		time.Now().Format("2006-01-02"),
		time.Now().Format("15:04"),
	)
	body := w.Body.String()
	if !strings.Contains(body, expected) {
		// t.Errorf("handler returned unexpected body: got %v want %v", w.Body.String(), expected)
		t.Errorf("handler returned unexpected body")
		ioutil.WriteFile("tmp-test_want.html", []byte(expected), 0777)
		ioutil.WriteFile("tmp-test_got.html", []byte(body), 0777)
	}
}
