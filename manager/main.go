package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/bluebrown/moby-ingress/pkg/reconcile"
	"github.com/docker/docker/client"
)

func main() {
	templatPath := flag.String("template", "./templates/haproxy.cfg.template", "path to template inside the container")
	flag.Parse()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	rc := reconcile.NewReconciler(cli, time.Second*30, template.Must(template.New(TemplateName(*templatPath)).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatPath)))
	rc.Reconcile(ctx)
	mux := NewMux(rc, *templatPath)
	server := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ERROR: listen: %s\n", err)
		}
	}()

	log.Printf("listening on %s\n", server.Addr)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done
	cancel()

	log.Println("Stopping server...")

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

}

func TemplateName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
