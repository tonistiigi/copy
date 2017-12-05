// +build !windows

package main

import (
	"os"
	"syscall"
)

var terminationSignals = []os.Signal{syscall.SIGTERM, syscall.SIGINT}
