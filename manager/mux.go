package main

import (
	"log"
	"net/http"
	"text/template"

	"github.com/Masterminds/sprig"
)

func NewMux(reconciler ReconciliationBroker, templatePath string) *http.ServeMux {
	// initialize a the server
	mux := http.NewServeMux()
	// if the template is not parsable, panic and exit
	configTemplate := template.Must(template.New(TemplateName(templatePath)).Funcs(sprig.TxtFuncMap()).ParseFiles(templatePath))

	// initialize the handlers
	confHandler := handleGetConfig(reconciler, configTemplate)
	patchHandler := handlePatchConfig(templatePath, configTemplate)

	// mux the handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL)
		// get the config
		if r.Method == "GET" {
			confHandler(w, r)
			return
		}
		// patch the config
		if r.Method == "PATCH" {
			patchHandler(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	return mux

}
