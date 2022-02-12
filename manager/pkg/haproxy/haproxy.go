package haproxy

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"text/template"

	"github.com/docker/docker/api/types/swarm"
)

type ConfigData struct {
	IngressClass string             `json:"-" mapstructure:"class"`
	Global       string             `json:"global,omitempty"`
	Defaults     string             `json:"defaults,omitempty"`
	Frontend     map[string]string  `json:"frontend,omitempty"`
	Backend      map[string]Backend `json:"backend,omitempty"`
}

type Backend struct {
	EndpointMode swarm.ResolutionMode `json:"endpoint_mode,omitempty"`
	Port         string               `json:"port,omitempty"`
	Replicas     uint64               `json:"replicas,omitempty"`
	Frontend     map[string]string    `json:"-"`
	Backend      string               `json:"backend,omitempty"`
}

type HaproxyConfig struct {
	Template *template.Template
	File     []byte
	Hash     string
	JSON     []byte
}

func (hp *HaproxyConfig) Set(conf ConfigData) error {
	b := new(bytes.Buffer)
	err := hp.Template.Execute(b, conf)
	if err != nil {
		return err
	}
	hp.File = b.Bytes()
	hashBytes := md5.Sum(hp.File)
	hp.Hash = hex.EncodeToString(hashBytes[:])
	jsonBytes, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	hp.JSON = jsonBytes
	return nil
}
