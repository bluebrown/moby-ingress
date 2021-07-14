package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

type templateData struct {
	ManagerInfo types.ContainerJSON
	Services    []swarm.Service
}

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
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		info, err := cli.ContainerInspect(ctx, os.Getenv("HOSTNAME"))
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Println(info.Config.Labels)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		t.Execute(w, templateData{info, services})
	})

	log.Fatal(http.ListenAndServe(":8080", nil))

}
