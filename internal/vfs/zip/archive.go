package zip

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const dirPrefix = "public/"
const maxSymlinkSize = 256

type zipArchive struct {
	path      string
	once      sync.Once
	done      chan struct{}
	zip       *zip.Reader
	zipCloser io.Closer
	files     map[string]*zip.File
	zipErr    error
}

func (a *zipArchive) openArchive(ctx context.Context) error {
	a.once.Do(func() {
		a.zip, a.zipCloser, a.zipErr = openZIPArchive(a.path)
		if a.zip != nil {
			a.processZip()
		}
		close(a.done)
	})

	// wait for it to close
	// or exit early
	select {
	case <-a.done:
	case <-ctx.Done():
	}
	return a.zipErr
}

func (a *zipArchive) processZip() {
	for _, file := range a.zip.File {
		if !strings.HasPrefix(file.Name, dirPrefix) {
			continue
		}

		a.files[file.Name] = file
	}

	// recycle memory
	a.zip.File = nil
}

func (a *zipArchive) close() {
	if a.zipCloser != nil {
		a.zipCloser.Close()
	}
	a.zipCloser = nil
	a.zip = nil
}

func (a *zipArchive) findFile(name string) *zip.File {
	name = filepath.Clean(name)
	name = strings.TrimPrefix(name, "/")
	return a.files[name]
}

func (a *zipArchive) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	file := a.findFile(name)
	if file == nil {
		return nil, os.ErrNotExist
	}

	return file.FileInfo(), nil
}

func (a *zipArchive) EvalSymlinks(ctx context.Context, name string) (string, error) {
	file := a.findFile(name)
	if file == nil {
		return "", os.ErrNotExist
	}

	if file.FileInfo().Mode()&os.ModeSymlink != os.ModeSymlink {
		return name, nil
	}

	// TODO: to be implemented
	return "", errors.New("to be implemented")
}

func (a *zipArchive) Open(ctx context.Context, name string) (vfs.File, error) {
	file := a.files[name]
	if file == nil {
		return nil, os.ErrNotExist
	}

	dataOffset, err := file.DataOffset()
	if err != nil {
		return nil, err
	}

	// TODO: We can support `io.Seeker` if file would not be compressed

	if !isHTTPArchive(a.path) {
		return file.Open()
	}

	var reader io.ReadCloser
	reader = &httpReader{
		URL: a.path,
		Off: dataOffset,
		N:   int64(file.UncompressedSize64),
	}

	switch file.Method {
	case zip.Deflate:
		reader = newDeflateReader(reader)

	case zip.Store:
		// no-op

	default:
		return nil, fmt.Errorf("unsupported compression: %x", file.Method)
	}

	return reader, nil
}

func newArchive(path string) *zipArchive {
	return &zipArchive{
		path: path,
		done: make(chan struct{}),
	}
}
