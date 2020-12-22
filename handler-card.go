package struc2frm

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CardH is an example http handler func
func CardH(w http.ResponseWriter, req *http.Request) {

	w.Header().Add("Content-Type", "text/html")

	s2f := New()
	s2f.ShowHeadline = true
	s2f.FocusFirstError = true
	s2f.SetOptions("department", []string{"ub", "fm"}, []string{"UB", "FM"})
	s2f.SetOptions("items2", []string{"anton", "berta", "caesar", "dora"}, []string{"Anton", "Berta", "Caesar", "Dora"})
	// s2f.Method = "GET"

	// init values - non-multiple
	frm := entryForm{
		// HashKey:    time.Now().Format("2006-01-02"),
		Department: "ub",
		Groups:     2,
		DateLayout: "[2006-01-02]",
		Date:       time.Now().Format("2006-01-02"),
		Time:       time.Now().Format("15:04"),
		CheckThis:  true,
	}

	dept := s2f.DefaultOptionKey("department")
	frm.Items = strings.Join(itemGroups[dept], "\n")

	errs, _ := frm.Validate()
	s2f.AddErrors(errs) // add errors only for a populated form

	fmt.Fprintf(
		w,
		defaultHTML,
		s2f.Card(frm),
		"",
	)

}
