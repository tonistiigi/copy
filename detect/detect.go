package detect

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

type ArchiveType int

const (
	Unknown ArchiveType = iota
	Uncompressed
	Gzip
	Bzip2
	Xz
)

func DetectArchiveType(f string) ArchiveType {
	file, err := os.Open(f)
	if err != nil {
		return Unknown
	}
	defer file.Close()
	br := bufio.NewReader(file)
	dt, err := br.Peek(6)
	t := Unknown
	if err == nil {
		t = detectCompression(dt)
	}
	rc, err := uncompressedStream(br, t)
	if err != nil {
		return Unknown
	}
	defer rc.Close()

	r := tar.NewReader(rc)
	_, err = r.Next()
	if err == nil {
		if t == Unknown {
			return Uncompressed
		}
		return t
	}
	return Unknown
}

func detectCompression(source []byte) ArchiveType {
	for compression, m := range map[ArchiveType][]byte{
		Bzip2: {0x42, 0x5A, 0x68},
		Gzip:  {0x1F, 0x8B, 0x08},
		Xz:    {0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
	} {
		if bytes.Equal(m, source[:len(m)]) {
			return compression
		}
	}
	return Unknown
}

func uncompressedStream(r io.Reader, t ArchiveType) (io.ReadCloser, error) {
	switch t {
	case Gzip:
		gzReader, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		return gzReader, nil
	case Bzip2:
		return ioutil.NopCloser(bzip2.NewReader(r)), nil
	case Xz:
		return newXzReader(r)
	default:
		return ioutil.NopCloser(r), nil
	}
}

func newXzReader(r io.Reader) (io.ReadCloser, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "xz", "-d", "-c", "-q")
	cmd.Stdin = r
	outrc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	errCh := make(chan error)
	go func() {
		errCh <- cmd.Run()
	}()
	return &rcWrapper{outrc, func() error {
		cancel()
		err := <-errCh
		outrc.Close()
		return err
	}}, nil
}

type rcWrapper struct {
	io.ReadCloser
	close func() error
}

func (rc *rcWrapper) Close() error {
	return rc.close()
}
