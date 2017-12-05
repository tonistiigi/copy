package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
)

var appContextCache context.Context
var appContextOnce sync.Once

// appContext returns a static context that reacts to termination signals of the
// running process. Useful in CLI tools.
func appContext() context.Context {
	appContextOnce.Do(func() {
		signals := make(chan os.Signal, 2048)
		signal.Notify(signals, terminationSignals...)

		const exitLimit = 3
		retries := 0

		ctx, cancel := context.WithCancel(context.Background())
		appContextCache = ctx

		go func() {
			for {
				<-signals
				cancel()
				retries++
				if retries >= exitLimit {
					log.Printf("got %d SIGTERM/SIGINTs, forcing shutdown", retries)
					os.Exit(1)
				}
			}
		}()
	})
	return appContextCache
}
