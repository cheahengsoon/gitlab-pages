package ratelimiter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"gitlab.com/gitlab-org/gitlab-pages/internal/lru"
)

const (
	// DefaultPerDomainFrequency the maximum number of requests per second to be allowed per domain.
	// The default value of 25ms equals 1 request every 25ms -> 40 rps
	DefaultPerDomainFrequency = 25 * time.Millisecond
	// DefaultPerDomainBurstSize is the maximum burst allowed per rate limiter
	// E.g. The first 40 requests within 25ms will succeed, but the 41st will fail until the next
	// refill occurs at DefaultPerDomainFrequency, allowing only 1 request per rate frequency.
	DefaultPerDomainBurstSize = 40

	// avg of ~18,000 unique domains per hour
	// https://log.gprd.gitlab.net/app/lens#/edit/3c45a610-15c9-11ec-a012-eb2e5674cacf?_g=h@e78830b
	defaultDomainsItems              = 20000
	defaultDomainsExpirationInterval = time.Hour
)

type cache interface {
	FindOrFetch(cacheNamespace, key string, fetchFn func() (interface{}, error)) (interface{}, error)
}

// Option function to configure a RateLimiter
type Option func(*RateLimiter)

// RateLimiter holds a map ot domain names with counters that enable rate limiting per domain.
// It uses "golang.org/x/time/rate" as its Token Bucket rate limiter per domain entry.
// See example https://www.fatalerrors.org/a/design-and-implementation-of-time-rate-limiter-for-golang-standard-library.html
// Cleanup runs every cleanupTimer iteration over all domains and removing them if
// the time since counter.lastSeen is greater than the domainMaxTTL.
type RateLimiter struct {
	now                func() time.Time
	perDomainFrequency time.Duration
	perDomainBurstSize int
	domainsCache       cache
	// TODO: add sourceIPCache https://gitlab.com/gitlab-org/gitlab-pages/-/issues/630
}

// New creates a new RateLimiter with default values that can be configured via Option functions
func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		now:                time.Now,
		perDomainFrequency: DefaultPerDomainFrequency,
		perDomainBurstSize: DefaultPerDomainBurstSize,
		domainsCache: lru.New(
			"domains",
			defaultDomainsItems,
			defaultDomainsExpirationInterval,
			// TODO: @jaime to add proper metrics in subsequent MR
			prometheus.NewGaugeVec(prometheus.GaugeOpts{}, []string{"op"}),
			prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"op", "cache"}),
		),
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl
}

// WithNow replaces the RateLimiter now function
func WithNow(now func() time.Time) Option {
	return func(rl *RateLimiter) {
		rl.now = now
	}
}

// WithPerDomainFrequency allows configuring perDomain frequency for the RateLimiter
func WithPerDomainFrequency(d time.Duration) Option {
	return func(rl *RateLimiter) {
		rl.perDomainFrequency = d
	}
}

// WithPerDomainBurstSize configures burst per domain for the RateLimiter
func WithPerDomainBurstSize(burst int) Option {
	return func(rl *RateLimiter) {
		rl.perDomainBurstSize = burst
	}
}

func (rl *RateLimiter) getDomainCounter(domain string) *rate.Limiter {
	limiterI, _ := rl.domainsCache.FindOrFetch(domain, domain, func() (interface{}, error) {
		return rate.NewLimiter(rate.Every(rl.perDomainFrequency), rl.perDomainBurstSize), nil
	})

	return limiterI.(*rate.Limiter)
}

// DomainAllowed checks that the requested domain can be accessed within
// the maxCountPerDomain in the given domainWindow.
func (rl *RateLimiter) DomainAllowed(domain string) (res bool) {
	limiter := rl.getDomainCounter(domain)

	// AllowN allows us to use the rl.now function so we can test this more easily.
	return limiter.AllowN(rl.now(), 1)
}
