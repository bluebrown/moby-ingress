package main

import (
	"context"
	"flag"
	"html/template"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {

	templatPath := flag.String("template", "/src/haproxy.cfg.template", "path to template inside the container")
	flag.Parse()

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	t := template.Must(template.ParseFiles(*templatPath))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		services, err := cli.ServiceList(ctx, types.ServiceListOptions{})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/text")
		w.WriteHeader(http.StatusOK)
		t.Execute(w, services)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))

}
