package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/tonistiigi/copy/detect"
)

// cp with Dockerfile ADD/COPY semantics

func main() {
	var upackF bool
	flag.BoolVar(&upackF, "unpack", false, "")
	flag.Parse()
	args := flag.Args()
	if err := copy(args, upackF); err != nil {
		panic(err)
	}
}

func copy(args []string, unpack bool) error {
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
	origDest := dest

	if !strings.HasSuffix(dest, "/") && !isdir {
		dest = path.Dir(dest)
	}

	if err := os.MkdirAll(dest, 0700); err != nil {
		return err
	}

	// if target is dir extract or copy all
	fi, err := os.Stat(origDest)
	if err == nil {
		if fi.IsDir() && unpack {
			for _, src := range srcs {
				if err := runUnpack(src, origDest, detect.DetectArchiveType(src)); err != nil {
					return err
				}

			}
			return nil
		}
	}

	// create destination directory for single archive source
	if unpack && len(srcs) == 1 {
		typ := detect.DetectArchiveType(srcs[0])
		if typ != detect.Unknown {
			if err := os.MkdirAll(origDest, 0700); err != nil {
				return err
			}
			if err := runUnpack(srcs[0], origDest, typ); err != nil {
				return err
			}
			return nil
		}
	}

	return runCp(srcs, origDest)
}

func runCp(srcs []string, dest string) error {
	cmd := exec.CommandContext(appContext(), "cp", append(append([]string{"-a", "--reflink=auto"}, srcs...), dest)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runUnpack(src, dest string, t detect.ArchiveType) error {
	if t == detect.Unknown {
		return runCp([]string{src}, dest)
	}
	flags := "-xv"
	switch t {
	case detect.Gzip:
		flags += "z"
	case detect.Bzip2:
		flags += "j"
	case detect.Xz:
		flags += "J"
	}
	cmd := exec.CommandContext(appContext(), "tar", flags+"f", src, "-C", dest)
	log.Println("exec", cmd.Path, cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
