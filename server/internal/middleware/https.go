package middleware

import (
	"net/http"
	"os"
	"strings"
)

func ForceHTTPS(next http.Handler) http.Handler {
	if strings.ToLower(strings.TrimSpace(os.Getenv("QF_FORCE_HTTPS"))) != "true" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
		if proto == "https" || r.TLS != nil {
			next.ServeHTTP(w, r)
			return
		}
		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	})
}
