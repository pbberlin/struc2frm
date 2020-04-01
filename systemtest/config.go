package main

import (
	"encoding/json"
	"log"
	"os"
)

// ConfigT holds the application config
type ConfigT struct {
	AppName       string `json:"app_name"`       // with case, i.e. 'My App'
	URLPathPrefix string `json:"urlpath_prefix"` // lower case - no slashes, i.e. 'myapp'
	BindHost      string `json:"bind_host"`      // "0.0.0.0"
	BindSocket    int    `json:"bind_socket"`    // 8081 or 80

	ItemGroups map[string][]string `json:"itemgroups"`
}

var cfgS *ConfigT // package variable 'singleton' - needs to be an allocated struct - to hold pointer receiver-re-assignment

func cfgLoad() {
	f, err := os.Open("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(f)
	tempCfg := ConfigT{}
	err = decoder.Decode(&tempCfg)
	if err != nil {
		log.Fatal(err)
	}
	if tempCfg.AppName == "" {
		log.Fatal("Config underspecified; at least app_name should be set")
	}
	//
	cfgS = &tempCfg // replace pointer in one go - should be threadsafe
	bts, err := json.MarshalIndent(cfgS, " ", "\t")
	if err != nil {
		log.Fatalf("Could not re-marshal config: %v", err)
	}
	if len(bts) > 700 {
		bts = bts[:700]
	}
	log.Printf("\n%s\n...config loaded", bts)
}

// CfgGet provides access to the app configuration
func CfgGet() *ConfigT {
	return cfgS
}
