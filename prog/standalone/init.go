package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/logutils"
)

var (
	templatPath string
	logLevel    string
	mock        bool
)

func init() {
	flag.StringVar(&logLevel, "log-level", "warn", "log level")
	flag.StringVar(&templatPath, "template", "./assets/haproxy.cfg.template", "path to template inside the container")
	flag.BoolVar(&mock, "mock", false, "mock mode")

	flag.Parse()

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(strings.ToUpper(logLevel)),
		Writer:   os.Stderr,
	}

	log.SetOutput(filter)

	if logLevel == "debug" {
		log.SetFlags(log.Ltime | log.Llongfile)
	}
}
