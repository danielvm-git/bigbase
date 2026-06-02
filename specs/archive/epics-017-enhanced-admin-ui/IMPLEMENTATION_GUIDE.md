# BigBase Console - Prototype Implementation Guide

**Status**: Ready for Development  
**Prototype File**: `BigBase Console.html`  
**Target Technology**: React + TypeScript  

## Overview

This document guides the implementation of the BigBase Console prototype as a production-ready web application. The prototype is feature-complete and ready to be ported to a real tech stack.

## What the Prototype Includes

✅ **Included**:
- Complete UI design system (colors, typography, spacing)
- All 8 main screens fully designed
- Component library (24 primitives)
- Dark mode implementation
- Responsive design (desktop, tablet, mobile)
- Mock data structures and sample content
- Navigation and routing logic
- Form interactions and validations
- Toast notification system
- Status badges and loading states

❌ **Not Included** (Plan in EPIC 2+):
- Backend API integration
- Real authentication
- Database queries
- File uploads
- Git OAuth flow
- WebSocket/realtime updates

## Implementation Architecture

### Recommended Tech Stack

```
Frontend:
├─ Framework: React 18+ with TypeScript
├─ Build Tool: Vite or Next.js
├─ Styling: CSS-in-JS (Emotion, styled-components) or Tailwind
├─ UI Library: Radix UI (for accessibility) or keep custom
├─ HTTP Client: Fetch API + React Query
├─ Router: React Router v6 or custom hash router
├─ State: React Context + useReducer or Zustand
└─ Icons: React Icon components (keep Lucide)

Backend (Future):
├─ API Framework: Go + Chi/Gin or TypeScript + Express
├─ Database: PostgreSQL 14+
├─ Authentication: JWT + OAuth 2.0
├─ Cache: Redis for sessions and metrics
└─ File Storage: S3-compatible (Minio, AWS S3)
```

## File Organization

```
bigbase-console/
├─ src/
│  ├─ components/
│  │  ├─ primitives/         (Button, Input, Card, etc.)
│  │  ├─ screens/            (Dashboard, Sites, SQL, etc.)
│  │  ├─ layout/             (Sidebar, Header, Container)
│  │  └─ common/             (Icon, Avatar, Badge, Toast)
│  ├─ pages/
│  │  ├─ Dashboard.tsx
│  │  ├─ Sites.tsx
│  │  ├─ SqlEditor.tsx
│  │  ├─ Storage.tsx
│  │  ├─ Users.tsx
│  │  ├─ Repos.tsx
│  │  ├─ Pipelines.tsx
│  │  └─ Monitoring.tsx
│  ├─ hooks/
│  │  ├─ useToast.ts
│  │  ├─ useTheme.ts
│  │  └─ useRouter.ts
│  ├─ context/
│  │  ├─ ToastContext.tsx
│  │  ├─ ThemeContext.tsx
│  │  └─ DataContext.tsx
│  ├─ types/
│  │  ├─ index.ts            (TypeScript interfaces)
│  │  └─ api.ts              (API response types)
│  ├─ utils/
│  │  ├─ format.ts           (timeAgo, statusVariant)
│  │  ├─ data.ts             (mock data)
│  │  └─ colors.ts           (theme tokens)
│  ├─ styles/
│  │  ├─ globals.css         (design system tokens)
│  │  ├─ components.css      (component styles)
│  │  └─ theme.css           (dark mode)
│  └─ App.tsx                (main app shell)
├─ public/
│  └─ icons/                 (favicon, logo)
├─ tests/
│  ├─ components/
│  ├─ pages/
│  └─ utils/
├─ package.json
├─ tsconfig.json
└─ README.md
```

## Component Migration Path

### Phase 1: Extract Components from Prototype
1. Copy HTML/CSS from `BigBase Console.html`
2. Convert to React components
3. Extract design tokens to CSS variables
4. Create component library documentation

### Phase 2: Build Component Library
1. Implement each primitive component:
   - Button, Input, Card, Badge, Avatar
   - Tables, Tabs, Modals, Forms
   - Icons, Spinners, Alerts, Toasts
2. Test in isolation with Storybook
3. Document with examples

### Phase 3: Implement Screens
1. Create page components for each screen
2. Integrate primitives into layouts
3. Add routing (React Router or custom)
4. Connect mock data

### Phase 4: State Management
1. Set up Context API or Zustand
2. Implement theme persistence
3. Create toast provider
4. Prepare for API integration

### Phase 5: API Integration
1. Define API contract (OpenAPI schema)
2. Replace mock data with API calls
3. Add React Query for data fetching
4. Implement error handling

