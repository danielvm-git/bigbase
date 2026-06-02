---
id: e17s08
title: Dashboard primitives
status: done
legacy_slice: "017-H"
tasks:
  - desc: Build MetricCard — reusable stat card with label, value, trend, color
    verify: "cd ui && grep -q 'export.*MetricCard' src/components/MetricCard.tsx && npm run build"
  - desc: Build RequestChart — horizontal bar chart from status code buckets
    verify: "cd ui && grep -q 'export.*RequestChart' src/components/RequestChart.tsx && npm run build"
  - desc: Build ComponentHealthGrid — grid of named components with health status
    verify: "cd ui && grep -q 'export.*ComponentHealthGrid' src/components/ComponentHealthGrid.tsx && npm run build"
  - desc: Build QuickActions — button strip with icon+label+link actions
    verify: "cd ui && grep -q 'export.*QuickActions' src/components/QuickActions.tsx && npm run build"
  - desc: Write MetricCard.test.tsx covering all props and edge cases
    verify: "cd ui && npx vitest run src/components/MetricCard.test.tsx"
  - desc: Write RequestChart.test.tsx covering bar rendering and empty state
    verify: "cd ui && npx vitest run src/components/RequestChart.test.tsx"
  - desc: Write ComponentHealthGrid.test.tsx covering healthy/degraded/warning states
    verify: "cd ui && npx vitest run src/components/ComponentHealthGrid.test.tsx"
  - desc: Write QuickActions.test.tsx covering action list, click handler, disabled state
    verify: "cd ui && npx vitest run src/components/QuickActions.test.tsx"
  - desc: Export all 4 from components/index.ts barrel
    verify: "cd ui && grep -q 'MetricCard' src/components/index.ts && grep -q 'RequestChart' src/components/index.ts && grep -q 'ComponentHealthGrid' src/components/index.ts && grep -q 'QuickActions' src/components/index.ts"
  - desc: Full suite — all 4 components build + all tests pass with coverage
    verify: "cd ui && npm test -- MetricCard RequestChart ComponentHealthGrid QuickActions -- --coverage"

context: |
  The DashboardPage currently renders metrics and charts inline (via
  DashboardMetrics.tsx which composites multiple stats). This story extracts 4
  reusable primitive components: MetricCard, RequestChart, ComponentHealthGrid,
  QuickActions. None of these components exist yet. They will be used in
  DashboardPage and potentially other monitoring views.
---

## Implementation Steps

**Context**: `DashboardMetrics.tsx` renders a grid of `Card` components with inline JSX for request rate, error rate, CPU, and component count. This story extracts those into 4 standalone reusable primitives. Each primitive must be individually testable without mocking the dashboard API layer.

### Step 1: Build MetricCard

Create `ui/src/components/MetricCard.tsx`:
```tsx
interface MetricCardProps {
  label: string;        // "Request Rate", "CPU", etc.
  value: string | number;
  subtitle?: string;     // "total requests", "goroutines"
  trend?: 'up' | 'down' | 'flat';
  color?: 'success' | 'warning' | 'error' | 'neutral';
  secondaryValue?: string;
}
```

Renders a `Card` with `CardHeader` for the label, a large stat value with CSS
class `stat-value`, optional subtitle, and optional trend arrow. Color prop
controls the value text color via CSS custom properties.

→ verify: `cd ui && grep -q 'export.*MetricCard' src/components/MetricCard.tsx && npm run build`

### Step 2: Build RequestChart

Create `ui/src/components/RequestChart.tsx`:
```tsx
interface RequestChartProps {
  byStatus: Record<string, number>;  // {"200": 142, "404": 3, "500": 1}
  total: number;
}
```

Renders a horizontal stacked bar chart using nested `<div>` elements with CSS
widths proportional to each status code count. Colors: 2xx=green, 3xx=blue,
4xx=amber, 5xx=red. Shows "No requests" when total=0. Uses CSS custom
properties for colors. No chart library dependency — pure CSS bars.

→ verify: `cd ui && grep -q 'export.*RequestChart' src/components/RequestChart.tsx && npm run build`

### Step 3: Build ComponentHealthGrid

