package main

import (
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
)

// txtRetryDelay is how long watchDiscoveryTXT waits to try again after a
// failed update of the advertisement.
const txtRetryDelay = 10 * time.Second

// txtSetter is the part of discovery.Broadcaster watchDiscoveryTXT needs.
type txtSetter interface {
	SetTXT(txt []string) error
}

// watchDiscoveryTXT re-advertises the TXT record whenever the users change, so
// the setup flag follows the admin state: the device agent's CreateAdmin makes
// the first admin, and an admin can be removed again later. build computes the
// record from the current state; set is a no-op when it has not changed, so
// every user update (not just the ones that flip the admin state) can simply
// trigger a rebuild.
//
// updates must be subscribed before the record the broadcaster started with
// was built, otherwise a change in between is missed.
//
// A failed update is retried after retryDelay, because the record would
// otherwise keep claiming the old admin state until the next user change. The
// returned stop func ends the watcher and closes updates; it blocks until the
// goroutine has exited and may be called more than once.
func watchDiscoveryTXT(updates *queue.Sub[core.UserUpdate], set txtSetter, build func() []string, retryDelay time.Duration) (stop func()) {
	done := make(chan struct{})
	exited := make(chan struct{})

	go func() {
		defer close(exited)
		defer updates.Close()

		var retry <-chan time.Time
		apply := func() {
			if err := set.SetTXT(build()); err != nil {
				log.Warnf("failed to update the mDNS TXT record, will retry: %s", err)
				retry = time.After(retryDelay)
				return
			}
			retry = nil
		}

		for {
			select {
			case <-done:
				return
			case <-updates.Sub():
				apply()
			case <-retry:
				apply()
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() { close(done) })
		<-exited
	}
}
