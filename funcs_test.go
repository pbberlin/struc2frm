package struc2frm

import (
	"log"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime)
}

func TestLabelize(t *testing.T) {

	tests := []struct {
		in   string
		want string
	}{
		{
			in:   "bond_fund",
			want: "Bond fund",
		},
		{
			in:   "bondFund",
			want: "Bond fund",
		},
		{
			in:   "bondFUND",
			want: "Bond fund",
		},
		{
			in:   "BONDFund",
			want: "Bondfund",
		},
	}
	for idx, tt := range tests {
		got := labelize(tt.in)
		if got != tt.want {
			t.Errorf("idx%2v: %-16v is %-16v should be %v", idx, tt.in, got, tt.want)
		} else {
			t.Logf("idx%2v: %-16v is %-16v indeed", idx, tt.in, got)
		}
	}
}

func TestContainsComma(t *testing.T) {

	tests := []struct {
		in   string
		want bool
	}{
		{
			in:   `"form:"subtype='select'"`,
			want: false,
		},
		{
			in:   `"form:"subtype='select',accesskey='p',onchange='true',title='loading items'"`,
			want: false,
		},
		{
			in:   `"maxlength='16',size='16',suffix='salt&comma; changes randomness'"`,
			want: false,
		},
		{
			in:   `"accesskey='t',maxlength='16',size='16',pattern='[0-9\\.\\-/]{2&comma;10}',placeholder='2006/01/02 15:04'"`,
			want: false,
		},
		{
			in:   `"maxlength='16',size='16',suffix='salt&comma; changes randomness'"`,
			want: false,
		},
		{
			in:   `"accesskey='t',maxlength='16',size='16',pattern='[0-9\\.\\-/]{2,10}',placeholder='2006/01/02 15:04'"`,
			want: true,
		},
		{
			in:   `"maxlength='16',size='16',suffix='salt, changes randomness'"`,
			want: true,
		},
	}
	for idx, tt := range tests {
		got := commaInsideQuotes(tt.in)
		if got != tt.want {
			t.Errorf("idx%2v: %-16v is %-16v should be %v", idx, tt.in, got, tt.want)
		} else {
			t.Logf("idx%2v: %-16v is %-16v indeed", idx, tt.in, got)
		}
	}
}
