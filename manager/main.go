package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
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
	c := ConfigMap{}
	c = parseLabels(info.Config.Labels, c)
	if val, ok := c["ingress"]; ok {
		mapstructure.Decode(val, &conf)
	}

	for _, svc := range services {
		c := ConfigMap{}
		c = parseLabels(svc.Spec.Labels, c)
		if val, ok := c["ingress"]; ok {
			be := Backend{}
			mapstructure.Decode(val, &be)
			be.Replicas = *svc.Spec.Mode.Replicated.Replicas
			conf.Backend[svc.Spec.Name] = be
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

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func walk(parts []string, val string, conf ConfigMap) {

	// if its the last part
	// assign the value to the key in the map
	if len(parts) == 1 {
		if oldval, ok := conf[parts[0]]; ok {
			switch v := oldval.(type) {
			case ConfigMap:
				v[parts[0]] = val
			}
		} else {
			conf[parts[0]] = val
		}
		return
	}

	// create map at part if not exists
	if _, ok := conf[parts[0]]; !ok {
		c := ConfigMap{}
		conf[parts[0]] = c
	}

	// it can be that c is not a map here but a string
	// in that case i should wrap c in a new map
	switch v := conf[parts[0]].(type) {
	case string:
		c := ConfigMap{}
		conf[parts[0]] = c
		conf[parts[0]].(ConfigMap)[parts[0]] = v
	}

	// use current part conf map as next conf
	c := conf[parts[0]].(ConfigMap)

	walk(parts[1:], val, c)
}

func parseLabels(labels map[string]string, conf ConfigMap) ConfigMap {
	for k, v := range labels {
		walk(strings.Split(k, "."), v, conf)
	}
	return conf
}

type ConfigMap map[string]interface{}

type Backend struct {
	Port     string `json:"port,omitempty"`
	Replicas uint64 `json:"replicas,omitempty"`
	Frontend string `json:"frontend,omitempty"`
	Path     string `json:"path,omitempty"`
	Host     string `json:"host,omitempty"`
	Backend  string `json:"backend,omitempty"`
}

type ConfigData struct {
	Global   string             `json:"global,omitempty"`
	Defaults string             `json:"defaults,omitempty"`
	Frontend map[string]string  `json:"frontend,omitempty"`
	Backend  map[string]Backend `json:"backend,omitempty"`
}

func (c ConfigData) toJsonBytes() []byte {
	b, _ := json.Marshal(c)
	return b
}
