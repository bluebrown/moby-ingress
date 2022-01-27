package main

import (
	"flag"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/docker/docker/client"
)

func main() {
	// the initial template
	templatPath := flag.String("template", "./templates/haproxy.cfg.template", "path to template inside the container")
	flag.Parse()

	// if the template is not parsable, panic and exit
	configTemplate := template.Must(template.New(TemplateName(*templatPath)).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath))

	// initialize a new docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// run the server
	RunServer(cli, configTemplate, *templatPath)
}
