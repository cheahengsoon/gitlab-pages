package acme

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

// NewMiddleware returns middleware which handle ACME challenges
func NewMiddleware(handler http.Handler, m *Middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := request.GetDomain(r)

		if m.ServeAcmeChallenges(w, r, domain) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}
