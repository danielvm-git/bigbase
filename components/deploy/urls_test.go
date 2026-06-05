package deploy

import "testing"

func TestSiteSlug(t *testing.T) {
	if got := SiteSlug("My App"); got != "my-app" {
		t.Fatalf("SiteSlug(My App) = %q, want my-app", got)
	}
	if got := SiteSlug("add-tutorial-requests-site"); got != "add-tutorial-requests-site" {
		t.Fatalf("unexpected slug: %q", got)
	}
}

func TestDeploymentURL(t *testing.T) {
	if got := deploymentURL("", true, "repo", 10958); got != "http://localhost:10958" {
		t.Fatalf("dev URL = %q", got)
	}
	want := "https://test-repo.bigbase.click"
	if got := deploymentURL("bigbase.click", true, "test-repo", 10958); got != want {
		t.Fatalf("prod URL = %q, want %q", got, want)
	}
}

func TestDeploymentHost(t *testing.T) {
	if got := deploymentHost("bigbase.click", "My Site"); got != "my-site.bigbase.click" {
		t.Fatalf("host = %q", got)
	}
}

func TestHostFromDeploymentURL(t *testing.T) {
	if got := HostFromDeploymentURL("https://my-app.bigbase.click"); got != "my-app.bigbase.click" {
		t.Fatalf("host = %q", got)
	}
	if got := HostFromDeploymentURL(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
}
