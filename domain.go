package main

import (
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type domain struct {
	Group       string
	Project     string
	Config      *domainConfig
	certificate *tls.Certificate
}

func (d *domain) notFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (d *domain) tryFile(w http.ResponseWriter, r *http.Request, projectName, subPath string) bool {
	publicPath := filepath.Join(*pagesRoot, d.Group, projectName, "public")
	fullPath := filepath.Join(publicPath, subPath)
	fullPath = filepath.Clean(fullPath)
	if !strings.HasPrefix(fullPath, publicPath) {
		return false
	}

	fi, err := os.Lstat(fullPath)
	if err != nil {
		return false
	}

	// If this file is directory, open the index.html
	if fi.IsDir() {
		fullPath = filepath.Join(fullPath, "index.html")
		fi, err = os.Lstat(fullPath)
		if err != nil {
			return false
		}
	}

	// We don't allow to open non-regular files
	if !fi.Mode().IsRegular() {
		return false
	}

	// Open and serve content of file
	file, err := os.Open(fullPath)
	if err != nil {
		return false
	}
	defer file.Close()

	fi, err = file.Stat()
	if err != nil {
		return false
	}

	http.ServeContent(w, r, filepath.Base(file.Name()), fi.ModTime(), file)
	return true
}

func (d *domain) serveFromGroup(w http.ResponseWriter, r *http.Request) {
	// The Path always contains "/" at the beggining
	split := strings.SplitN(r.URL.Path, "/", 3)

	if len(split) >= 2 {
		subPath := ""
		if len(split) >= 3 {
			subPath = split[2]
		}
		if d.tryFile(w, r, split[1], subPath) {
			return
		}
	}

	if d.tryFile(w, r, strings.ToLower(r.Host), r.URL.Path) {
		return
	}

	d.notFound(w, r)
}

func (d *domain) serveFromConfig(w http.ResponseWriter, r *http.Request) {
	if d.tryFile(w, r, d.Project, r.URL.Path) {
		return
	}

	d.notFound(w, r)
}

func (d *domain) ensureCertificate() (*tls.Certificate, error) {
	if d.Config == nil {
		return nil, errors.New("tls certificates can be loaded only for pages with configuration")
	}

	if d.certificate != nil {
		return d.certificate, nil
	}

	tls, err := tls.X509KeyPair([]byte(d.Config.Certificate), []byte(d.Config.Key))
	if err != nil {
		return nil, err
	}

	d.certificate = &tls
	return d.certificate, nil
}

func (d *domain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d.Config != nil {
		d.serveFromConfig(w, r)
	} else {
		d.serveFromGroup(w, r)
	}
}
