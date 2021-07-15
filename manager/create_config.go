package main

import (
	"bytes"
	"context"
	"os"
	"text/template"

	"github.com/bluebrown/labelparser"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
)

func CreateConfigData(ctx context.Context, cli client.Client) ConfigData {
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
