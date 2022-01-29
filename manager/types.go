package main

import (
	"context"
	"encoding/json"
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

// TODO: handle potential errors
func (c ConfigData) ToJsonBytes() []byte {
	b, _ := json.Marshal(c)
	return b
}

type Reconciliation struct {
	Config ConfigData
	Error  error
}

type Subscription struct {
	CH  chan Reconciliation
	Ctx context.Context
}

type Reconciler struct {
	cli           *client.Client
	tickspeed     time.Duration
	ticker        *time.Ticker
	Subscribers   map[chan Reconciliation]context.Context
	SubscribeChan chan Subscription
}

type ReconciliationBroker interface {
	NextValue(ctx context.Context) (subscription chan Reconciliation)
}
