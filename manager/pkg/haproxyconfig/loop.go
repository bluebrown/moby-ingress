package haproxyconfig

import (
	"context"
	"log"
	"text/template"
	"time"

	"github.com/bluebrown/labelparser"
	"github.com/bluebrown/moby-ingress/pkg/container/provider"
	"github.com/mitchellh/mapstructure"
)

func Loop(ctx context.Context, tpl *template.Template, rawControllerLabels map[string]string, providers []provider.Provider) <-chan *HaproxyConfig {
	ch := make(chan *HaproxyConfig)
	hc := HaproxyConfig{Template: tpl}

	controllerLabels := map[string]interface{}{}
	labelparser.Parse(rawControllerLabels, &controllerLabels)

	// set the filter options to the ingres class of the controller
	opts := &provider.ServiceListOptions{
		Filters: map[string][]string{
			"label": {"ingress.class=" + rawControllerLabels["ingress.class"]},
		},
	}

	defaultConf := &ConfigData{}
	defaultConf.Backend = make(map[string]*Backend)
	if val, ok := controllerLabels["ingress"]; ok {
		mapstructure.Decode(val, defaultConf)
	}

	// a pointer to the config object that is reused to avoid
	// memory allocation and garbage collection
	conf := &ConfigData{}

	// this helper function is used to decode the services
	// and add them to the config
	makeConf := func() {
		// config must be reset before each call
		// otherwise the config will be appended to
		reset(defaultConf, conf)
		// get the services
		for _, p := range providers {
			services, err := p.Services(opts)
			if err != nil {
				log.Printf("[ERROR] %s", err)
				continue
			}
			DecodeServices(conf, services)
		}
		// set the config data to the haproxy config
		hc.Set(conf)
	}

	go func() {
		// create the initial config and store the hash
		makeConf()
		currentHash := hc.Hash
		ch <- &hc
		// loop until the program is interrupted
		log.Println("[DEBUG] starting loop...")
		for {
			select {
			case <-ctx.Done():
				log.Println("[INFO] Exiting...")
				close(ch)
				return
			case <-time.After(time.Second * 30):
				log.Println("[DEBUG] Checking for changes...")
				makeConf()
				if currentHash != hc.Hash {
					currentHash = hc.Hash
					ch <- &hc
				}
			}
		}
	}()

	return ch
}

func reset(src, dst *ConfigData) {
	dst.Backend = make(map[string]*Backend)
	dst.Frontend = make(map[string]string, len(src.Frontend))
	dst.Global = src.Global
	dst.Defaults = src.Defaults
	for k, v := range src.Frontend {
		dst.Frontend[k] = v
	}
}
