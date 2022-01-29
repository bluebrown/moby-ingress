package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"text/template"

	"github.com/Masterminds/sprig"
)

func handleGetConfig(recon *Reconciler, templ *template.Template) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ch := recon.Subscribe()
		outcome := <-ch
		recon.Unsubscribe(ch)

		if outcome.Error != nil {
			log.Printf("error getting config: %s", outcome.Error)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// return json if accept header is application/json
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(outcome.Config.ToJsonBytes())
			return
		}

		// otherwise render the template
		err := templ.Execute(w, outcome.Config)

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
