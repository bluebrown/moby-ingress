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

// create the config data from the docker client
// TODO: implement some caching mechanism to avoid calling the docker client
// every time a request is made by the loadbalancer
func CreateConfigData(ctx context.Context, cli client.Client) ConfigData {
	// initialize the a new config struct
	conf := ConfigData{}
	conf.Backend = make(map[string]Backend)

	// get the list of swarm services
	// TODO: implement a way to handle compose services as well
	services, err := cli.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		// we should probably not panic here
		panic(err)
	}

	// get the labels from the manager itself, as it holds global
	// and frontend configs
	info, err := cli.ContainerInspect(ctx, os.Getenv("HOSTNAME"))
	if err != nil {
		panic(err)
	}

	// parse the manager labels labels
	c := map[string]interface{}{}
	labelparser.Parse(info.Config.Labels, &c)

	// if it contains ingress rules decode them
	if val, ok := c["ingress"]; ok {
		mapstructure.Decode(val, &conf)
	}

	// iterate through all services to merge their configs into
	// the config data, created from the manager labels
	for _, svc := range services {

		// the backend name is the service name
		BackendName := svc.Spec.Name

		// get the service labels
		c := map[string]interface{}{}
		labelparser.Parse(svc.Spec.Labels, &c)

		// if it contains ingres rules decode them
		if configMap, ok := c["ingress"]; ok {
			// get the backend config
			be := Backend{}
			mapstructure.Decode(configMap, &be)

			// replicas are used from the service spec
			be.Replicas = *svc.Spec.Mode.Replicated.Replicas

			// if there are config snippets, merge them into the corresponding frontends
			// from the manager labels
			for name, snippet := range be.Frontend {

				// TODO:: inform the user if the frontend name is not defined in the manager labels
				if _, ok := conf.Frontend[name]; ok {

					// TODO: we should not panic here, but rather log the error
					tmpl := template.Must(template.New("backend").Parse(snippet))
					data := new(bytes.Buffer)
					tmpl.Execute(data, struct{ Name string }{Name: BackendName})
					conf.Frontend[name] += data.String()

				}

			}

			// add the backend to the config data
			conf.Backend[BackendName] = be

		}
	}
	// return the final config struct
	return conf
}
