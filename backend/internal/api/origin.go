package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func allowOrigin(r *http.Request, whitelist []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}
	if host == "127.0.0.1" || host == "localhost" || host == "::1" {
		return true
	}
	for _, w := range whitelist {
		if strings.EqualFold(strings.TrimSpace(w), origin) {
			return true
		}
	}
	return false
}
