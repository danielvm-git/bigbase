package main

import (
	"context"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/mcp"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/components/sites"
)

// Composition-root adapters keep components free of cross-imports (ECC).

type mcpDeployAdapter struct{ d *deploy.Deploy }

func (a mcpDeployAdapter) Trigger(ctx context.Context, repoID, branch, siteName, siteID string, passthroughPaths []string, appType, manifestPath string) (*mcp.DeploymentResult, error) {
	dep, err := a.d.Trigger(ctx, repoID, branch, siteName, siteID, passthroughPaths, appType, manifestPath)
	if err != nil {
		return nil, err
	}
	return &mcp.DeploymentResult{ID: dep.ID, URL: dep.URL, Status: dep.Status}, nil
}

type mcpSiteListerAdapter struct{ s *sites.Sites }

func (a mcpSiteListerAdapter) ListSites(ctx context.Context) ([]mcp.SiteInfo, error) {
	list, err := a.s.ListSites(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]mcp.SiteInfo, 0, len(list))
	for _, s := range list {
		out = append(out, toMCPSiteInfo(s))
	}
	return out, nil
}

func (a mcpSiteListerAdapter) GetSite(ctx context.Context, siteID string) (*mcp.SiteInfo, error) {
	s, err := a.s.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	info := toMCPSiteInfo(*s)
	return &info, nil
}

func toMCPSiteInfo(s sites.Site) mcp.SiteInfo {
	info := mcp.SiteInfo{
		ID:               s.ID,
		Name:             s.Name,
		GitRepoID:        s.GitRepoID,
		ProductionBranch: s.ProductionBranch,
	}
	if s.LatestDeployment != nil {
		info.LatestDeployment = &mcp.SiteDeployment{
			ID:     s.LatestDeployment.ID,
			URL:    s.LatestDeployment.URL,
			Status: s.LatestDeployment.Status,
		}
	}
	if s.DeployDefaults != nil {
		info.DeployDefaults = &mcp.SiteDeployDefaults{
			AppType:          s.DeployDefaults.AppType,
			BuildCommand:     s.DeployDefaults.BuildCommand,
			StartCommand:     s.DeployDefaults.StartCommand,
			PassthroughPaths: s.DeployDefaults.PassthroughPaths,
			HealthPath:       s.DeployDefaults.HealthPath,
			Env:              s.DeployDefaults.Env,
		}
	}
	return info
}

type mcpSiteKeyAdapter struct{ a *auth.Auth }

func (a mcpSiteKeyAdapter) CreateSiteKey(ctx context.Context, siteID, name string, scopes []string) (string, string, error) {
	return a.a.CreateSiteKey(ctx, siteID, name, scopes)
}

func (a mcpSiteKeyAdapter) ListSiteKeys(ctx context.Context, siteID string) ([]mcp.SiteKeyEntry, error) {
	keys, err := a.a.ListSiteKeys(ctx, siteID)
	if err != nil {
		return nil, err
	}
	out := make([]mcp.SiteKeyEntry, 0, len(keys))
	for _, k := range keys {
		out = append(out, mcp.SiteKeyEntry{
			KeyID:      k.KeyID,
			Name:       k.Name,
			Prefix:     k.Prefix,
			CreatedAt:  k.CreatedAt,
			Revoked:    k.Revoked,
			LastUsedAt: k.LastUsedAt,
		})
	}
	return out, nil
}

func (a mcpSiteKeyAdapter) RevokeSiteKey(ctx context.Context, siteID, keyID string) error {
	return a.a.RevokeSiteKey(ctx, siteID, keyID)
}

func (a mcpSiteKeyAdapter) ResolveSiteKey(rawKey string) (string, error) {
	return a.a.ResolveSiteKey(rawKey)
}

type mcpEnvVarAdapter struct{ s *sites.Sites }

func (a mcpEnvVarAdapter) ListSiteEnvVars(ctx context.Context, siteID string) ([]mcp.SiteEnvVar, error) {
	evs, err := a.s.ListSiteEnvVars(ctx, siteID)
	if err != nil {
		return nil, err
	}
	out := make([]mcp.SiteEnvVar, 0, len(evs))
	for _, ev := range evs {
		out = append(out, mcp.SiteEnvVar{
			ID:           ev.ID,
			SiteID:       ev.SiteID,
			Key:          ev.Key,
			ValuePreview: ev.ValuePreview,
			IsBuildTime:  ev.IsBuildTime,
			IsRuntime:    ev.IsRuntime,
			CreatedAt:    ev.CreatedAt,
			UpdatedAt:    ev.UpdatedAt,
		})
	}
	return out, nil
}

func (a mcpEnvVarAdapter) SetSiteEnvVars(ctx context.Context, siteID string, vars map[string]string) ([]mcp.SiteEnvVar, error) {
	evs, err := a.s.SetSiteEnvVars(ctx, siteID, vars)
	if err != nil {
		return nil, err
	}
	out := make([]mcp.SiteEnvVar, 0, len(evs))
	for _, ev := range evs {
		out = append(out, mcp.SiteEnvVar{
			ID:           ev.ID,
			SiteID:       ev.SiteID,
			Key:          ev.Key,
			ValuePreview: ev.ValuePreview,
			IsBuildTime:  ev.IsBuildTime,
			IsRuntime:    ev.IsRuntime,
			CreatedAt:    ev.CreatedAt,
			UpdatedAt:    ev.UpdatedAt,
		})
	}
	return out, nil
}

func (a mcpEnvVarAdapter) DeleteSiteEnvVar(ctx context.Context, siteID, key string) error {
	return a.s.DeleteSiteEnvVar(ctx, siteID, key)
}

type deployDiagnosisAdapter struct{ m *monitoring.Monitoring }

func (a deployDiagnosisAdapter) GetDiagnosis(ctx context.Context, deployID string) (deploy.Diagnosis, bool, error) {
	d, ok, err := a.m.GetDiagnosis(ctx, deployID)
	if err != nil || !ok {
		return deploy.Diagnosis{}, ok, err
	}
	return deploy.Diagnosis{Diagnosis: d.Diagnosis, Model: d.Model, CreatedAt: d.CreatedAt}, true, nil
}

type deployRelatedEventsAdapter struct{ m *monitoring.Monitoring }

func (a deployRelatedEventsAdapter) GetRelatedEvents(ctx context.Context, deployID string) (deploy.RelatedEvents, bool, error) {
	rel, ok, err := a.m.GetRelatedEvents(ctx, deployID)
	if err != nil || !ok {
		return deploy.RelatedEvents{}, ok, err
	}
	out := deploy.RelatedEvents{
		DeployID:    rel.DeployID,
		WindowStart: rel.WindowStart,
		WindowEnd:   rel.WindowEnd,
		Events:      map[string][]deploy.CorrelatedEvent{},
	}
	for cat, evs := range rel.Events {
		converted := make([]deploy.CorrelatedEvent, 0, len(evs))
		for _, e := range evs {
			converted = append(converted, deploy.CorrelatedEvent{
				Hook: e.Hook, Data: e.Data, Timestamp: e.Timestamp,
			})
		}
		out.Events[cat] = converted
	}
	return out, true, nil
}
