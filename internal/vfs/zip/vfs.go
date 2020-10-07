package zip

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/karlseguin/ccache/v2"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	// TODO: make these configurable https://gitlab.com/gitlab-org/gitlab-pages/-/issues/464
	defaultCacheExpirationInterval = time.Minute
	defaultCacheCleanupInterval    = time.Minute / 2
	defaultCacheRefreshInterval    = time.Minute / 2
)

var (
	errAlreadyCached = errors.New("archive already cached")
)

// zipVFS is a simple cached implementation of the vfs.VFS interface
type zipVFS struct {
	cacheMu         *sync.Mutex
	cache           *ccache.Cache
	dataOffsetCache *ccache.Cache
	readlinkCache   *ccache.Cache

	archiveCount int64
}

// New creates a zipVFS instance that can be used by a serving request
func New() vfs.VFS {
	return &zipVFS{
		cacheMu: &sync.Mutex{},
		// TODO: add cache operation callbacks https://gitlab.com/gitlab-org/gitlab-pages/-/issues/465
		cache: ccache.New(ccache.Configure().MaxSize(1000).
			ItemsToPrune(200).OnDelete(
			func(item *ccache.Item) {
				metrics.ZipCachedArchives.Dec()

				item.Value().(*zipArchive).onEvicted()
			})),
		dataOffsetCache: ccache.New(ccache.Configure().MaxSize(10000).
			ItemsToPrune(2000)),
		readlinkCache: ccache.New(ccache.Configure().MaxSize(1000).
			ItemsToPrune(200)),
	}
}

// Root opens an archive given a URL path and returns an instance of zipArchive
// that implements the vfs.VFS interface.
// To avoid using locks, the findOrOpenArchive function runs inside of a for
// loop until an archive is either found or created and saved.
// If findOrOpenArchive returns errAlreadyCached, the for loop will continue
// to try and find the cached archive or return if there's an error, for example
// if the context is canceled.
func (fs *zipVFS) Root(ctx context.Context, path string) (vfs.Root, error) {
	urlPath, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	return fs.findOrOpenArchive(ctx, urlPath.String())
	// // we do it in loop to not use any additional locks
	// for {
	// 	root, err :=
	// 	if err == errAlreadyCached {
	// 		continue
	// 	}
	//
	// 	return root, err
	// }
}

func (fs *zipVFS) Name() string {
	return "zip"
}

// findOrOpenArchive if found in fs.cache refresh if needed and return it.
// otherwise open the archive and try to save it, if saving fails it's because
// the archive has already been cached (e.g. by another concurrent request)
func (fs *zipVFS) findOrOpenArchive(ctx context.Context, path string) (*zipArchive, error) {
	var archive *zipArchive

	fs.cacheMu.Lock()
	defer fs.cacheMu.Unlock()

	item := fs.cache.Get(path)
	if item == nil || item.Expired() {
		if item != nil {

		}
		archive = newArchive(fs, path, DefaultOpenTimeout)

		fs.cache.Set(path, archive, defaultCacheExpirationInterval)

		metrics.ZipServingArchiveCache.WithLabelValues("miss").Inc()
		metrics.ZipCachedArchives.Inc()
	} else {
		archive = fs.cache.Get(path).Value().(*zipArchive)

		metrics.ZipServingArchiveCache.WithLabelValues("hit").Inc()
	}

	err := archive.openArchive(ctx)
	if err != nil {
		return nil, err
	}

	return archive, nil
}
