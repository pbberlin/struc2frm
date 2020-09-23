package struc2frm

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

type userDataFormT struct {
	Gender      string `json:"gender"        form:"subtype=select,size='1'"`
	Decade      string `json:"decade"        form:"subtype=select,size='1',label='Decade of birth'"`
	Culture     string `json:"culture"       form:"subtype=select,size='1',label='Cultural background',suffix='as influence on taste'"`
	Ownership   string `json:"ownership"     form:"subtype=select,size='1',label='Home ownership',suffix=''"`
	Separator01 string `json:"separator01"   form:"subtype='separator'"`
	Buying      bool   `json:"buying"        form:"label='Home buying experience'"`
	Recent      string `json:"recent"        form:"subtype=select,size='1',label='Last purchase',suffix=''"`

	Status string `json:"status"  form:"-"` // for server communication
	Msg    string `json:"msg"     form:"-"` // for server communication

	DontRender string `json:"dont_render"    form:"-"` // test - not rendered, despite value

}

// Validate checks whether form entries as a whole are "submittable";
// implementation is optional
func (frm userDataFormT) Validate() bool {
	g1 := frm.Gender != "" && frm.Decade != "" && frm.Culture != "" && frm.Ownership != ""
	g2 := frm.Buying && frm.Recent != "" || !frm.Buying
	return g1 && g2
}

func TestCardView(t *testing.T) {

	ud := userDataFormT{
		Gender:    "m",
		Decade:    "1960s",
		Culture:   "european",
		Ownership: "renter",

		DontRender: "should not appear",
	}

	s2f := New()

	s2f.SkipEmpty = true

	s2f.AddOptions("gender", []string{"", "f", "m", "o"},
		[]string{"Please choose", "female", "male", "third"})
	s2f.AddOptions("decade", []string{"", "before1960", "1960s", "1970s", "1980s", "1990s", "2000s", "2010s-"},
		[]string{"Please choose", "before 1960", "1960-69", "1970-79", "1980-89", "1990-99", "2000-09", "2010-"})
	s2f.AddOptions("culture", []string{"", "european", "near-east", "asian", "indian", "african"},
		[]string{"Please choose", "European", "Near East", "Asian", "Indian", "African"})
	s2f.AddOptions("ownership", []string{"", "owner", "renter"},
		[]string{"Please choose", "Homeowner", "Renter"})
	s2f.AddOptions("recent", []string{"", "2020s", "2010s", "2000s", "1990s", "1980s", "before1980"},
		[]string{"Please choose", "2020s", "2010s", "2000s", "1990s", "1980s", "before"})

	got := s2f.Card(ud)

	// Check the response body
	want := `<div class='struc2frm struc2frm-%v'>
<ul>
	<li>
	<span style='display: inline-block; width: 40%%;' >Gender:</span>  %v  
	</li>
	<li>
	<span style='display: inline-block; width: 40%%;' >Decade of birth:</span>  %v  
	</li>
	<li>
	<span style='display: inline-block; width: 40%%;' >Cultural background:</span>  %v  
<span class='postlabel' >as influence on taste</span>	</li>
	<li>
	<span style='display: inline-block; width: 40%%;' >Home ownership:</span>  %v  
	</li>
	<div class='separator'></div>
	<li>
	<span style='display: inline-block; width: 40%%;' >Home buying experience:</span>  false  
	</li>
</ul>
</div><!-- </div class='struc2frm'... -->
`

	want = fmt.Sprintf(
		want,
		s2f.InstanceID,
		"male",     // ud.Gender is "m"
		"1960-69",  // ud.Decade is 1960s
		"European", // ud.Culture is "european"
		"Renter",   //   ud.Ownership is "renter"
	)
	if !strings.Contains(string(got), want) {
		t.Errorf("got != want")
		ioutil.WriteFile("tmp-cardview_want.html", []byte(want), 0777)
		ioutil.WriteFile("tmp-cardview_got.html", []byte(got), 0777)
	}
}
