package provider

import (
	"context"
	"strings"

	"github.com/bluebrown/moby-ingress/pkg/container"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type SwarmProvider struct {
	Context context.Context
	Client  *client.Client
}

func (s SwarmProvider) Services(opts *ServiceListOptions) ([]container.Service, error) {

	mobyOpts := moby.ServiceListOptions{}

	if len(opts.Filters) > 0 {
		mobyOpts.Filters = DockerFilter(opts)
	}

	services, err := s.Client.ServiceList(s.Context, mobyOpts)
	if err != nil {
		return nil, err
	}

	var result []container.Service

	for _, swarmService := range services {
		result = append(result, container.Service{
			Name:         swarmService.Spec.Name,
			EndpointMode: string(swarmService.Endpoint.Spec.Mode),
			Replicas:     *swarmService.Spec.Mode.Replicated.Replicas,
			Labels:       swarmService.Spec.Labels,
		})
	}

	return result, nil

}

type DockerProvider struct {
	Context context.Context
	Client  *client.Client
}

func (d DockerProvider) Services(opts *ServiceListOptions) ([]container.Service, error) {

	mobyOpts := moby.ContainerListOptions{}

	if len(opts.Filters) > 0 {
		mobyOpts.Filters = DockerFilter(opts)
	}

	containers, err := d.Client.ContainerList(d.Context, mobyOpts)
	if err != nil {
		return nil, err
	}

	var result []container.Service
	temp := make(map[string]container.Service)

	for _, cnt := range containers {
		if cnt.Labels["com.docker.swarm.service.name"] != "" {
			continue
		}

		networkAlias := cnt.Labels["ingress.network-alias"]
		composeSvcLabel := cnt.Labels["com.docker.compose.service"]

		name := cnt.Names[0]

		if networkAlias != "" {
			name = networkAlias
		} else if composeSvcLabel != "" {
			name = composeSvcLabel
		}

		if entry, ok := temp[name]; ok {
			entry.Replicas++
			temp[name] = entry
		} else {
			temp[name] = container.Service{
				Name:         strings.TrimPrefix(name, "/"),
				Labels:       cnt.Labels,
				Replicas:     1,
				EndpointMode: "dnsrr",
			}
		}
	}

	for _, r := range temp {
		result = append(result, r)
	}

	return result, nil
}

func DockerFilter(opts *ServiceListOptions) filters.Args {
	filter := filters.NewArgs()
	for key, vals := range opts.Filters {
		for _, val := range vals {
			filter.Add(key, val)
		}
	}
	return filter
}
