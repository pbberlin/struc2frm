package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/pbberlin/struc2frm"
)

func main() {

	rand.Seed(time.Now().UTC().UnixNano())
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	log.SetFlags(log.Lshortfile | log.Ltime)

	struc2frm.CfgLoad()
	pfx := struc2frm.CfgGet().URLPathPrefix

	mux1 := http.NewServeMux() // base router

	mux1.HandleFunc("/", struc2frm.FormH)
	if pfx != "" {
		mux1.HandleFunc("/"+pfx, struc2frm.FormH)
		mux1.HandleFunc("/"+pfx+"/", struc2frm.FormH)
	}

	mux1.HandleFunc("/file-upload", struc2frm.FileUploadH)
	if pfx != "" {
		mux1.HandleFunc("/"+pfx+"/file-upload", struc2frm.FileUploadH)
		mux1.HandleFunc("/"+pfx+"/file-upload/", struc2frm.FileUploadH)
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

	IPPort := fmt.Sprintf("%v:%v", struc2frm.CfgGet().BindHost, struc2frm.CfgGet().BindSocket)
	log.Printf("starting http server at %v ... ", IPPort)
	log.Printf("==========================")
	log.Printf("  ")

	log.Fatal(http.ListenAndServe(IPPort, mux4))

}
