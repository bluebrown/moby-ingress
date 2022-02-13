package haproxyconfig

import (
	"bytes"
	"log"
	"strings"
	"text/template"

	"github.com/bluebrown/labelparser"
	"github.com/bluebrown/moby-ingress/pkg/container"
	"github.com/mitchellh/mapstructure"
)

func DecodeServices(conf *ConfigData, svc []container.Service) {
	for _, s := range svc {
		if s.Labels["ingress.role"] == "controller" {
			continue
		}
		backendName := s.Name
		svcLabels := map[string]interface{}{}
		labelparser.Parse(s.Labels, &svcLabels)
		if ingressLabels, ok := svcLabels["ingress"]; ok {
			be := &Backend{}
			mapstructure.Decode(ingressLabels, be)
			be.Replicas = s.Replicas
			be.EndpointMode = s.EndpointMode
			DecodeFrontendSnippets(conf, be, &backendName)
			conf.Backend[backendName] = be
		}
	}
}

func DecodeFrontendSnippets(conf *ConfigData, be *Backend, backendName *string) {
	for name, snippet := range be.Frontend {
		if _, ok := conf.Frontend[name]; ok {
			tmpl, err := template.New("backend").Parse(snippet)
			if err != nil {
				log.Printf("[WARN] skipping backend %s, failed to parse backend snippet: %v", be.Backend, err)
				continue
			}
			data := new(bytes.Buffer)
			tmpl.Execute(data, struct{ Name string }{Name: *backendName})
			stringData := data.String()
			if !strings.HasSuffix(stringData, "\n") {
				stringData += "\n"
			}
			conf.Frontend[name] += stringData
		}
	}
}
