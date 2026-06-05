package deploy

import (
	"fmt"
	"net/url"
	"strings"
)

// SiteSlug converts a repo or site name into a DNS-safe hostname label.
func SiteSlug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-':
			b.WriteRune(r)
		case r == '_', r == ' ', r == '.', r == '/':
			b.WriteRune('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "site"
	}
	return slug
}

func deploymentHost(publicDomain, repoName string) string {
	if publicDomain == "" {
		return ""
	}
	return SiteSlug(repoName) + "." + publicDomain
}

// HostFromDeploymentURL extracts the hostname from a stored deployment URL.
func HostFromDeploymentURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		return strings.ToLower(raw)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func deploymentURL(publicDomain string, useHTTPS bool, repoName string, port int) string {
	if publicDomain == "" {
		return fmt.Sprintf("http://localhost:%d", port)
	}
	scheme := "http"
	if useHTTPS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, deploymentHost(publicDomain, repoName))
}
