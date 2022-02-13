package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/bluebrown/moby-ingress/pkg/container/provider"
	"github.com/bluebrown/moby-ingress/pkg/haproxy"
	"github.com/bluebrown/moby-ingress/pkg/haproxyconfig"
	"github.com/bluebrown/moby-ingress/pkg/util"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

func main() {
	// parse the template
	tpl := template.Must(template.New(util.TemplateName(templatPath)).Funcs(sprig.TxtFuncMap()).ParseFiles(templatPath))

	// init the docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}

	// create a context that is cancelled when the program is interrupted
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// get the controller labels
	var rawControllerLabels map[string]string
	if mock {
		rawControllerLabels = util.MockControllerLabels()
	} else {
		controllerInfo, err := cli.ContainerInspect(ctx, os.Getenv("HOSTNAME"))
		if err != nil {
			log.Fatalf("[ERROR] %s", err)
		}
		rawControllerLabels = controllerInfo.Config.Labels
	}

	// register the docker provider
	providers := []provider.Provider{
		provider.DockerProvider{
			Context: ctx,
			Client:  cli,
		},
	}

	// if docker is in swarm mode, register the swarm provider
	dockerInfo, err := cli.Info(ctx)
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}

	if dockerInfo.Swarm.LocalNodeState == swarm.LocalNodeStateActive {
		providers = append(providers, provider.SwarmProvider{
			Context: ctx,
			Client:  cli,
		})
	}

	// create a new haproxy manager
	hp := haproxy.New("haproxy.cfg")

	// start the loop to get config file updates
	ch := haproxyconfig.Loop(ctx, tpl, rawControllerLabels, providers)

	// get the initial config
	hc := <-ch
	err = os.WriteFile(hp.ConfigPath, hc.File, 0644)
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}

	// validate the config
	err = hp.Validate()
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}

	// start the haproxy process
	log.Println("[INFO] Starting haproxy")
	hp.Start()

	for hc = range ch {
		log.Println("[INFO] changes detected, reloading haproxy")
		err := os.WriteFile("haproxy.cfg", hc.File, 0644)
		if err != nil {
			log.Fatalf("[ERROR] %s", err)
		}
		err = hp.Validate()
		if err != nil {
			log.Printf("[WARN] new config is invalid, skipping reload:\n%s\n", string(hc.File))
		} else {
			hp.Reload()
		}
	}

	log.Println("[INFO] Exiting")
	hp.Stop()

}
