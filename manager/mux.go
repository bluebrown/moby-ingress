package main

import (
	"log"
	"net/http"
)

func NewMux(reconciler ReconciliationBroker, templatePath string) *http.ServeMux {
	// initialize a the server
	mux := http.NewServeMux()

	// initialize the handlers
	confHandler := handleGetConfig(reconciler)
	putTemplateHandler := handlePutTemplate(reconciler)

	// mux the handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL)
		// get the config
		if r.Method == "GET" {
			confHandler(w, r)
			return
		}
		// patch the config
		if r.Method == "PUT" {
			putTemplateHandler(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	return mux

}
