package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
)

func main() {
	configPath := flag.String("config", "/usr/local/etc/haproxy/haproxy.cfg", "path to config inside the container")
	configPath = flag.String("c", "/usr/local/etc/haproxy/haproxy.cfg", "path to config inside the container")
	flag.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		save, err := ioutil.ReadFile(*configPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := ioutil.WriteFile(*configPath, body, 0644); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		validate := exec.Command("haproxy", "-W", "-c", "-f", *configPath)
		valout, err := validate.Output()
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		status := validate.ProcessState.ExitCode()

		fmt.Println("exit code validate:", status)
		fmt.Println()

		if status != 0 {
			ioutil.WriteFile(*configPath, save, 0644)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(valout)
			return

		}

		cmd := exec.Command("kill", "-s", "SIGUSR2", "1")
		stdout, err := cmd.Output()
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Println("exit code kill:", cmd.ProcessState.ExitCode())

		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Println(string(stdout))

		w.WriteHeader(http.StatusCreated)
	})

	log.Fatal(http.ListenAndServe(":7868", nil))

}