## Key Design Tokens to Extract

### Colors
```typescript
const colors = {
  brand: {
    50: '#eeecfc',
    100: '#e0ddf a',
    500: '#4f46e5',
    600: '#4338ca',
    700: '#3730a3',
  },
  neutral: {
    0: '#ffffff',
    25: '#fafafb',
    50: '#edede0',
    // ... 25 levels total
  },
  semantic: {
    success: '#10b981',
    warning: '#f59e0b',
    error: '#ef4444',
    info: '#3b82f6',
  },
};
```

### Typography
```typescript
const typography = {
  display: { size: '40px', weight: 700 },
  h1: { size: '24px', weight: 600 },
  h2: { size: '20px', weight: 600 },
  body: { size: '16px', weight: 400 },
  caption: { size: '12px', weight: 400 },
};
```

### Spacing
```typescript
const space = {
  0: '0',
  1: '2px',
  2: '4px',
  3: '6px',
  4: '8px',
  // ... up to space-32: '64px'
};
```

## Data Types to Implement

```typescript
interface Site {
  id: string;
  name: string;
  framework: 'Astro' | 'Next.js' | 'Vite' | 'SvelteKit' | 'Static';
  repo: string;
  branch: string;
  status: 'ready' | 'building' | 'failed';
  url: string;
  commit: string;
  updated: Date;
  deployments: Deployment[];
}

interface Deployment {
  id: string;
  status: 'ready' | 'building' | 'failed';
  commit: string;
  commitMsg: string;
  when: Date;
  duration: string;
  logs: string[];
}

interface User {
  id: string;
  email: string;
  role: 'owner' | 'admin' | 'member';
  verified: boolean;
  created: Date;
}

interface Function {
  id: string;
  name: string;
  runtime: string;
  trigger: 'HTTP' | 'Schedule';
  timeout: number;
  status: 'active' | 'inactive';
  updated: Date;
}
```

## API Endpoints to Mock First

```typescript
// Mock API responses matching these endpoints
const API_ENDPOINTS = {
  sites: {
    list: 'GET /api/v1/sites',
    create: 'POST /api/v1/sites',
    get: 'GET /api/v1/sites/:id',
  },
  deployments: {
    list: 'GET /api/v1/deployments',
    get: 'GET /api/v1/deployments/:id',
    logs: 'GET /api/v1/deployments/:id/logs',
  },
  users: {
    list: 'GET /api/v1/users',
    invite: 'POST /api/v1/users',
  },
  repos: {
    list: 'GET /api/v1/repos',
  },
  pipelines: {
    list: 'GET /api/v1/pipelines',
  },
  sql: {
    query: 'POST /api/v1/sql/query',
  },
  storage: {
    buckets: 'GET /api/v1/storage/buckets',
    files: 'GET /api/v1/storage/:bucket/files',
  },
};
```

## Important Implementation Details

### 1. Routing Strategy
```typescript
// Use hash routing for SPA simplicity
const routes = {
  '/': Dashboard,
  '/sites': Sites,
  '/sites/:id': SiteDetail,
  '/sql': SqlEditor,
  '/storage': Storage,
  '/users': Users,
  '/repos': Repos,
  '/pipelines': Pipelines,
  '/monitoring': Monitoring,
};
```

### 2. Theme Toggle Implementation
```typescript
// Persist theme to localStorage
function useTheme() {
  const [theme, setTheme] = useState('light');
  
  useEffect(() => {
    const saved = localStorage.getItem('bigbase-theme');
    if (saved) setTheme(saved);
  }, []);
  
  const toggleTheme = (t: string) => {
    setTheme(t);
    document.documentElement.setAttribute('data-theme', t);
    localStorage.setItem('bigbase-theme', t);
  };
  
  return [theme, toggleTheme];
}
```

### 3. Toast Notifications
```typescript
// Use context provider at root level
function App() {
  return (
    <ToastProvider>
      <Main />
    </ToastProvider>
  );
}

// Use anywhere
const toast = useToast();
toast.push({
  type: 'success',
  title: 'Deployment started',
  msg: 'Build logs available on deployment page',
  duration: 4200,
});
```

### 4. Form Handling
```typescript
// Use React Hook Form for validation
const form = useForm<CreateSiteInput>({
  mode: 'onBlur',
  defaultValues: { ... },
});

// Validate site name: alphanumeric + hyphens only
const validateSiteName = (name: string) => {
  return /^[a-z0-9-]+$/.test(name) || 'Invalid site name';
};
```