Create `ui/src/components/ComponentHealthGrid.tsx`:
```tsx
interface ComponentHealth {
  name: string;
  status: 'healthy' | 'degraded' | 'down' | 'unknown';
  version?: string;
}
interface ComponentHealthGridProps {
  components: ComponentHealth[];
}
```

Renders a responsive grid of health cards (2 columns desktop, 1 column mobile).
Each card shows component name, status badge (colored), and optional version.
Empty state: "No component data available".

→ verify: `cd ui && grep -q 'export.*ComponentHealthGrid' src/components/ComponentHealthGrid.tsx && npm run build`

### Step 4: Build QuickActions

Create `ui/src/components/QuickActions.tsx`:
```tsx
interface QuickAction {
  icon: string;    // emoji or component name
  label: string;
  link: string;    // route path
  disabled?: boolean;
}
interface QuickActionsProps {
  actions: QuickAction[];
  onAction: (link: string) => void;
}
```

Renders a horizontal button strip (wrapping). Each button shows icon + label,
calls `onAction(link)` on click. Disabled buttons are grayed out. Empty array
shows "No quick actions" placeholder.

→ verify: `cd ui && grep -q 'export.*QuickActions' src/components/QuickActions.tsx && npm run build`

### Step 5: Write MetricCard.test.tsx

Pattern: `Badge.test.tsx`
- Renders label, value, and subtitle
- Applies success/warning/error colors
- Shows trend arrow (↑ for up, ↓ for down)
- Shows secondaryValue when provided
- Handles numeric zero (0) correctly

→ verify: `cd ui && npx vitest run src/components/MetricCard.test.tsx`

### Step 6: Write RequestChart.test.tsx

- Renders bars for each status code bucket
- Bar widths proportional to count vs total
- Shows "No requests" when total is 0
- Colors map: "200" → green, "500" → red
- Handles missing keys gracefully

→ verify: `cd ui && npx vitest run src/components/RequestChart.test.tsx`

### Step 7: Write ComponentHealthGrid.test.tsx

- Renders all components with correct status badges
- Maps status to variant: healthy→success, degraded→warning, down→error, unknown→neutral
- Shows version when provided
- Empty array shows placeholder
- Responsive: renders with CSS grid class

→ verify: `cd ui && npx vitest run src/components/ComponentHealthGrid.test.tsx`

### Step 8: Write QuickActions.test.tsx

- Renders all action buttons with icons and labels
- Clicking a button calls `onAction` with the link
- Clicking a disabled button does not call `onAction`
- Empty actions shows placeholder
- Multiple actions render in horizontal layout

→ verify: `cd ui && npx vitest run src/components/QuickActions.test.tsx`

### Step 9: Barrel exports

Add all 4 new exports to `ui/src/components/index.ts`.

→ verify: `cd ui && grep -q 'MetricCard' src/components/index.ts && grep -q 'RequestChart' src/components/index.ts && grep -q 'ComponentHealthGrid' src/components/index.ts && grep -q 'QuickActions' src/components/index.ts`

### Step 10: Full verify

→ verify: `cd ui && npm test -- MetricCard RequestChart ComponentHealthGrid QuickActions -- --coverage`

## Verification Script (Manual)

1. `cd ui && npm run build` — builds cleanly
2. `cd ui && npx vitest run src/components/MetricCard.test.tsx` — green
3. `cd ui && npx vitest run src/components/RequestChart.test.tsx` — green
4. `cd ui && npx vitest run src/components/ComponentHealthGrid.test.tsx` — green
5. `cd ui && npx vitest run src/components/QuickActions.test.tsx` — green
6. `cd ui && npm test -- MetricCard RequestChart ComponentHealthGrid QuickActions -- --coverage` — all pass

## Out of scope
- Integrating these primitives into DashboardPage (done in e17s09)
- Real-time WebSocket updates for health grid
- Animated chart transitions
- Exporting these for external use (library packaging)

## Risks
- RequestChart: CSS-only bars mean no SVG/Canvas — verify they render correctly across browsers
- ComponentHealthGrid: status-to-color mapping must match `statusBadgeVariant` conventions
- QuickActions: route navigation is tested via mock `onAction`, not via router
