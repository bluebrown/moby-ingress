package main

import (
	"log"
	"net/http"
	"text/template"

	"github.com/docker/docker/client"
)

func RunServer(cli *client.Client, configTemplate *template.Template, templatePath string) {
	// initialize the handlers
	confHandler := handleGetConfig(cli, configTemplate)
	patchHandler := handlePatchConfig(templatePath, configTemplate)

	// mux the handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	// start the server
	log.Println("Starting server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
