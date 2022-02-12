package decode

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/bluebrown/labelparser"
	"github.com/bluebrown/moby-ingress/pkg/haproxy"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
)

// create the config data from the docker client
func DecodeConfigData(ctx context.Context, cli *client.Client) (haproxy.ConfigData, error) {
	dockerInfo, err := cli.Info(ctx)
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}

	// check docker runs in swarm mode
	isSwarmMode := dockerInfo.Swarm.LocalNodeState == swarm.LocalNodeStateActive

	// initialize the a new config struct
	conf := haproxy.ConfigData{}
	conf.Backend = make(map[string]haproxy.Backend)

	// get the labels from the manager itself, as it holds global
	// and frontend configs
	managerContainerJson, err := cli.ContainerInspect(ctx, os.Getenv("HOSTNAME"))
	if err != nil {
		return conf, err
	}
	// create initial config data from the manager labels
	DecodeManagerInfo(&conf, managerContainerJson)

	// if swarm mode is enabled, get the services and merge their configs
	if isSwarmMode {
		// swarm services
		opts := moby.ServiceListOptions{}
		// if an ingress class was specified, use it to filter the services
		if conf.IngressClass != "" {
			opts.Filters = filters.NewArgs(filters.KeyValuePair{Key: "label", Value: "ingress.class=" + conf.IngressClass})
		}
		// get the list of swarm services
		services, err := cli.ServiceList(ctx, opts)
		if err != nil {
			return conf, err
		}
		// parse the services
		DecodeSwarmServices(&conf, services)
	}

	// decode compose services
	composeOpts := moby.ContainerListOptions{}
	composeOpts.Filters = filters.NewArgs(filters.Arg("label", "com.docker.compose.project"))

	// if an ingress class was specified, use it to filter the services
	if conf.IngressClass != "" {
		composeOpts.Filters.Add("label", "ingress.class="+conf.IngressClass)
	}
	containers, err := cli.ContainerList(ctx, composeOpts)
	if err != nil {
		return conf, err
	}

	ss := ConvertComposeContainersToSwarmServices(containers)
	DecodeSwarmServices(&conf, ss)

	// return the final config struct
	return conf, nil
}

// parse the manager labels into the config data
func DecodeManagerInfo(conf *haproxy.ConfigData, info moby.ContainerJSON) {
	c := map[string]interface{}{}
	labelparser.Parse(info.Config.Labels, &c)

	// if it contains ingress rules decode them
	if val, ok := c["ingress"]; ok {
		mapstructure.Decode(val, &conf)
	}
}

// iterate through all services to merge their configs into
// the config data, created from the manager labels
func DecodeSwarmServices(conf *haproxy.ConfigData, services []swarm.Service) {
	for _, svc := range services {
		log.Printf("[DEBUG] Parsing service %s...", svc.Spec.Name)
		DecodeBackend(conf, svc)
	}
}

func DecodeBackend(conf *haproxy.ConfigData, svc swarm.Service) {
	backendName := svc.Spec.Name
	// get the service labels
	svcLabels := map[string]interface{}{}
	labelparser.Parse(svc.Spec.Labels, &svcLabels)

	// if it contains ingres rules decode them
	if ingressLabels, ok := svcLabels["ingress"]; ok {
		// get the backend config
		be := &haproxy.Backend{}
		mapstructure.Decode(ingressLabels, be)

		// replicas are used from the service spec
		be.Replicas = *svc.Spec.Mode.Replicated.Replicas

		// if there are config snippets, merge them into the corresponding frontends
		// from the manager labels
		DecodeFrontendSnippets(conf, be, backendName)

		// add the endpointmode
		be.EndpointMode = svc.Endpoint.Spec.Mode

		// add the backend to the config data
		conf.Backend[backendName] = *be
		log.Printf("[INFO] using backend '%s' with port %s and %d replicas", backendName, be.Port, be.Replicas)
	}
}

func DecodeFrontendSnippets(conf *haproxy.ConfigData, be *haproxy.Backend, backendName string) {
	for name, snippet := range be.Frontend {
		if _, ok := conf.Frontend[name]; ok {
			// try to parse the snippet as template
			tmpl, err := template.New("backend").Parse(snippet)
			if err != nil {
				log.Printf("[WARN] skipping backend %s, failed to parse backend snippet: %v", be.Backend, err)
				continue
			}
			// execute the template and pass the backend name to it
			data := new(bytes.Buffer)
			tmpl.Execute(data, struct{ Name string }{Name: backendName})
			// make sure the snippets ends with a new line
			stringData := data.String()
			if !strings.HasSuffix(stringData, "\n") {
				stringData += "\n"
			}
			// append snippet to the corresponding frontend
			conf.Frontend[name] += stringData
		} else {
			log.Printf("[WARN] skipping frontend %s for backend %s, frontend name not found in manager labels", name, be.Backend)
		}
	}

}

func ConvertComposeContainersToSwarmServices(containers []moby.Container) []swarm.Service {
	services := map[string]swarm.Service{}
	for _, container := range containers {
		log.Printf("[DEBUG] Parsing container %s...", container.ID)

		// skip manager and loadbalancer
		role := container.Labels["ingress.role"]
		if role == "manager" || role == "loadbalancer" {
			log.Printf("[DEBUG] Skipping container %s, it is a %s", container.ID, role)
			continue
		}

		// get the compose service name
		svcName := container.Labels["com.docker.compose.service"]
		// if the service is not in the list, add it
		if _, ok := services[svcName]; !ok {
			first := uint64(1)
			services[svcName] = swarm.Service{
				Endpoint: swarm.Endpoint{
					Spec: swarm.EndpointSpec{
						Mode: swarm.ResolutionModeDNSRR,
					},
				},
				Spec: swarm.ServiceSpec{
					Annotations: swarm.Annotations{
						Name:   svcName,
						Labels: container.Labels,
					},
					Mode: swarm.ServiceMode{
						Replicated: &swarm.ReplicatedService{
							Replicas: &first,
						},
					},
				},
			}
		} else {
			// otherwise, increase the replicas
			*services[svcName].Spec.Mode.Replicated.Replicas++
		}
	}
	ss := []swarm.Service{}
	for _, svc := range services {
		log.Printf("[DEBUG] Converted compose service to swarm service with name %s and %d replicas\n", svc.Spec.Annotations.Name, *svc.Spec.Mode.Replicated.Replicas)
		ss = append(ss, svc)
	}
	return ss
}
