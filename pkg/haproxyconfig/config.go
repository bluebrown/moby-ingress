package haproxyconfig

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"text/template"
)

type ConfigData struct {
	IngressClass string              `json:"-" mapstructure:"class"`
	Global       string              `json:"global,omitempty"`
	Defaults     string              `json:"defaults,omitempty"`
	Frontend     map[string]string   `json:"frontend,omitempty"`
	Backend      map[string]*Backend `json:"backend,omitempty"`
}

type Backend struct {
	EndpointMode string            `json:"endpoint_mode,omitempty"`
	Port         string            `json:"port,omitempty"`
	Replicas     uint64            `json:"replicas,omitempty"`
	Frontend     map[string]string `json:"-"`
	Backend      string            `json:"backend,omitempty"`
}

type HaproxyConfig struct {
	Template *template.Template
	File     []byte
	Hash     string
}

func (hc *HaproxyConfig) Set(data *ConfigData) error {
	b := new(bytes.Buffer)
	err := hc.Template.Execute(b, data)
	if err != nil {
		return err
	}
	hc.File = b.Bytes()
	hashBytes := md5.Sum(hc.File)
	hc.Hash = hex.EncodeToString(hashBytes[:])
	return nil
}
