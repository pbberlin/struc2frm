package main

import (
	"crypto"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/form"
	"github.com/pbberlin/struc2frm"
)

var defaultHTML = ""

func init() {
	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath := path.Join(path.Dir(filename), "tpl-main.html")
	bts, err := ioutil.ReadFile(sourceDirPath)
	if err != nil {
		log.Fatalf("Could not load main template: %v", err)
	}
	defaultHTML = string(bts)
}

func main() {

	rand.Seed(time.Now().UTC().UnixNano())
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	log.SetFlags(log.Lshortfile | log.Ltime)

	cfgLoad()
	pfx := CfgGet().URLPathPrefix

	mux1 := http.NewServeMux() // base router
	mux1.HandleFunc("/", mainH)
	if pfx != "" {
		mux1.HandleFunc("/"+pfx, mainH)
		mux1.HandleFunc("/"+pfx+"/", mainH)
	}

	mux4 := http.NewServeMux() // top router for non-middlewared handlers
	mux4.Handle("/", mux1)

	serveIcon := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		// w.Header().Set("Cache-Control", fmt.Sprintf("public,max-age=%d", 60*60*24))
		fv := "favicon.ico"
		bts, _ := ioutil.ReadFile("./static/" + fv)
		fmt.Fprint(w, bts)
		// log.Printf("%v bytes written", len(bts))
	}
	mux4.HandleFunc("favicon.ico", serveIcon)
	mux4.HandleFunc("/favicon.ico", serveIcon)
	if pfx != "" {
		mux1.HandleFunc("/"+pfx+"/favicon.ico", serveIcon)
		mux1.HandleFunc("/"+pfx+"/favicon.ico/", serveIcon)
	}

	IPPort := fmt.Sprintf("%v:%v", CfgGet().BindHost, CfgGet().BindSocket)
	log.Printf("starting http server at %v ... ", IPPort)
	log.Printf("==========================")
	log.Printf("  ")

	log.Fatal(http.ListenAndServe(IPPort, mux4))

}

func mainH(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	err := req.ParseForm()
	if err != nil {
		log.Fatalf("Could not parse form: %v", err)
	}

	// bts2, _ := json.MarshalIndent(req.Form, " ", "\t")
	// fmt.Fprintf(w, "<br><br>Form was: <pre>%v</pre> <br>\n", string(bts2))

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
