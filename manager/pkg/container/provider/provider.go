package provider

import (
	"github.com/bluebrown/moby-ingress/pkg/container"
)

type ServiceListOptions struct {
	Filters map[string][]string
}

type Provider interface {
	Services(opts *ServiceListOptions) ([]container.Service, error)
}
