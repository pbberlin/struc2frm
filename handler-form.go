package struc2frm

import (
	"crypto"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

var itemGroups = map[string][]string{
	"ub": {
		"Brutsyum, Zusoh",
		"Dovosuke, Udsyuke",
		"Fyrkros, Loekyo",
		"Gyaffsydu, Loekusde",
		"Heyos, Ysyr",
		"Rtoynbsonnos, Tars",
	},
	"fm": {
		"Bsackbuaos, Punk",
		"Bachos-Keonon, Tasd",
		"Hiroso, Meivynu",
		"Rachydt, Racho",
		"Ruchsiedos, Misea",
	},
}

type entryForm struct {
	Department  string   `json:"department,omitempty"    form:"subtype='select',accesskey='p',onchange='true',label='Department/Abteilung',title='loading items'"`
	Separator01 string   `json:"separator01,omitempty"   form:"subtype='separator'"`
	HashKey     string   `json:"hashkey,omitempty"       form:"maxlength='16',size='16',autocapitalize='off',suffix='salt&comma; changes randomness'"` // the &comma; instead of , prevents wrong parsing
	Groups      int      `json:"groups,omitempty"        form:"min=1,max='100',maxlength='3',size='3'"`
	Items       string   `json:"items,omitempty"         form:"subtype='textarea',cols='22',rows='4',maxlength='4000',label='Textarea of<br>line items',title='add times - delimited by newline (enter)'"`
	Items2      []string `json:"items2,omitempty"        form:"subtype='select',size='3',multiple='true',label='Multi<br>select<br>dropdown',autofocus='true'"`
	Group01     string   `json:"group01,omitempty"       form:"subtype='fieldset'"`
	Date        string   `json:"date,omitempty"          form:"subtype='date',nobreak=true,min='1989-10-29',max='2030-10-29'"`
	Time        string   `json:"time,omitempty"          form:"subtype='time',maxlength='12',inputmode='numeric',size='12'"`
	Group02     string   `json:"group02,omitempty"       form:"subtype='fieldset'"`
	// stackoverflow.com/questions/399078 - inside character classes escape ^-]\
	DateLayout string `json:"date_layout,omitempty"   form:"accesskey='t',maxlength='16',size='16',pattern='[0-9\\.\\-/]{2&comma;10}',placeholder='2006/01/02 15:04',label='Layout of the date'"` // 2006-01-02 15:04
	CheckThis  bool   `json:"check_this,omitempty"    form:"suffix='without consequence'"`

	// Requires distinct way of form parsing
	// Upload     []byte `json:"upload,omitempty"       form:"accesskey='u',accept='.xlsx'"`

	// Email would be
	// Email string `json:"email"        form:"maxlength='42',size='28',pattern='[a-zA-Z0-9\\.\\-_%+]+@[a-zA-Z0-9\\.\\-]+\\.[a-zA-Z]{2&comma;18}'"`
}

// Validate checks whether form entries as a whole are "submittable";
// more than just 'populated';
// Validate generates error messages
func (frm entryForm) Validate() (map[string]string, bool) {
	errs := map[string]string{}
	g1 := frm.Department != ""
	g2 := frm.CheckThis
	if !frm.CheckThis {
		errs["check_this"] = "You need to comply"
	}
	g3 := frm.Items != ""
	return errs, g1 && g2 && g3
}

// FormH is an example http handler func
func FormH(w http.ResponseWriter, req *http.Request) {

	w.Header().Add("Content-Type", "text/html")

	s2f := New()
	s2f.ShowHeadline = true
	s2f.FocusFirstError = true
	s2f.SetOptions("department", []string{"ub", "fm"}, []string{"UB", "FM"})
	s2f.SetOptions("items2", []string{"anton", "berta", "caesar", "dora"}, []string{"Anton", "Berta", "Caesar", "Dora"})
	// s2f.Method = "GET"

	// init values - non-multiple
	frm := entryForm{
		HashKey: time.Now().Format("2006-01-02"),
		Groups:  2,
		Date:    time.Now().Format("2006-01-02"),
		Time:    time.Now().Format("15:04"),
	}

	// pulling in values from http request
	populated, err := Decode(req, &frm)
	if populated && err != nil {
		s2f.AddError("global", fmt.Sprintf("cannot decode form: %v<br>\n <pre>%v</pre>", err, indentedDump(req.Form)))
		log.Printf("cannot decode form: %v<br>\n <pre>%v</pre>", err, indentedDump(req.Form))
	}

	// init values - multiple
	if !populated {
		if len(frm.Items2) == 0 {
			frm.Items2 = []string{"berta", "dora"}
		}
	}

	if req.Form.Get("debug") != "" {
		fmt.Fprintf(w, "<pre>%v</pre>", indentedDump(req.Form))
		fmt.Fprintf(w, "<pre>%v</pre>", indentedDump(frm))
	}

	dept := req.FormValue("department")
	if dept == "" {
		dept = s2f.DefaultOptionKey("department")
	}
	frm.Items = strings.Join(itemGroups[dept], "\n")

	errs, valid := frm.Validate()

	//
	// business logic: reshuffling...
	bins := [][]string{}
	binsF := "" // formatted as html

	if populated {

		if !valid {
			s2f.AddErrors(errs) // add errors only for a populated form
		} else {
			// further processing
			// see below
		}

		salt1 := req.FormValue("hashkey")
		salt2 := "dudoedeldu"

		num, _ := strconv.Atoi(req.FormValue("groups"))
		items := strings.Split(req.FormValue("items"), "\n")
		for i := 0; i < len(items); i++ {
			items[i] = strings.TrimSpace(items[i])
		}

		itemMp := map[string]string{}
		keys := []string{}

		hasher := crypto.MD5.New()
		for _, item := range items {
			hasher.Write([]byte(item + salt1 + salt2))
			key := string(hasher.Sum(nil))
			itemMp[key] = item
			keys = append(keys, key)
		}
		sort.Strings(keys)
		items = make([]string, 0, len(items))
		for _, key := range keys {
			items = append(items, itemMp[key])
		}

		bins = make([][]string, num)

		for itemCounter, item := range items {
			binID := itemCounter % num
			bins[binID] = append(bins[binID], item)
		}

		for i := 0; i < len(bins); i++ {
			binsF += "<div class='res'>\n"
			binsF += fmt.Sprintf("\t<b>Group %v</b><br>\n", i+1)
			for j := 0; j < len(bins[i]); j++ {
				binsF += "\t" + bins[i][j] + "<br>\n"
			}
			binsF += "</div>\n\n"
		}

	}

	// render to HTML for user input / error correction
	if !valid {
		// fmt.Fprint(w, s2f.Form(frm))
	}

	fmt.Fprintf(
		w,
		defaultHTML,
		s2f.Form(frm),
		binsF,
	)

}
