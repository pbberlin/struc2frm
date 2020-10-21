package struc2frm

// Validator interface is non mandatory helper interface for form structs;
// it returns error messages suitable for s2f.AddErrors;
// a valid form struct enables further processing;
type Validator interface {
	Validate() (map[string]string, bool)
}
