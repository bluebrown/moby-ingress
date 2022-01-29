package main

import (
	"context"
	"time"

	"github.com/docker/docker/client"
)

func NewReconciler(cli *client.Client, tickspeed time.Duration) *Reconciler {
	// the true tickspeed will be set after the first tick
	// which is seperately handled by the Reconcile function
	return &Reconciler{
		cli:             cli,
		tickspeed:       tickspeed,
		ticker:          time.NewTicker(time.Hour),
		Subscribers:     make(map[chan Reconciliation]struct{}),
		SubscribeChan:   make(chan chan Reconciliation),
		UnsubscribeChan: make(chan chan Reconciliation),
	}
}

// Subscribe returns a channel that will receive the config data on
// each tick of the reconillation loop
func (r *Reconciler) Subscribe() (subscription chan Reconciliation) {
	subscription = make(chan Reconciliation, 1)
	r.SubscribeChan <- subscription
	return subscription
}

// unsubscribe removes the subscription channel from the list of subscribers
// so that the channel wont receive any more messages
func (r *Reconciler) Unsubscribe(subscription chan Reconciliation) {
	r.UnsubscribeChan <- subscription
}

func (r *Reconciler) publishConf(ctx context.Context) {
	if len(r.Subscribers) == 0 {
		return
	}
	conf, err := CreateConfigData(ctx, r.cli)
	for sub := range r.Subscribers {
		sub <- Reconciliation{conf, err}
	}
}

// starts the reconliiaction loop in a goroutine
// the loop will run until the context is canceled
// each tick will call the publishConf function
// and publish the config data to all subscribers
func (r *Reconciler) Reconcile(ctx context.Context) {
	first := make(chan struct{}, 1)

	go func() {
		time.Sleep(time.Second * 5)
		first <- struct{}{}
	}()

	go func() {
		for {
			select {
			// if the context is done, stop the loop
			case <-ctx.Done():
				r.ticker.Stop()
				return

			// unsubscribe a subscriber
			case subscriber := <-r.UnsubscribeChan:
				close(subscriber)
				delete(r.Subscribers, subscriber)

				// subscribe a subscriber
			case subscriber := <-r.SubscribeChan:
				r.Subscribers[subscriber] = struct{}{}

			// create the config data on each tick
			// as long as there is at least one subscriber
			// otherwise do nothing and wait for the next tick
			case <-r.ticker.C:
				r.publishConf(ctx)

			// wait for the first tick and set the actual tickspeed
			case <-first:
				r.publishConf(ctx)
				r.ticker.Reset(r.tickspeed)
				first = nil
			}
		}
	}()
}
