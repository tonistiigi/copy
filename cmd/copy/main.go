package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/containerd/continuity/fs"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/pkg/errors"
	"github.com/tonistiigi/copy/detect"
	"github.com/tonistiigi/copy/user"
	copy "github.com/tonistiigi/fsutil/copy"
)

// cp with Dockerfile ADD/COPY semantics

type opts struct {
	unpack bool
	chown  *copy.ChownOpt
	root   string
}

type chown struct {
	uid, gid int
}

func main() {
	var opt opts
	var username string
	flag.BoolVar(&opt.unpack, "unpack", false, "")
	flag.StringVar(&username, "chown", "", "")

	flag.Parse()
	args := flag.Args()

	if username != "" {
		uid, gid, err := user.GetUser(appcontext.Context(), "/", username)
		if err != nil {
			panic(err)
		}
		opt.chown = &copy.ChownOpt{Uid: int(uid), Gid: int(gid)}
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	opt.root = wd

	if err := runCopy(appcontext.Context(), args, opt); err != nil {
		panic(err)
	}
}

func runCopy(ctx context.Context, args []string, opt opts) error {
	if len(args) < 2 {
		return fmt.Errorf("invalid args %v", args)
	}

	srcs := args[:len(args)-1]
	isdir := false

	for i, src := range srcs {
		fi, err := os.Lstat(src)
		if err == nil && fi.IsDir() {
			isdir = true
			srcs[i] = path.Clean(src) + "/."
		}
	}

	if len(srcs) > 1 {
		isdir = true
	}

	dest := args[len(args)-1]

	// This handles the case where destination is a symlink.
	if fi, err := os.Lstat(path.Clean(dest)); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			safeDest, err := fs.RootPath(opt.root, dest)
			if err != nil {
				return err
			}
			if strings.HasSuffix(dest, "/") {
				dest = safeDest + "/"
			} else {
				dest = safeDest
			}
		}
	}

	origDest := dest
	// If destination is a file, then ensure dest is always a directory by using the parent
	if !strings.HasSuffix(dest, "/") && !isdir {
		dest = path.Dir(dest)
	}

	if err := mkdirp(dest, opt); err != nil {
		return err
	}

	// if target is dir extract or copy all
	fi, err := os.Stat(dest)
	if err == nil {
		if fi.IsDir() && opt.unpack {
			for _, src := range srcs {
				if err := runUnpack(ctx, src, origDest, detect.DetectArchiveType(src), opt); err != nil {
					return err
				}

			}
			return nil
		}
	}

	// create destination directory for single archive source
	if opt.unpack && len(srcs) == 1 {
		typ := detect.DetectArchiveType(srcs[0])
		if typ != detect.Unknown {
			if err := runUnpack(ctx, srcs[0], origDest, typ, opt); err != nil {
				return err
			}
			return nil
		}
	}

	return runCp(ctx, srcs, origDest, opt)
}

func runCp(ctx context.Context, srcs []string, dest string, opt opts) error {
	xattrErrorHandler := func(dst, src, key string, err error) error {
		log.Println(err)
		return nil
	}
	for _, src := range srcs {
		if err := copy.Copy(ctx, src, dest, copy.AllowWildcards, copy.WithXAttrErrorHandler(xattrErrorHandler),
			func(ci *copy.CopyInfo) {
				ci.Chown = opt.chown
			}); err != nil {
			return errors.Wrapf(err, "failed to copy %s to %s", src, dest)
		}
	}
	return nil
}

func runUnpack(ctx context.Context, src, dest string, t detect.ArchiveType, opt opts) error {
	if t == detect.Unknown {
		return runCp(ctx, []string{src}, dest, opt)
	}

	// f, err := os.Open(src)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to open %s", src)
	// }
	//
	// if _, err := archive.Extract(ctx, dest, f); err != nil {
	// 	f.Close()
	// 	return err
	// }
	// return nil
	flags := "-xv"
	switch t {
	case detect.Gzip:
		flags += "z"
	case detect.Bzip2:
		flags += "j"
	case detect.Xz:
		flags += "J"
	}

	if err := mkdirp(dest, opt); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "tar", flags+"f", src, "-C", dest)
	log.Println("exec", cmd.Path, cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func mkdirp(p string, opt opts) error {
	if err := os.MkdirAll(p, 0755); err != nil {
		return err
	}
	if chown := opt.chown; chown != nil {
		if err := os.Lchown(p, chown.Uid, chown.Gid); err != nil {
			return errors.Wrapf(err, "failed to chown %s", p)
		}
	}
	return nil
}
