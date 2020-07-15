package struc2frm

import (
	"testing"
)

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
