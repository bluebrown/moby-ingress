package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/docker/docker/client"
)

func main() {
	// the initial template
	templatPath := flag.String("template", "./haproxy.cfg.template", "path to template inside the container")
	flag.Parse()

	// use the last part of the path as template name
	parts := strings.Split(*templatPath, "/")
	name := parts[len(parts)-1]

	// if the template is not parsable, panic and exit
	t := template.Must(template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath))

	// initialize a new docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// this endpoint is used by the loadbalancer to fetch the current config periodically
	// TODO: implement content negotiation to return either text or json
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conf := CreateConfigData(r.Context(), *cli)
		w.Header().Set("Content-Type", "text/plain")
		err := t.Execute(w, conf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// this endpoint returns the config as json object. It is not really used
	// but may be useful for certian use cases, i.e someone wants to use the
	// config in different ways
	http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		conf := CreateConfigData(r.Context(), *cli)
		w.Header().Set("Content-Type", "application/json")
		w.Write(conf.ToJsonBytes())
	})

	// this handler allows to update the template at runtime
	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// read the new template from the request body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// get a backup of the current template
		save, err := ioutil.ReadFile(*templatPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// write the new template to the file
		if err := ioutil.WriteFile(*templatPath, body, 0644); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// try to parse the new template
		// if its not valid, restore the old one, and return an error
		// TODO: we don't need to write it first and then parse it, also the old template is already
		// parsed in memory, so we don't need to parse it again
		t, err = template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath)
		if err != nil {
			ioutil.WriteFile(*templatPath, save, 0644)
			w.WriteHeader(http.StatusBadRequest)
			t = template.Must(template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath))
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	// start the server
	log.Fatal(http.ListenAndServe(":8080", nil))

}
