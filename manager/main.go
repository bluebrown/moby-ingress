package main

import (
	"context"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}

	rc := reconcile.NewReconciler(
		cli,
		time.Second*30,
		template.Must(template.New(TemplateName(templatPath)).Funcs(sprig.TxtFuncMap()).ParseFiles(templatPath)),
	)
	rc.Reconcile(ctx)

	mux := NewMux(rc, templatPath)
	server := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[ERROR] listen: %s\n", err)
		}
	}()

	log.Printf("[INFO] listening on %s\n", server.Addr)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	log.Println("[INFO] Stopping server...")

	toCtx, toCancel := context.WithTimeout(ctx, time.Second*5)
	defer toCancel()
	if err := server.Shutdown(toCtx); err != nil {
		log.Fatalf("[ERROR] Server Shutdown Failed:%+v", err)
	}

}

func TemplateName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
