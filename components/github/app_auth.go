package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func (g *GitHub) appJWT() (string, error) {
	keyPEM, err := os.ReadFile(g.privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("read private key: %w", err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyPEM)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    g.appID,
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(key)
}

func (g *GitHub) installationToken(ctx context.Context, installationID int64) (string, error) {
	jwtStr, err := g.appJWT()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwtStr)
	req.Header.Set("Accept", "application/vnd.github+json")

	res, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return "", fmt.Errorf("github token: %s: %s", res.Status, string(body))
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Token, nil
}

func (g *GitHub) fetchInstallation(ctx context.Context, installationID int64) (*installationDetails, error) {
	jwtStr, err := g.appJWT()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwtStr)
	req.Header.Set("Accept", "application/vnd.github+json")

	res, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github installation: %s", res.Status)
	}

	var out struct {
		Account struct {
			Login string `json:"login"`
		} `json:"account"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &installationDetails{AccountLogin: out.Account.Login}, nil
}

func (g *GitHub) listInstallationRepos(ctx context.Context, installationID int64) ([]Repo, error) {
	token, err := g.installationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.github.com/installation/repositories?per_page=100")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	res, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("github repos: %s: %s", res.Status, string(body))
	}

	var out struct {
		Repositories []struct {
			ID          int64  `json:"id"`
			FullName    string `json:"full_name"`
			DefaultBranch string `json:"default_branch"`
			Private     bool   `json:"private"`
			Description string `json:"description"`
		} `json:"repositories"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	repos := make([]Repo, 0, len(out.Repositories))
	for _, r := range out.Repositories {
		branch := r.DefaultBranch
		if branch == "" {
			branch = "main"
		}
		repos = append(repos, Repo{
			ID:            r.ID,
			FullName:      r.FullName,
			DefaultBranch: branch,
			Private:       r.Private,
			Description:   r.Description,
		})
	}
	return repos, nil
}
