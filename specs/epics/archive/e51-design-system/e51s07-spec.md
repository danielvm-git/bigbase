# e51s07: Data Visualization & Editor Components

**Story ID:** e51s07 | **Epic:** e51 | **BCPs:** 4 | **Status:** planned
**Type:** feat | **Context:** domain
**Depends on:** e51s02 (Core Components) | **Blocks:** e61 (Usage Dashboard)

## §1 — Summary

Extend the admin console's data visualization toolkit with reusable chart
components beyond the existing Sparkline, DonutGauge, and BarGauge. Add
`AreaChart` (time-series stacked area), `CodeBlock` (syntax-highlighted code
display), `JsonTree` (collapsible JSON viewer), and enhance the existing
`BarGauge` to support grouped/stacked bars. All components use SVG rendering
(no external charting library) and consume the design token system (e51s01).
Build with accessibility: charts include data tables for screen readers.

## §2 — Motivation

The Monitoring page, Usage Dashboard (e61), and build logs already need richer
data visualization than the current sparklines and simple gauges. Instead of
adding a heavy charting library (recharts ~140KB), we extend our lightweight
SVG approach — consistent with the existing chart components, zero extra deps,
and full control over token integration and accessibility.

## §3 — Background / Context

- Existing chart components (all SVG-based, zero deps):
  - `Sparkline` — tiny inline line chart
  - `DonutGauge` — circular progress indicator
  - `BarGauge` — horizontal/vertical single-series bar
  - `RequestChart` — request volume visualization (limited)
- Existing: `TerminalLogViewer` — terminal-style log output
- Existing: `StreamLog` — real-time streaming log display
- Monitoring page consumes these for CPU/memory metrics
- Design tokens (e51s01) provide brand colors, neutral scale, semantic colors

## §4 — Zoom-Out Check

- **Module purpose**: Admin console data visualization layer
- **Callers**: MonitoringPage, DashboardPage (system metrics), SiteDetailPage (deploy metrics), Usage Dashboard (e61)
- **Contracts**: SVG-rendered React components; accept data as typed arrays; render within container dimensions; support light/dark theme via CSS custom properties

## §5 — Prior Art

- Existing `Sparkline`, `DonutGauge`, `BarGauge` set the pattern: pure SVG, zero deps
- Recharts, visx, nivo are popular React charting libs but add 100-200KB bundle
- For admin dashboards with simple metric displays, SVG primitives are sufficient

## §6 — Design Decisions

| Decision | Rationale |
|----------|-----------|
| SVG rendering, no charting library | Matches existing pattern; admin dashboards have simple charts; keep bundle small |
| `CodeBlock`: Prism.js or highlight.js for syntax HL? | Neither — use CSS-only with semantic class names (`.code-block .code-keyword`). For real syntax highlighting, defer to e61 which can add a lightweight highlighter if needed. |
| `JsonTree`: recursive React component | JSON data explorer with collapsible nodes — pure React, no library |
| `AreaChart`: SVG `<path>` with filled area | Use cubic bezier smoothing; accept time-series data |
| Accessibility: SVG charts include `<table>` alternative | Charts are visual; screen readers get a data table with the same numbers |

## §7 — Architecture / Component Design

```
ui/src/components/
  AreaChart.tsx         ← NEW: time-series area chart with optional multiple series
  AreaChart.test.tsx    ← NEW
  CodeBlock.tsx         ← NEW: styled code display (no syntax highlighting)
  CodeBlock.test.tsx    ← NEW
  JsonTree.tsx          ← NEW: collapsible JSON viewer
  JsonTree.test.tsx     ← NEW

ui/src/components/
  BarGauge.tsx          ← ENHANCE: add grouped/stacked mode
  BarGauge.test.tsx     ← ENHANCE: add tests for new modes
```

## §8 — Data Model / Types

```typescript
// AreaChart
interface AreaChartProps {
  data: {
    series: {
      name: string
      color?: string       // defaults to brand accent gradient
      points: { x: string | number; y: number }[]
    }[]
  }
  width?: number
  height?: number
  showLegend?: boolean
  showGrid?: boolean
  title?: string
  /** Accessibility: table with same data for screen readers */
  a11yTable?: boolean      // default true
}

// CodeBlock
interface CodeBlockProps {
  code: string
  language?: string         // for CSS class (e.g., "json", "yaml", "sql", "bash")
  showLineNumbers?: boolean
  maxHeight?: number        // scroll if taller
  className?: string
}

// JsonTree
interface JsonTreeProps {
  data: unknown
  rootLabel?: string
  maxDepth?: number         // auto-collapse beyond this depth
  expandAll?: boolean
  searchTerm?: string       // highlight matching keys
  className?: string
}

// BarGauge (enhanced — existing + new props)
interface BarGaugeProps {
  // existing...
  mode?: 'single' | 'grouped' | 'stacked'  // NEW
  segments?: { label: string; value: number; color?: string }[]  // NEW for grouped/stacked
}
```

## §9 — API / Interface Contract