### 5. Build Log Streaming
```typescript
// Fetch logs and stream to terminal-style display
async function* streamBuildLogs(deploymentId: string) {
  const response = await fetch(`/api/v1/deployments/${deploymentId}/logs`);
  const reader = response.body?.getReader();
  // Stream lines to UI in real-time
}
```

## Testing Checklist

### Unit Tests
- [ ] Button components render correctly
- [ ] Input validation works
- [ ] Theme toggle persists
- [ ] Toast notifications display

### Integration Tests
- [ ] Dashboard loads without errors
- [ ] Site creation wizard completes
- [ ] Deployment triggers and updates status
- [ ] Dark mode affects all screens
- [ ] Mobile layout responds correctly

### E2E Tests (Future)
- [ ] Full user signup flow
- [ ] Create site end-to-end
- [ ] Deploy and view logs
- [ ] Invite user to team

## Accessibility Checklist

- [ ] All buttons have text labels or aria-label
- [ ] Form inputs have associated labels
- [ ] Color contrast ≥ 4.5:1 everywhere
- [ ] Focus rings visible on keyboard navigation
- [ ] Status badges announce with screen readers
- [ ] Icons have semantic meaning or ARIA
- [ ] Modal dialogs trap focus

## Performance Optimization

### Initial Load
1. Code split by route
2. Lazy load screens with React.lazy()
3. Cache component library in static files
4. Compress SVG icons

### Runtime Performance
1. Memoize expensive components
2. Use React.memo for list items
3. Implement virtual scrolling for large tables
4. Debounce search/filter inputs

### Bundle Size
- React: ~40KB gzipped
- Component CSS: ~20KB gzipped
- App code: ~50KB gzipped
- **Target**: <150KB total

## Deployment Strategy

### Development
```bash
npm install
npm run dev          # Vite dev server
npm run test         # Run tests
```

### Build
```bash
npm run build        # Production build
npm run build:analyze # Check bundle size
```

### Deploy
```bash
# Docker container
docker build -t bigbase-console:latest .
docker run -p 3000:3000 bigbase-console

# Or static host (Vercel, Netlify, etc.)
npm run build
npm run deploy
```

## Integration with Backend

### Phase 1: Mock API
```typescript
// api/client.ts
export async function fetchSites(): Promise<Site[]> {
  // Return mock data
  return DATA.SITES;
}
```

### Phase 2: Real API
```typescript
// api/client.ts
export async function fetchSites(): Promise<Site[]> {
  const res = await fetch('/api/v1/sites', {
    headers: { Authorization: `Bearer ${token}` },
  });
  return res.json();
}
```

### Phase 3: React Query
```typescript
// hooks/useSites.ts
export function useSites() {
  return useQuery({
    queryKey: ['sites'],
    queryFn: fetchSites,
    staleTime: 60000,
  });
}

// Component
function SitesList() {
  const { data: sites, isLoading } = useSites();
  return isLoading ? <Skeleton /> : <Sites data={sites} />;
}
```

## Monitoring & Debugging

### Browser DevTools
- React DevTools extension
- Chrome Network tab for API calls
- Console for errors and logs

### Error Tracking
- Sentry integration (future)
- Error boundaries for crashed components
- User session replay (future)

### Performance Monitoring
- Lighthouse CI
- Web Vitals tracking
- Bundle analysis

## Success Criteria

✅ **Implementation Complete When**:
1. All 8 screens render without errors
2. Navigation works smoothly
3. Dark mode toggle persists
4. Responsive design works on mobile
5. Component library documented
6. Unit tests cover primitives
7. Integration tests cover workflows
8. Accessibility audit passes WCAG 2.1 AA
9. Bundle size < 150KB gzipped
10. Lighthouse score > 90

## Timeline

- **Week 1-2**: Project setup, design tokens, component library
- **Week 3-4**: Primitive components (Button, Input, Card, etc.)
- **Week 5-6**: Layout and screen components
- **Week 7**: Routing, state management, theme toggle
- **Week 8**: Testing, accessibility, performance
- **Week 9+**: Backend integration (EPIC 2+)

## References

- **Design File**: `BigBase Console.html` (this repo)
- **Design System Doc**: `SYSTEM_DESIGN.md`
- **Component Inventory**: `COMPONENT_INVENTORY.md`
- **Release Plan**: `../docs/RELEASE_PLAN.md`
- **Prototype Chat**: `../chats/chat1.md`

## Questions & Support

For implementation questions, refer to:
1. COMPONENT_INVENTORY.md for component specs
2. SYSTEM_DESIGN.md for architecture
3. BigBase Console.html for visual reference
4. RELEASE_PLAN.md for scope and timeline
