package struc2frm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"time"
)

// var tokenSaltNotWorking = GeneratePassword(22) // not interoperational between multiple instances of go-questionnaire, transferrer, generator

// set timezone to a constant - this is important for client-server talks, e.g. appengine frankfurt runs on different zone
var fixedLocation = time.FixedZone("UTC_-2", -2*60*60)

// tok rounds time to hours
// and computes a hash from it
func tok(hoursOffset int, salt string) string {
	hasher := sha256.New()
	_, err := io.WriteString(hasher, salt)
	if err != nil {
		log.Printf("Error writing salt to hasher: %v", err)
	}
	t := time.Now().In(fixedLocation)
	if hoursOffset != 0 {
		t = t.Add(time.Duration(hoursOffset) * time.Hour)
	}
	// log.Printf("token time: %v", t.Format("02.01.2006 15"))
	_, err = io.WriteString(hasher, t.Format("02.01.2006 15"))
	if err != nil {
		log.Printf("Error writing date-hour to hasher: %v", err)
	}
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}

// FormToken returns a form token.
// User independent.
// Should we add the user name into the hashed base?
func (s2f *s2FT) FormToken() string {
	return tok(0, s2f.Salt)
}

// ValidateFormToken checks tokens
// against current hour - back to n previous hours.
// Plus one more for bounding glitches / border crossing
// when the rounding jumps from 12:59 to 13:00.
// i.e.
// FormTimeout := 2
// lower bound := -4
// => Checking token against current hour, previous hour, second previous hour, third previous hour
func (s2f *s2FT) ValidateFormToken(arg string) error {
	lowerBound := s2f.FormTimeout*-1 - 1
	for i := 0; i >= lowerBound; i-- {
		if arg == tok(i, s2f.Salt) {
			return nil
		}
	}
	if arg == tok(1, s2f.Salt) {
		return nil
	}
	return fmt.Errorf("form token was not issued within the last two hours. \nPlease re-login")
}
