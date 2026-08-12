package main

import (
	"context"
	"errors"
	"strings"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/mcp"
	"github.com/danielvm/bigbase/components/messaging"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
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

// ResolveSiteKeyScopes implements mcp.SiteKeyAuthenticator (e89s07): bb_dep_
// keys resolve to their bound Site plus the key's own scopes.
func (a mcpSiteKeyAdapter) ResolveSiteKeyScopes(rawKey string) (string, []string, error) {
	return a.a.ResolveSiteKeyScopes(rawKey)
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

func (a mcpEnvVarAdapter) AuthorizeSiteTarget(ctx context.Context, siteID string, orgID int64) error {
	return a.s.AuthorizeSiteTarget(ctx, siteID, orgID)
}

// errProjectDenied is the non-disclosing denial shared by the project target
// authorizer for cross-organization targets, missing Projects, and Site-key
// binding mismatches. Existence is never disclosed.
var errProjectDenied = errors.New("project authorization denied")

// mcpProjectTargetAuthorizerAdapter implements mcp.ProjectTargetAuthorizer at
// the composition root. Organization principals must own the target Project
// (mirroring the SecretManager's requireProject ownership check used by the
// REST adapter); Site deploy keys are restricted to the Project of their bound
// Site. All ownership is derived from the authenticated principal, never from
// caller arguments (e89s07 P0-01/P0-03).
type mcpProjectTargetAuthorizerAdapter struct {
	db kernel.DBer
}

func (a mcpProjectTargetAuthorizerAdapter) AuthorizeProjectTarget(ctx context.Context, projectID string, p mcp.Principal) error {
	if projectID == "" {
		return errProjectDenied
	}
	switch p.Kind {
	case mcp.PrincipalOrg:
		if p.OrgID <= 0 {
			return errProjectDenied
		}
		var projectOrg int64
		err := a.db.QueryRowContext(ctx,
			`SELECT org_id FROM projects WHERE id = ?`, projectID).Scan(&projectOrg)
		if err != nil || projectOrg != p.OrgID {
			return errProjectDenied
		}
		return nil
	case mcp.PrincipalSite:
		if p.SiteID == "" {
			return errProjectDenied
		}
		var siteProjectID string
		err := a.db.QueryRowContext(ctx,
			`SELECT project_id FROM sites WHERE id = ?`, p.SiteID).Scan(&siteProjectID)
		if err != nil || siteProjectID != projectID {
			return errProjectDenied
		}
		return nil
	default:
		return errProjectDenied
	}
}

// mcpProjectSecretManagerAdapter implements mcp.ProjectSecretManager over the
// frozen secrets.SecretManager seam. It bridges the authenticated MCP
// principal into the SecretManager's auth context (org identity is injected
// from the credential — for Site principals from the bound Site's owning
// organization, server-derived) and translates storage failures into the MCP
// sentinel errors so tool responses stay generic and value-free.
type mcpProjectSecretManagerAdapter struct {
	m  *secrets.Secrets
	db kernel.DBer
}

// managerContext derives the SecretManager auth context from the principal.
// Organization principals inject their own org; Site principals inject the org
// that owns their bound Site (resolved server-side). A zero principal fails
// closed with ErrSecretOrganizationRequired semantics.
func (a mcpProjectSecretManagerAdapter) managerContext(ctx context.Context, p mcp.Principal) (context.Context, error) {
	switch p.Kind {
	case mcp.PrincipalOrg:
		if p.OrgID <= 0 {
			return ctx, secrets.ErrOrganizationRequired
		}
		return auth.WithOrgID(ctx, p.OrgID), nil
	case mcp.PrincipalSite:
		if p.SiteID == "" {
			return ctx, secrets.ErrOrganizationRequired
		}
		var orgID int64
		err := a.db.QueryRowContext(ctx,
			`SELECT org_id FROM sites WHERE id = ?`, p.SiteID).Scan(&orgID)
		if err != nil || orgID <= 0 {
			return ctx, secrets.ErrOrganizationRequired
		}
		return auth.WithOrgID(ctx, orgID), nil
	default:
		return ctx, secrets.ErrOrganizationRequired
	}
}

// mapSecretError translates frozen SecretManager failures into the MCP
// sentinels. Unknown and crypto errors collapse to ErrSecretInternal so the
// client only ever sees a generic message (SC-e89s07-P0-05).
func mapSecretError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, secrets.ErrSecretNotFound), errors.Is(err, secrets.ErrVersionNotFound):
		return mcp.ErrSecretNotFound
	case errors.Is(err, secrets.ErrOrganizationRequired),
		errors.Is(err, secrets.ErrProjectNotFound),
		errors.Is(err, secrets.ErrEnvironmentNotFound),
		errors.Is(err, secrets.ErrFolderNotFound):
		return mcp.ErrSecretTarget
	case errors.Is(err, secrets.ErrSecretAlreadyExists):
		return mcp.ErrSecretConflict
	case strings.Contains(err.Error(), "invalid secret key"):
		return mcp.ErrSecretInvalidKey
	case strings.Contains(err.Error(), "exceeds"):
		return mcp.ErrSecretTooLarge
	case errors.Is(err, secrets.ErrInvalidKeyMaterial),
		errors.Is(err, secrets.ErrDecryptionFailed),
		errors.Is(err, secrets.ErrActiveKeyNotFound):
		return mcp.ErrSecretInternal
	default:
		return mcp.ErrSecretInternal
	}
}

