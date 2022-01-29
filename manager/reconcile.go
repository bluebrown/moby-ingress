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
		cli:           cli,
		tickspeed:     tickspeed,
		ticker:        time.NewTicker(time.Hour),
		Subscribers:   make(map[chan Reconciliation]context.Context),
		SubscribeChan: make(chan Subscription, 10),
	}
}

// returns a channel that will receive the a reconciliation on the next tick
// and is closed afterwards, so it does not receive more than one message
func (r *Reconciler) NextValue(ctx context.Context) (subscription chan Reconciliation) {
	subscription = make(chan Reconciliation, 1)
	r.SubscribeChan <- Subscription{subscription, ctx}
	return subscription
}

func (r *Reconciler) publishConf(ctx context.Context) {
	if len(r.Subscribers) == 0 {
		return
	}
	conf, err := CreateConfigData(ctx, r.cli)
	for ch, cx := range r.Subscribers {
		// send the value only if the context is not done
		if cx.Err() == nil {
			ch <- Reconciliation{conf, err}
		}
		// close the channel and delete it from the list
		// afterwards as this is a one-shot channel
		close(ch)
		delete(r.Subscribers, ch)
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

				// subscribe a subscriber
			case subscriber := <-r.SubscribeChan:
				// if the context is already done
				// close the channel and don't add it to the list
				if subscriber.Ctx.Err() != nil {
					close(subscriber.CH)
					continue
				}
				// otherwise, add it to the list
				r.Subscribers[subscriber.CH] = subscriber.Ctx

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
