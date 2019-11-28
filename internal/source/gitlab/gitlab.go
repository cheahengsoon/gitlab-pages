package gitlab

import (
	"errors"
	"net/http"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

// Gitlab source represent a new domains configuration source. We fetch all the
// information about domains from GitLab instance.
type Gitlab struct {
	client Client
	cache  *cache.Cache // WIP
}

// New returns a new instance of gitlab domain source.
func New(config client.Config) *Gitlab {
	return &Gitlab{client: client.NewFromConfig(config), cache: cache.New()}
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
func (g *Gitlab) GetDomain(name string) (*domain.Domain, error) {
	response, err := g.client.GetVirtualDomain(name)
	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, errors.New("could not fetch a domain information")
	}

	domain := domain.Domain{
		Name:            name,
		CertificateCert: response.Certificate,
		CertificateKey:  response.Key,
		Resolver:        g,
	}

	return &domain, nil
}

// Resolve is supposed to get the serving lookup path based on the request from
// the GitLab source
func (g *Gitlab) Resolve(r *http.Request) (*serving.LookupPath, string, error) {
	response, err := g.client.GetVirtualDomain(r.Host)
	if err != nil {
		return nil, "", err
	}

	for _, lookup := range response.LookupPaths {
		if strings.Contains(r.URL.Path, lookup.Prefix) {
			lookupPath := &serving.LookupPath{
				Prefix:             lookup.Prefix,
				Path:               strings.TrimPrefix(lookup.Source.Path, "/"),
				IsNamespaceProject: (lookup.Prefix == "/" && len(response.LookupPaths) > 1),
				IsHTTPSOnly:        lookup.HTTPSOnly,
				HasAccessControl:   lookup.AccessControl,
				ProjectID:          uint64(lookup.ProjectID),
			}

			requestPath := strings.TrimPrefix(r.URL.Path, lookup.Prefix)

			return lookupPath, requestPath, nil
		}
	}

	return nil, "", errors.New("could not match lookup path")
}
