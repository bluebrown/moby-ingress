package reconcile

import (
	"context"
	"log"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/bluebrown/moby-ingress/pkg/decode"
	"github.com/bluebrown/moby-ingress/pkg/haproxy"
)

// returns a channel that will receive the a reconciliation on the next tick
// and is closed afterwards, so it does not receive more than one message
func (r *Reconciler) NextValue(ctx context.Context, hash string) (subChan chan *haproxy.HaproxyConfig) {
	subChan = make(chan *haproxy.HaproxyConfig, 1)
	r.SubscribeChan <- Subscription{ctx, hash, subChan}
	return subChan
}

// get the next config data and publish it to all subscribers
// if the hash hasn't changed, don't publish the data
// as all subscribers have subscribed with the current hash
func (r *Reconciler) publishConf(ctx context.Context) {
	conf, err := decode.DecodeConfigData(ctx, r.cli)
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

			// set the new template concurrency safe
			case tpl := <-r.SetTemplateChan:
				r.haproxyConfig.Template = tpl

			}
		}
	}()
}

// set the new template concurrency safe.
// the new template is used to render the config data
// after it has been set which may happen after the next tick
// but is likely to happen before the next tick
func (r *Reconciler) SetTemplate(rawTemplate string) error {
	tpl, err := template.New("haproxy.cfg.template").Funcs(sprig.TxtFuncMap()).Parse(rawTemplate)
	if err != nil {
		return err
	}
	r.SetTemplateChan <- tpl
	return nil
}
