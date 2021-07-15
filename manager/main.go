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

	templatPath := flag.String("template", "./haproxy.cfg.template", "path to template inside the container")
	flag.Parse()

	parts := strings.Split(*templatPath, "/")
	name := parts[len(parts)-1]
	t := template.Must(template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath))

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conf := CreateConfigData(r.Context(), *cli)
		w.Header().Set("Content-Type", "text/plain")
		t.Execute(w, conf)
	})

	http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		conf := CreateConfigData(r.Context(), *cli)
		w.Header().Set("Content-Type", "application/json")
		w.Write(conf.ToJsonBytes())
	})

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		save, err := ioutil.ReadFile(*templatPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := ioutil.WriteFile(*templatPath, body, 0644); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		t, err = template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath)
		if err != nil {
			ioutil.WriteFile(*templatPath, save, 0644)
			w.WriteHeader(http.StatusBadRequest)
			t = template.Must(template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath))
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))

}
