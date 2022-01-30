package decode

import (
	"testing"

	"github.com/bluebrown/moby-ingress/pkg/haproxy"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
)

func TestManagerLabels(t *testing.T) {
	conf := haproxy.ConfigData{}
	conf.Backend = make(map[string]haproxy.Backend)

	info := moby.ContainerJSON{
		Config: &container.Config{
			Labels: map[string]string{
				"ingress.class":            "haproxy",
				"ingress.global":           "spread-checks 15\n",
				"ingress.defaults":         "timeout connect 5s\n",
				"ingress.frontend.default": "bind *:80\n",
			},
		},
	}

	DecodeManagerInfo(&conf, info)

	if conf.Global != "spread-checks 15\n" {
		t.Errorf("Expected 'spread-checks 15\n', got '%s'", conf.Global)
	}

	if conf.Defaults != "timeout connect 5s\n" {
		t.Errorf("Expected 'timeout connect 5s\n', got '%s'", conf.Defaults)
	}

	if conf.Frontend["default"] != "bind *:80\n" {
		t.Errorf("Expected 'bind *:80\n', got '%s'", conf.Frontend["default"])
	}
}

func TestSwarmServices(t *testing.T) {
	conf := haproxy.ConfigData{}
	conf.Backend = make(map[string]haproxy.Backend)
	conf.Frontend = make(map[string]string)
	conf.Frontend["default"] = ""

	svcs := []swarm.Service{FakeService("test-app", 1, map[string]string{
		"ingress.class":            "haproxy",
		"ingress.frontend.default": "default_backend {{.Name}}",
		"ingress.backend":          "http-response set-header X-Server %s",
	})}

	DecodeSwarmServices(&conf, svcs)

	if len(conf.Backend) != 1 {
		t.Errorf("Expected 1 backends, got %d", len(conf.Backend))
	}

	if _, ok := conf.Backend["test-app"]; !ok {
		t.Errorf("Expected backend 'test-app', not found")
	}

	if conf.Backend["test-app"].Replicas != 1 {
		t.Errorf("Expected 1 replicas, got %d", conf.Backend["test-app"].Replicas)
	}

	if conf.Frontend["default"] != "default_backend test-app\n" {
		t.Errorf("Expected 'default_backend test-app', got '%s'", conf.Frontend["default"])
	}

	if conf.Backend["test-app"].Backend != "http-response set-header X-Server %s" {
		t.Errorf("Expected 'http-response set-header X-Server %s', got '%s'", conf.Backend["test-app"].Backend, conf.Backend["test-app"].Backend)
	}
}

func FakeService(name string, replicas uint64, labels map[string]string) swarm.Service {
	return swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name:   name,
				Labels: labels,
			},
			Mode: swarm.ServiceMode{
				Replicated: &swarm.ReplicatedService{
					Replicas: &replicas,
				},
			},
		},
	}
}
