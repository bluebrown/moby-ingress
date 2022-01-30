package main

import (
	"io/ioutil"
	"net/http"

	"github.com/bluebrown/moby-ingress/pkg/reconcile"
)

func handleGetConfig(recon reconcile.ReconciliationBroker) http.HandlerFunc {
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

func handlePutTemplate(recon reconcile.ReconciliationBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// read the new template from the request body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// set the new template
		err = recon.SetTemplate(string(body))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

}
