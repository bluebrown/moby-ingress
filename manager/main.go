package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {

	templatPath := flag.String("template", "./haproxy.cfg.template", "path to template inside the container")
	flag.Parse()

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	parts := strings.Split(*templatPath, "/")
	name := parts[len(parts)-1]
	t := template.Must(template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		services, err := cli.ServiceList(ctx, types.ServiceListOptions{})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		t.Execute(w, services)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))

}