- All chart components render SVG elements, consuming `currentColor` and CSS custom properties for theming
- `CodeBlock` renders `<pre><code>` with semantic class names
- `JsonTree` renders recursive `<details>`/`<summary>` for collapsible nodes
- All accept `className` and export from `components/index.ts`

## §10 — State Management

- `JsonTree`: internal collapse/expand state per node; `expandAll` prop overrides
- `AreaChart`: pure render from props — no internal state

## §11 — Error Handling

- `JsonTree`: handles circular references gracefully (detects and shows "[Circular]")
- `AreaChart`: renders empty state when data is empty; handles single-point series
- `CodeBlock`: renders empty `<pre>` when code is empty string

## §12 — Testing Strategy

| Component | Tests |
|-----------|-------|
| AreaChart | renders SVG, renders correct number of `<path>` elements for multi-series, renders legend, renders accessible data table, handles empty data, handles single point |
| CodeBlock | renders code text, applies language class, shows line numbers, scrolls at maxHeight |
| JsonTree | renders string/number/boolean/array/object nodes, collapses nested objects, expands on click, handles null, handles circular ref, highlights search matches, respects maxDepth |
| BarGauge (enhanced) | renders grouped bars, renders stacked bars, renders legend for segments, accessibility table |

## §13 — Performance Considerations

- SVG rendering is GPU-accelerated by browsers — performant for admin dashboard data volumes (<1000 points)
- `JsonTree` uses `React.memo` on nodes to avoid re-rendering the entire tree on expand/collapse
- `AreaChart` uses SVG `<path>` with `d` attribute computed once per render (no animation runtime)

## §14 — Security Considerations

- `CodeBlock`: renders code as text content (not innerHTML) — no XSS
- `JsonTree`: renders data as text nodes — no XSS
- All chart components render SVG via React JSX (not dangerouslySetInnerHTML)

## §15 — Accessibility

| Component | A11y requirements |
|-----------|------------------|
| AreaChart | `role="img"`, `aria-label`, child `<table>` with same data for screen readers |
| CodeBlock | `role="region"`, `aria-label` (e.g., "JSON code block"), line numbers are `aria-hidden` |
| JsonTree | `<details>`/`<summary>` for collapsible nodes — native keyboard accessible |
| BarGauge | `role="img"`, `aria-label`, accessible data table |

## §16 — Internationalization (i18n)

No hardcoded strings except aria-labels which accept props.

## §17 — Acceptance Criteria (Gherkin)

```gherkin
Scenario: AreaChart renders multi-series data
  Given an AreaChart with 2 series each having 5 data points
  When rendered
  Then two SVG path areas are visible and a legend shows both series names

Scenario: AreaChart includes accessible data table
  Given an AreaChart with data
  When rendered
  Then a visually-hidden table contains the same data values

Scenario: CodeBlock renders with line numbers
  Given a CodeBlock with code="line1\nline2" and showLineNumbers=true
  When rendered
  Then two lines are displayed with "1" and "2" line numbers

Scenario: JsonTree expands and collapses
  Given a JsonTree with {a: {b: 1}}
  When rendered
  Then "b: 1" is hidden
  When the user clicks the expand toggle for "a"
  Then "b: 1" becomes visible

Scenario: JsonTree handles circular references
  Given a JsonTree with a circular reference
  When rendered
  Then "[Circular]" is displayed instead of an infinite loop

Scenario: BarGauge renders grouped bars
  Given a BarGauge with mode="grouped" and 3 segments
  When rendered
  Then 3 bars are visible side by side with correct proportional widths
```

## §18 — Verification Script (Step-by-Step)

1. Run new component tests: `cd ui && npx vitest run src/components/AreaChart.test.tsx src/components/CodeBlock.test.tsx src/components/JsonTree.test.tsx`
2. Run enhanced BarGauge tests: `cd ui && npx vitest run src/components/BarGauge.test.tsx`
3. Run all tests: `cd ui && npm test`
4. Type check: `cd ui && npx tsc --noEmit`
5. Build UI: `cd ui && npm run build`
6. Build Go: `cd .. && go build ./...`

## §19 — Out of Scope

- Real-time updating charts (WebSocket streaming)
- Interactive chart features (zoom, pan, tooltip on hover)
- Export charts as PNG/SVG
- Gauge/radial chart
- Scatter plot / bubble chart
- Pie chart (DonutGauge covers this use case)
- Heatmap
- Treemap / sankey diagram

## §20 — Risks

| Risk | Mitigation |
|------|-----------|
| SVG path calculation errors for edge cases (single point, all zeros) | Test with edge case data explicitly |
| JsonTree performance with deeply nested objects (>1000 nodes) | Virtualize with maxDepth; collapse deep nodes by default |
| BarGauge grouped/stacked changes break existing BarGauge usage | Add new props with defaults; existing single-series usage unchanged |
| CodeBlock without syntax highlighting looks plain | Design uses subtle CSS coloring — enough for JSON/YAML readability; full highlighting deferred to e61 |
