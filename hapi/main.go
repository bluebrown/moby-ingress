package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
)

var (
	myClient        = &http.Client{Timeout: 10 * time.Second}
	configPath      string
	templatePath    string
	managerEndpoint string
	t               *template.Template
)

func main() {

	configPath = getenv("CONFIG_PATH", "/usr/local/etc/haproxy/haproxy.cfg")
	templatePath = getenv("TEMPLATE_PATH", "/usr/local/etc/haproxy/haproxy.templ.cfg")
	managerEndpoint = getenv("MANAGER_ENDPOINT", "http://manager:8080/json")

	si := getenv("SCRAPE_INTERVAL", "30")
	scrapeInterval, err := strconv.Atoi(si)
	if err != nil {
		fmt.Println(err)
		scrapeInterval = 30
	}

	parts := strings.Split(templatePath, "/")
	name := parts[len(parts)-1]
	t = template.Must(template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(templatePath))

	tickspeed := time.Second * time.Duration(scrapeInterval)
	bg := context.Background()
	ctx, cancel := context.WithCancel(bg)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(tickspeed):
				r, err := myClient.Get(managerEndpoint)
				if err != nil {
					fmt.Printf("could not fetch from %s: %v", managerEndpoint, err)
					continue
				}

				if r.StatusCode > 300 {
					fmt.Printf("could not fetch from %s: response status: %v", managerEndpoint, r.Status)

					continue
				}

				// read the post body to a map
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					fmt.Println("could not read body:", err)
					r.Body.Close()
					continue
				}

				var objmap map[string]interface{}
				err = json.Unmarshal(body, &objmap)
				if err != nil {
					fmt.Println("could not map data:", err)
					r.Body.Close()
					continue
				}

				r.Body.Close()

				err = processConf(objmap, t, configPath)
				if err != nil {
					fmt.Println("stopped processing conf:", err)
					continue
				}

				// reload worker is all is ok
				cmd := exec.Command("kill", "-s", "SIGUSR2", "1")
				err = cmd.Run()
				if err != nil {
					fmt.Println("could not reload worker", err)
					continue
				}

				fmt.Println("new config created and worker reloaded")
			}
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
	)

	<-signalChan
	log.Print("os.Interrupt - shutting down...\n")
}

func processConf(objmap map[string]interface{}, t *template.Template, configPath string) error {
	// take checksum of current conf
	oldsum, err := md5sum(configPath)
	if err != nil {
		return err
	}

	// open file and execute template
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}

	// create conf
	err = t.Execute(f, objmap)
	f.Close()
	if err != nil {
		return err
	}

	// take checksum of new conf
	newsum, err := md5sum(configPath)
	if err != nil {
		return err
	}

	// of checksum are the same do nothing
	if oldsum == newsum {
		return errors.New("checksums are identical")
	}

	// validate the file
	validate := exec.Command("haproxy", "-W", "-c", "-f", configPath)
	err = validate.Run()
	if err != nil {
		return err
	}
	exitCode := validate.ProcessState.ExitCode()
	if exitCode != 0 {
		return errors.New("non zero exit code for validation")
	}

	return nil
}

func md5sum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
