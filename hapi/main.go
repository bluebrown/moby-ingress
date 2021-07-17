package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
)

var (
	myClient     = &http.Client{Timeout: 10 * time.Second}
	configPath   *string
	templatePath *string
	t            *template.Template
)

func main() {
	configPath = flag.String("config", "/usr/local/etc/haproxy/haproxy.cfg", "path to config inside the container")
	configPath = flag.String("c", "/usr/local/etc/haproxy/haproxy.cfg", "path to config inside the container")
	templatePath = flag.String("template", "/usr/local/etc/haproxy/haproxy.templ.cfg", "path to template inside the container")
	flag.Parse()

	parts := strings.Split(*templatePath, "/")
	name := parts[len(parts)-1]
	t = template.Must(template.New(name).Funcs(sprig.TxtFuncMap()).ParseFiles(*templatePath))

	for {
		time.Sleep(time.Second * 30)
		r, err := myClient.Get("http://manager:8080/json")
		if err != nil {
			fmt.Println(err)
			continue
		}

		// read the post body to a map
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			r.Body.Close()
			continue
		}

		var objmap map[string]interface{}
		err = json.Unmarshal(body, &objmap)
		if err != nil {
			fmt.Println(err)
			r.Body.Close()
			continue
		}

		r.Body.Close()

		err = processConf(objmap, t, *configPath)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// reload worker is all is ok
		cmd := exec.Command("kill", "-s", "SIGUSR2", "1")
		err = cmd.Run()
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("new config created and worker reloaded")
	}

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

func PosrHandler(w http.ResponseWriter, r *http.Request) {

	// only accept post method
	if r.Method != "POST" {
		fmt.Fprintln(w, "only post method is allowed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// read the post body to a map
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintln(w, "could not read post body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var objmap map[string]interface{}
	err = json.Unmarshal(body, &objmap)
	if err != nil {
		fmt.Fprintln(w, "could not parse post body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = processConf(objmap, t, *configPath)
	if err != nil {
		fmt.Fprintf(w, "not reloading worker due to %v\n", err)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// reload worker is all is ok
	cmd := exec.Command("kill", "-s", "SIGUSR2", "1")
	err = cmd.Run()
	if err != nil {
		fmt.Fprintln(w, "could not reload worker")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "new config created and worker reloaded")
	w.WriteHeader(http.StatusCreated)

}
