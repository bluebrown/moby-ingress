package main

import (
	"context"
	"text/template"
	"time"

	"github.com/docker/docker/client"
)

type ConfigData struct {
	IngressClass string             `json:"-" mapstructure:"class"`
	Global       string             `json:"global,omitempty"`
	Defaults     string             `json:"defaults,omitempty"`
	Frontend     map[string]string  `json:"frontend,omitempty"`
	Backend      map[string]Backend `json:"backend,omitempty"`
}

type Backend struct {
	Port     string            `json:"port,omitempty"`
	Replicas uint64            `json:"replicas,omitempty"`
	Frontend map[string]string `json:"-"`
	Backend  string            `json:"backend,omitempty"`
}

type Reconciliation struct {
	Config ConfigData
	Error  error
}

type Subscription struct {
	Ctx  context.Context
	Hash string
	CH   chan *HaproxyConfig
}

type HaproxyConfig struct {
	Template *template.Template
	File     []byte
	Hash     string
	JSON     []byte
}

type Reconciler struct {
	cli           *client.Client
	ticker        *time.Ticker
	Subscribers   map[chan *HaproxyConfig]context.Context
	SubscribeChan chan Subscription
	haproxyConfig *HaproxyConfig
}

type ReconciliationBroker interface {
	NextValue(ctx context.Context, hash string) (subscription chan *HaproxyConfig)
}