func (a mcpProjectSecretManagerAdapter) EnsureFolder(ctx context.Context, p mcp.Principal, projectID, environmentID, name string) (string, error) {
	mctx, err := a.managerContext(ctx, p)
	if err != nil {
		return "", mapSecretError(err)
	}
	folder, err := a.m.EnsureFolder(mctx, projectID, environmentID, name)
	if err != nil {
		return "", mapSecretError(err)
	}
	return folder.ID, nil
}

func (a mcpProjectSecretManagerAdapter) ListSecrets(ctx context.Context, p mcp.Principal, projectID, environmentID, folderID string) ([]mcp.SecretMetadata, error) {
	mctx, err := a.managerContext(ctx, p)
	if err != nil {
		return nil, mapSecretError(err)
	}
	items, err := a.m.ListSecrets(mctx, projectID, environmentID, folderID)
	if err != nil {
		return nil, mapSecretError(err)
	}
	out := make([]mcp.SecretMetadata, 0, len(items))
	for _, s := range items {
		out = append(out, toMCPSecretMetadata(s))
	}
	return out, nil
}

func (a mcpProjectSecretManagerAdapter) GetSecretMetadata(ctx context.Context, p mcp.Principal, projectID, environmentID, folderID, key string) (mcp.SecretMetadata, error) {
	mctx, err := a.managerContext(ctx, p)
	if err != nil {
		return mcp.SecretMetadata{}, mapSecretError(err)
	}
	meta, err := a.m.GetSecretMetadata(mctx, projectID, environmentID, folderID, key)
	if err != nil {
		return mcp.SecretMetadata{}, mapSecretError(err)
	}
	return toMCPSecretMetadata(meta), nil
}

func (a mcpProjectSecretManagerAdapter) ReadSecretValue(ctx context.Context, p mcp.Principal, projectID, environmentID, folderID, key string) (mcp.SecretValue, error) {
	mctx, err := a.managerContext(ctx, p)
	if err != nil {
		return mcp.SecretValue{}, mapSecretError(err)
	}
	value, err := a.m.ReadSecretValue(mctx, projectID, environmentID, folderID, key)
	if err != nil {
		return mcp.SecretValue{}, mapSecretError(err)
	}
	return mcp.SecretValue{
		SecretID:  value.SecretID,
		Key:       value.Key,
		Version:   value.Version,
		Value:     value.Value,
		KeyID:     value.KeyID,
		Algorithm: value.Algorithm,
	}, nil
}

func (a mcpProjectSecretManagerAdapter) CreateSecret(ctx context.Context, p mcp.Principal, projectID, environmentID, folderID, key, value string) (mcp.SecretMetadata, error) {
	mctx, err := a.managerContext(ctx, p)
	if err != nil {
		return mcp.SecretMetadata{}, mapSecretError(err)
	}
	meta, err := a.m.CreateSecret(mctx, projectID, environmentID, folderID, key, value)
	if err != nil {
		return mcp.SecretMetadata{}, mapSecretError(err)
	}
	return toMCPSecretMetadata(meta), nil
}

func (a mcpProjectSecretManagerAdapter) UpdateSecret(ctx context.Context, p mcp.Principal, projectID, environmentID, folderID, key, value string) (mcp.SecretMetadata, error) {
	mctx, err := a.managerContext(ctx, p)
	if err != nil {
		return mcp.SecretMetadata{}, mapSecretError(err)
	}
	meta, err := a.m.UpdateSecret(mctx, projectID, environmentID, folderID, key, value)
	if err != nil {
		return mcp.SecretMetadata{}, mapSecretError(err)
	}
	return toMCPSecretMetadata(meta), nil
}

func (a mcpProjectSecretManagerAdapter) DeleteSecret(ctx context.Context, p mcp.Principal, projectID, environmentID, folderID, key string) error {
	mctx, err := a.managerContext(ctx, p)
	if err != nil {
		return mapSecretError(err)
	}
	if err := a.m.DeleteSecret(mctx, projectID, environmentID, folderID, key); err != nil {
		return mapSecretError(err)
	}
	return nil
}

func toMCPSecretMetadata(s secrets.SecretMetadata) mcp.SecretMetadata {
	return mcp.SecretMetadata{
		ID:             s.ID,
		ProjectID:      s.ProjectID,
		EnvironmentID:  s.EnvironmentID,
		FolderID:       s.FolderID,
		Key:            s.Key,
		CurrentVersion: s.CurrentVersion,
		ValuePreview:   s.ValuePreview,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
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

// alertNotifierAdapter bridges monitoring.AlertNotifier (which uses the
// monitoring.AlertEvent type) to messaging.SMTPAlertNotifier (which uses its
// own messaging.AlertEvent to avoid an import cycle). (Issue #178.)
type alertNotifierAdapter struct{ m *messaging.SMTPAlertNotifier }

func (a alertNotifierAdapter) NotifyAlert(ctx context.Context, ev monitoring.AlertEvent) error {
	return a.m.NotifyAlert(ctx, messaging.AlertEvent{
		AlertID:    ev.AlertID,
		IncidentID: ev.IncidentID,
		Name:       ev.Name,
		Metric:     ev.Metric,
		Value:      ev.Value,
		Threshold:  ev.Threshold,
		Operator:   ev.Operator,
	})
}
