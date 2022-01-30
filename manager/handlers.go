package main

import (
	"io/ioutil"
	"net/http"
	"text/template"

	"github.com/Masterminds/sprig"
)

func handleGetConfig(recon ReconciliationBroker, templ *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		hash := r.Header.Get("Config-Hash")
		select {
		// if the client closed the connection, return
		case <-ctx.Done():
			return
		// if the reconciliation broker has a new config, return it
		case hpConf := <-recon.NextValue(ctx, hash):
			// set the hash header
			w.Header().Add("Config-Hash", hpConf.Hash)
			// return json if accept header is application/json
			if r.Header.Get("Accept") == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.Write(hpConf.JSON)
				return
			}
			// otherwise render the template
			w.Write(hpConf.File)
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
