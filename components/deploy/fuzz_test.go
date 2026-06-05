package deploy

import "testing"

// FuzzSiteSlug fuzzes the SiteSlug function with arbitrary strings.
func FuzzSiteSlug(f *testing.F) {
	seeds := []string{
		"my-app",
		"My App",
		"hello_world",
		"test.site/app",
		"UPPERCASE",
		"---leading---trailing---",
		"special!@#$chars",
		"a",
		"",
		"   spaces   ",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		slug := SiteSlug(name)
		// Slug must never be empty
		if slug == "" {
			t.Errorf("SiteSlug(%q) returned empty slug", name)
		}
		// Slug must be DNS-safe: only lowercase alphanumeric and hyphens
		for _, r := range slug {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				t.Errorf("SiteSlug(%q) = %q contains invalid char %c", name, slug, r)
			}
		}
		// Slug must not start or end with hyphen
		if slug[0] == '-' || slug[len(slug)-1] == '-' {
			t.Errorf("SiteSlug(%q) = %q starts or ends with hyphen", name, slug)
		}
	})
}

// FuzzHostFromDeploymentURL fuzzes HostFromDeploymentURL with arbitrary strings.
func FuzzHostFromDeploymentURL(f *testing.F) {
	seeds := []string{
		"https://my-app.example.com",
		"http://localhost:8080",
		"my-app.example.com",
		"",
		"invalid-url:::",
		"https://192.168.1.1:3000",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		host := HostFromDeploymentURL(raw)
		// Must never panic
		_ = host
	})
}
