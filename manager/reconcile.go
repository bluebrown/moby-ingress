package main

import (
	"context"
	"log"
	"text/template"
	"time"

	"github.com/docker/docker/client"
)

func NewReconciler(cli *client.Client, tickspeed time.Duration, tpl *template.Template) *Reconciler {
	hp := HaproxyConfig{}
	hp.Template = tpl
	err := hp.Set(ConfigData{})
	if err != nil {
		panic(err)
	}

	r := Reconciler{
		cli:           cli,
		haproxyConfig: &hp,
		ticker:        time.NewTicker(tickspeed),
		Subscribers:   make(map[chan *HaproxyConfig]context.Context),
		SubscribeChan: make(chan Subscription, 10),
	}

	return &r

}

// returns a channel that will receive the a reconciliation on the next tick
// and is closed afterwards, so it does not receive more than one message
func (r *Reconciler) NextValue(ctx context.Context, hash string) (subChan chan *HaproxyConfig) {
	subChan = make(chan *HaproxyConfig, 1)
	r.SubscribeChan <- Subscription{ctx, hash, subChan}
	return subChan
}

// get the next config data and publish it to all subscribers
// if the hash hasn't changed, don't publish the data
// as all subscribers have subscribed with the current hash
func (r *Reconciler) publishConf(ctx context.Context) {
	conf, err := CreateConfigData(ctx, r.cli)
	if err != nil {
		log.Printf("[ERROR] %s", err)
		return
	}
	oldHash := r.haproxyConfig.Hash
	r.haproxyConfig.Set(conf)
	if oldHash == r.haproxyConfig.Hash {
		return
	}
	if len(r.Subscribers) == 0 {
		return
	}
	for ch, cx := range r.Subscribers {
		if cx.Err() == nil {
			ch <- r.haproxyConfig
		}
		close(ch)
		delete(r.Subscribers, ch)
	}
}

// starts the reconliiaction loop in a goroutine
// the loop will run until the context is canceled
// each tick will call the publishConf function
// and publish the config data to all subscribers
func (r *Reconciler) Reconcile(ctx context.Context) {
	r.publishConf(ctx)
	go func() {
		for {
			select {
			// if the context is done, stop the loop
			case <-ctx.Done():
				r.ticker.Stop()
				return

			// subscribe a subscriber
			case subscription := <-r.SubscribeChan:
				// if the context is already done
				// close the channel and don't add it to the list
				if subscription.Ctx.Err() != nil {
					close(subscription.CH)
					continue
				}
				// if the hash is not the current hash
				// send the config immediately
				if subscription.Hash != r.haproxyConfig.Hash {
					subscription.CH <- r.haproxyConfig
					continue
				}
				// otherwise, add it to the list
				r.Subscribers[subscription.CH] = subscription.Ctx

			// create the config data on each tick
			// as long as there is at least one subscriber
			// otherwise do nothing and wait for the next tick
			case <-r.ticker.C:
				r.publishConf(ctx)

			}
		}
	}()
}
