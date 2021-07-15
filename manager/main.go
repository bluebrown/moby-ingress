package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/bluebrown/labelparser"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
)

func createConfigData(ctx context.Context, cli client.Client) ConfigData {
	conf := ConfigData{}
	conf.Backend = make(map[string]Backend)

	services, err := cli.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	info, err := cli.ContainerInspect(ctx, os.Getenv("HOSTNAME"))
	if err != nil {
		panic(err)
	}
	c := map[string]interface{}{}
	labelparser.Parse(info.Config.Labels, &c)

	if val, ok := c["ingress"]; ok {
		mapstructure.Decode(val, &conf)
	}

	for _, svc := range services {

		BackendName := svc.Spec.Name

		c := map[string]interface{}{}
		labelparser.Parse(svc.Spec.Labels, &c)
		if configMap, ok := c["ingress"]; ok {
			be := Backend{}
			mapstructure.Decode(configMap, &be)
			be.Replicas = *svc.Spec.Mode.Replicated.Replicas
			for name, snippet := range be.Frontend {
				if _, ok := conf.Frontend[name]; ok {
					tmpl := template.Must(template.New("backend").Parse(snippet))
					data := new(bytes.Buffer)
					tmpl.Execute(data, struct{ Name string }{Name: BackendName})
					conf.Frontend[name] += data.String()
				}
			}

			conf.Backend[BackendName] = be
		}
	}

	return conf

}

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
		conf := createConfigData(r.Context(), *cli)
		w.Header().Set("Content-Type", "text/plain")
		t.Execute(w, conf)
	})

	http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		conf := createConfigData(r.Context(), *cli)
		w.Header().Set("Content-Type", "application/json")
		w.Write(conf.toJsonBytes())
	})

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		if err := ioutil.WriteFile(*templatPath, body, 0644); err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		t = template.Must(template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath))
		w.WriteHeader(http.StatusCreated)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))

}

type ConfigData struct {
	Global   string             `json:"global,omitempty"`
	Defaults string             `json:"defaults,omitempty"`
	Frontend map[string]string  `json:"frontend,omitempty"`
	Backend  map[string]Backend `json:"backend,omitempty"`
}

type Backend struct {
	Port     string            `json:"port,omitempty"`
	Replicas uint64            `json:"replicas,omitempty"`
	Frontend map[string]string `json:"-"`
	Backend  string            `json:"backend,omitempty"`
}

func (c ConfigData) toJsonBytes() []byte {
	b, _ := json.Marshal(c)
	return b
}
