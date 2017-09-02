package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// cp with Dockerfile ADD/COPY semantics

func main() {
	flag.Parse()
	args := flag.Args()
	if err := copy(args); err != nil {
		panic(err)
	}
}

func copy(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("invalid args %v", args)
	}

	srcs := args[:len(args)-1]
	isdir := false

	for i, src := range srcs {
		fi, err := os.Lstat(src)
		if err != nil {
			return fmt.Errorf("lstat %s: %v", src, err)
		}
		if fi.IsDir() {
			isdir = true
			srcs[i] = path.Clean(src) + "/."
		}
	}

	if len(srcs) > 1 {
		isdir = true
	}

	dest := args[len(args)-1]

	if !strings.HasSuffix(dest, "/") && !isdir {
		dest = path.Dir(dest)
	}

	if err := os.MkdirAll(dest, 0700); err != nil {
		return err
	}

	cmd := exec.Command("cp", append(append([]string{"-a", "--reflink=auto"}, srcs...), args[len(args)-1])...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
