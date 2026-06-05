import React, { useState } from 'react';
import { Button } from './Button';
import { Badge, StatusBadge } from './Badge';
import { Input } from './Input';
import { Card, EmptyState } from './Card';

/**
 * Component stories / examples — one block per component covering every
 * documented state (default · hover · focus · active · disabled · loading ·
 * error · empty). Framework-agnostic: works as plain render functions, or
 * drop the named exports into Storybook (CSF). Hover/focus/active are driven
 * by CSS, so they're demonstrated live rather than as separate snapshots.
 */

/* ── Button ─────────────────────────────────────────────── */
export const ButtonStates = () => (
  <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', alignItems: 'center' }}>
    <Button variant="primary">Create site</Button>
    <Button variant="primary" disabled>Disabled</Button>
    <Button variant="primary" loading="Deploying…">Deploy</Button>
    <Button variant="secondary">Redeploy</Button>
    <Button variant="danger">Delete site</Button>
    <Button variant="ghost" aria-label="More actions">⋯</Button>
    <Button variant="link">View all</Button>
    <Button variant="secondary" size="sm">Small</Button>
  </div>
);

/* ── Badge / StatusBadge ────────────────────────────────── */
export const BadgeVariants = () => (
  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
    <Badge variant="neutral">Email</Badge>
    <Badge variant="accent">PK</Badge>
    <Badge variant="info">Public</Badge>
    <StatusBadge status="ready" />
    <StatusBadge status="building" />
    <StatusBadge status="failed" />
    <StatusBadge status="pending" />
  </div>
);

/* ── Input (default / error / disabled) ─────────────────── */
export const InputStates = () => {
  const [email, setEmail] = useState('not-an-email');
  const valid = email.includes('@');
  return (
    <div style={{ display: 'grid', gap: 16, maxWidth: 320 }}>
      <Input label="Domain" placeholder="www.yourdomain.com" hint="We'll issue a certificate automatically." />
      <Input
        label="Email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        error={valid ? undefined : 'Enter a valid email address.'}
      />
      <Input label="Instance ID" value="bb_3f9a21" mono readOnly disabled />
      <Input label="Endpoint" prefix="POST" mono defaultValue="fn.bigbase.local/resize" />
    </div>
  );
};

/* ── Card + EmptyState ──────────────────────────────────── */
export const CardAndEmpty = () => (
  <div style={{ display: 'grid', gap: 16, maxWidth: 420 }}>
    <Card title="Build settings" interactive>
      <p style={{ color: 'var(--fg-secondary)', fontSize: 14 }}>vite build · output dist/</p>
    </Card>
    <EmptyState
      icon={<span aria-hidden="true">🚀</span>}
      title="Create your first site"
      action={<Button variant="primary">Create site</Button>}
    >
      Connect a Git repository and BigBase builds, deploys, and serves it — auto-redeploying on every push.
    </EmptyState>
  </div>
);

/* Storybook CSF default export (optional; ignored outside Storybook). */
const meta = { title: 'BigBase/Components' };
export default meta;
