package struc2frm

import (
	"crypto"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/form"
)

// FormH is an example http handler func
func FormH(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	err := req.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "Cannot parse form: %v<br>\n", err)
		return
	}

	// bts2, _ := json.MarshalIndent(req.Form, " ", "\t")
	// fmt.Fprintf(w, "<br><br>Form was: <pre>%v</pre> <br>\n", string(bts2))

	type entryForm struct {
		Department  string `json:"department,omitempty"    form:"subtype='select',accesskey='p',onchange='true',title='loading items'"`
		Separator01 string `json:"separator01,omitempty"   form:"subtype='separator'"`
		HashKey     string `json:"hashkey,omitempty"       form:"maxlength='16',size='16',suffix='salt&comma; changes randomness'"` // the &comma; instead of , prevents wrong parsing
		Groups      int    `json:"groups,omitempty"        form:"min=1,max='100',maxlength='3',size='3'"`
		Items       string `json:"items,omitempty"         form:"subtype='textarea',cols='22',rows='12',maxlength='4000',title='add times - delimited by newline (enter)'"`
		Group01     string `json:"group01,omitempty"       form:"subtype='fieldset'"`
		Date        string `json:"date,omitempty"          form:"subtype='date',nobreak=true,min='1989-10-29',max='2030-10-29'"`
		Time        string `json:"time,omitempty"          form:"subtype='time',maxlength='12',size='12'"`
		Group02     string `json:"group02,omitempty"       form:"subtype='fieldset'"`
		// stackoverflow.com/questions/399078 - inside character classes escape ^-]\
		DateLayout string `json:"date_layout,omitempty"   form:"accesskey='t',maxlength='16',size='16',pattern='[0-9\\.\\-/]{2&comma;10}',placeholder='2006/01/02 15:04'"` // 2006-01-02 15:04
		CheckThis  bool   `json:"checkthis,omitempty"     form:"suffix='without consequence'"`

		// Requires distinct way of form parsing
		// Upload     []byte `json:"upload,omitempty"       form:"accesskey='u',accept='.xlsx'"`

		// Email would be
		// Email string `json:"email"        form:"maxlength='42',size='28',pattern='[a-zA-Z0-9\\.\\-_%+]+@[a-zA-Z0-9\\.\\-]+\\.[a-zA-Z]{2&comma;18}'"`
	}

	s2f := New()
	s2f.ShowHeadline = true
	s2f.AddOptions("department", []string{"ub", "fm"}, []string{"UB", "FM"})

	// init values
	frm := entryForm{
		HashKey: time.Now().Format("2006-01-02"),
		Groups:  4,
		Date:    time.Now().Format("2006-01-02"),
		Time:    time.Now().Format("15:04"),
	}

	dec := form.NewDecoder()
	dec.SetTagName("json") // recognizes and ignores ,omitempty
	err = dec.Decode(&frm, req.Form)
	if err != nil {
		fmt.Fprintf(w, "Could not decode form: %v <br>\n", err)
	}

	dept := req.FormValue("department")
	if dept == "" {
		dept = s2f.DefaultOptionKey("department")
	}
	frm.Items = strings.Join(CfgGet().ItemGroups[dept], "\n")

	// fmt.Fprintf(w, "dept is %v<br>", dept)

	//
	// reshuffling...
	bins := [][]string{}
	binsF := "" // formatted as html
	if req.FormValue("btnSubmit") != "" {

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

	// log.Printf("%v", bins)

	fmt.Fprintf(
		w,
		defaultHTML,
		s2f.HTML(frm),
		binsF,
	)

}
