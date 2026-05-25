package handler

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	authLimitLoginBegin     = "login_begin"
	authLimitLoginFinish    = "login_finish"
	authLimitRegisterBegin  = "register_begin"
	authLimitRegisterFinish = "register_finish"
)

type authLimitPolicy struct {
	Limit  int
	Burst  int
	Window time.Duration
}

type AuthRateLimiter struct {
	mu           sync.Mutex
	trustedProxy bool
	policies     map[string]authLimitPolicy
	buckets      map[string]*authLimitBucket
}

type authLimitBucket struct {
	Count      int
	WindowEnds time.Time
}

func NewAuthRateLimiter(trustedProxy bool) *AuthRateLimiter {
	return &AuthRateLimiter{
		trustedProxy: trustedProxy,
		policies: map[string]authLimitPolicy{
			authLimitLoginBegin:     {Limit: 10, Burst: 20, Window: time.Minute},
			authLimitLoginFinish:    {Limit: 20, Burst: 30, Window: time.Minute},
			authLimitRegisterBegin:  {Limit: 3, Burst: 3, Window: time.Hour},
			authLimitRegisterFinish: {Limit: 10, Burst: 10, Window: time.Hour},
		},
		buckets: make(map[string]*authLimitBucket),
	}
}

func (l *AuthRateLimiter) Middleware(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retryAfter, ok := l.allow(name, clientIP(r, l.trustedProxy), time.Now())
			if !ok {
				w.Header().Set("Retry-After", retryAfter)
				if strings.Contains(r.Header.Get("Accept"), "application/json") || strings.Contains(r.Header.Get("Content-Type"), "application/json") {
					writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too_many_requests"})
					return
				}
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *AuthRateLimiter) allow(name, ip string, now time.Time) (string, bool) {
	policy, ok := l.policies[name]
	if !ok {
		return "", true
	}
	key := name + "\x00" + ip

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if !ok || !now.Before(bucket.WindowEnds) {
		l.buckets[key] = &authLimitBucket{Count: 1, WindowEnds: now.Add(policy.Window)}
		return "", true
	}

	limit := policy.Limit
	if policy.Burst > limit {
		limit = policy.Burst
	}
	if bucket.Count >= limit {
		seconds := int(time.Until(bucket.WindowEnds).Seconds())
		if seconds < 1 {
			seconds = 1
		}
		return strconv.Itoa(seconds), false
	}

	bucket.Count++
	return "", true
}

func clientIP(r *http.Request, trustedProxy bool) string {
	if trustedProxy {
		xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
		if xff != "" {
			if first, _, ok := strings.Cut(xff, ","); ok {
				return strings.TrimSpace(first)
			}
			return xff
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
