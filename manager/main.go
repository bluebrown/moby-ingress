package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
)

func main() {
	templatPath := flag.String("template", "./templates/haproxy.cfg.template", "path to template inside the container")
	flag.Parse()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	bgCtx := context.Background()
	ctx, cancel := context.WithCancel(bgCtx)

	rc := NewReconciler(cli, time.Minute)
	rc.Reconcile(ctx)

	server := &http.Server{
		Handler: NewMux(rc, *templatPath),
		Addr:    ":8080",
	}

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
