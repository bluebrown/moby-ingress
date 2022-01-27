package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/docker/docker/client"
)

func handleGetConfig(cli *client.Client, configTemplate *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conf, err := CreateConfigData(r.Context(), cli)
		if err != nil {
			log.Printf("ERROR: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(conf.ToJsonBytes())
			return
		}

		err = configTemplate.Execute(w, conf)
		if err != nil {
			log.Printf("ERROR: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}

func handlePatchConfig(templatePath string, configTemplate *template.Template) http.HandlerFunc {
	templateName := TemplateName(templatePath)
	return func(w http.ResponseWriter, r *http.Request) {
		// read the new template from the request body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// try to parse the new template
		newT, err := template.New(templateName).Funcs(sprig.TxtFuncMap()).Parse(string(body))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}

		// write the new template to the file
		if err := ioutil.WriteFile(templatePath, body, 0644); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// if everything went well, store the new template
		// in the configTemplate variable
		configTemplate = newT

	}

}
