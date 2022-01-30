package reconcile

import (
	"context"
	"text/template"
	"time"

	"github.com/bluebrown/moby-ingress/pkg/haproxy"
	"github.com/docker/docker/client"
)

type Reconciliation struct {
	Config haproxy.ConfigData
	Error  error
}

type Subscription struct {
	Ctx  context.Context
	Hash string
	CH   chan *haproxy.HaproxyConfig
}

type Reconciler struct {
	cli             *client.Client
	ticker          *time.Ticker
	Subscribers     map[chan *haproxy.HaproxyConfig]context.Context
	SubscribeChan   chan Subscription
	haproxyConfig   *haproxy.HaproxyConfig
	SetTemplateChan chan *template.Template
}

func NewReconciler(cli *client.Client, tickspeed time.Duration, tpl *template.Template) *Reconciler {
	hp := haproxy.HaproxyConfig{}
	hp.Template = tpl
	err := hp.Set(haproxy.ConfigData{})
	if err != nil {
		panic(err)
	}

	r := Reconciler{
		cli:             cli,
		haproxyConfig:   &hp,
		ticker:          time.NewTicker(tickspeed),
		Subscribers:     make(map[chan *haproxy.HaproxyConfig]context.Context),
		SubscribeChan:   make(chan Subscription, 10),
		SetTemplateChan: make(chan *template.Template),
	}

	return &r

}

type ReconciliationBroker interface {
	NextValue(ctx context.Context, hash string) (subscription chan *haproxy.HaproxyConfig)
	SetTemplate(rawTemplate string) error
}
