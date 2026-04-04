---
name: frontend-bundle-optimization
description: Frontend bundle optimization for React/Next.js — code splitting, dynamic imports, tree-shaking, bundle analysis, image/font optimization, and Lighthouse CI.
origin: vibemenu
---

# Frontend Bundle Optimization

Large JavaScript bundles slow first-load performance. This skill covers the concrete techniques to reduce bundle size, improve code splitting, and enforce performance budgets in CI.

## When to Activate

- First Contentful Paint (FCP) or Largest Contentful Paint (LCP) is slow
- Lighthouse Performance score is below 90
- Bundle size has grown beyond 250KB (compressed JS for initial load)
- Adding large third-party libraries (charting, rich text editors, date pickers)
- Setting up performance monitoring in CI/CD

## Next.js App Router Code Splitting

Every `page.tsx` is automatically a route segment boundary — Next.js creates a separate chunk for each page. No manual webpack config needed:

```
app/
├── page.tsx          → / route chunk
├── dashboard/
│   └── page.tsx      → /dashboard chunk (not loaded until user navigates here)
└── settings/
    └── page.tsx      → /settings chunk
```

**What's included in the initial bundle:**
- Layout files (`layout.tsx`) shared across routes
- Components imported directly in the root layout
- Shared providers (auth, theme, query client)

**Minimize the root layout bundle:**
```tsx
// app/layout.tsx
// ✅ GOOD: Only critical global styles and providers
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <AuthProvider>   {/* lightweight — just context */}
          {children}
        </AuthProvider>
      </body>
    </html>
  );
}

// ❌ BAD: Heavy libraries imported at root layout level (bundled into every page)
import { FullPageDashboard } from '@/components/Dashboard'; // pulls in charting library
```

## Dynamic Imports

Use `next/dynamic` for components not needed on the initial render:

```tsx
import dynamic from 'next/dynamic';
import { Skeleton } from '@/components/ui/skeleton';

// ✅ Heavy chart library — only loads when component mounts
const RevenueChart = dynamic(
  () => import('@/components/charts/RevenueChart'),
  {
    loading: () => <Skeleton className="h-64 w-full" />,
    ssr: false, // chart libraries often use window/canvas APIs
  }
);

// ✅ Modal — not in initial viewport, load on demand
const DeleteConfirmModal = dynamic(
  () => import('@/components/modals/DeleteConfirmModal'),
  { ssr: false }
);

// ✅ Rich text editor (typically 200KB+)
const RichTextEditor = dynamic(
  () => import('@/components/editor/RichTextEditor'),
  {
    loading: () => <div className="h-48 animate-pulse bg-muted rounded-md" />,
    ssr: false,
  }
);

// Usage — same as regular component
export function ProductPage() {
  const [showEditor, setShowEditor] = useState(false);

  return (
    <div>
      <RevenueChart />
      {showEditor && <RichTextEditor />}
    </div>
  );
}
```

**When to use `ssr: false`:**
- Browser-only APIs: `window`, `document`, `navigator`, `localStorage`
- Canvas or WebGL rendering libraries
- Drag-and-drop libraries
- Components that read viewport dimensions on mount

## Tree-Shaking

### Package.json Sideeffects

```json
{
  "name": "my-app",
  "sideEffects": false
}
```

For packages with CSS side effects:

```json
{
  "sideEffects": ["*.css", "*.scss"]
}
```

### Named Imports — Never Import Entire Libraries

```tsx
// ❌ BAD: Imports entire lodash (~70KB gzipped)
import _ from 'lodash';
const result = _.groupBy(items, 'category');

// ✅ GOOD: Import only the function used (~1KB)
import groupBy from 'lodash/groupBy';
// or
import { groupBy } from 'lodash-es'; // tree-shakeable ESM build

// ❌ BAD: Entire MUI imports defeat tree-shaking
import * as MUI from '@mui/material';
const { Button, TextField } = MUI;

// ✅ GOOD: Named imports allow tree-shaking
import { Button } from '@mui/material/Button';
import { TextField } from '@mui/material/TextField';
// or
import Button from '@mui/material/Button';

// ❌ BAD: Entire date-fns (but lodash-style imports work)
import * as dateFns from 'date-fns';

// ✅ GOOD: Named imports from date-fns (it's already tree-shakeable)
import { format, parseISO, differenceInDays } from 'date-fns';
```

### Avoid Barrel Re-Exports for Large Modules

```ts
// ❌ BAD: components/index.ts re-exports everything
export { Button } from './Button';
export { Input } from './Input';
export { DataTable } from './DataTable';   // pulls in heavy table library
export { RichEditor } from './RichEditor'; // pulls in ProseMirror

// When you import Button, you pull in DataTable and RichEditor too
import { Button } from '@/components';

// ✅ GOOD: Direct imports (no barrel that includes unrelated heavy components)
import { Button } from '@/components/Button';
import { Input } from '@/components/Input';
```

## Bundle Analyzer Setup

```bash
npm install --save-dev @next/bundle-analyzer
```

```js
// next.config.js
const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
  openAnalyzer: true,
});

/** @type {import('next').NextConfig} */
const nextConfig = {
  // your existing config
};

module.exports = withBundleAnalyzer(nextConfig);
```

```bash
# Run analysis — opens interactive treemap in browser
ANALYZE=true npm run build
```

**What to look for in the treemap:**
1. **Large `node_modules` blocks** — is `moment`, `lodash`, or a charting library included in the initial bundle?
2. **Duplicate packages** — two different versions of `react` or `date-fns` pulled by different packages
3. **Unexpectedly large chunks** — a route chunk that includes code from other routes (likely a barrel import issue)
4. **Libraries not in initial viewport** — if your chart library is in the initial bundle, make it dynamic

## Image Optimization

Always use `next/image` for content images. Never use raw `<img>` tags for anything loaded from your server:

```tsx
import Image from 'next/image';

// ✅ GOOD: Automatic WebP/AVIF, lazy loading, blur placeholder, layout stability
<Image
  src="/hero.jpg"
  alt="Hero image"
  width={1200}
  height={600}
  priority // use priority for above-the-fold images
  placeholder="blur"
  blurDataURL="data:image/jpeg;base64,/9j/4AAQSkZJRgAB..." // generate with plaiceholder
/>

// For dynamic images from CMS/CDN:
<Image
  src={product.imageUrl}
  alt={product.name}
  width={400}
  height={300}
  loading="lazy" // default — omit for above-the-fold
/>

// ❌ BAD: Raw <img> — no optimization, no lazy loading, causes layout shift
<img src="/hero.jpg" />
```

```js
// next.config.js — allow external image domains
const nextConfig = {
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: 'cdn.example.com' },
      { protocol: 'https', hostname: '**.cloudinary.com' },
    ],
    formats: ['image/avif', 'image/webp'], // prefer AVIF, fallback to WebP
  },
};
```

## Font Optimization

Use `next/font` — fonts are self-hosted at build time, eliminating layout shift and external requests:

```tsx
// app/layout.tsx
import { Inter, Fira_Code } from 'next/font/google';

const inter = Inter({
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-inter',
});

const firaCode = Fira_Code({
  subsets: ['latin'],
  weight: ['400', '500'],
  variable: '--font-fira-code',
});

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${inter.variable} ${firaCode.variable}`}>
      <body className="font-sans">{children}</body>
    </html>
  );
}
```

```tsx
// Local font
import localFont from 'next/font/local';

const customFont = localFont({
  src: './fonts/CustomFont.woff2',
  variable: '--font-custom',
  display: 'swap',
});
```

## Lighthouse CI

```bash
npm install --save-dev @lhci/cli
```

```json
// .lighthouserc.json
{
  "ci": {
    "collect": {
      "url": ["http://localhost:3000/", "http://localhost:3000/dashboard"],
      "startServerCommand": "npm run start",
      "numberOfRuns": 3
    },
    "assert": {
      "budgets": [
        {
          "path": "/*",
          "resourceSizes": [
            { "resourceType": "script", "budget": 300 },
            { "resourceType": "total", "budget": 500 }
          ],
          "resourceCounts": [
            { "resourceType": "third-party", "budget": 5 }
          ]
        }
      ],
      "preset": "lighthouse:recommended",
      "assertions": {
        "categories:performance": ["warn", { "minScore": 0.9 }],
        "categories:accessibility": ["error", { "minScore": 0.9 }],
        "first-contentful-paint": ["warn", { "maxNumericValue": 2000 }],
        "largest-contentful-paint": ["error", { "maxNumericValue": 3000 }],
        "cumulative-layout-shift": ["error", { "maxNumericValue": 0.1 }],
        "total-blocking-time": ["warn", { "maxNumericValue": 300 }]
      }
    },
    "upload": {
      "target": "temporary-public-storage"
    }
  }
}
```

```yaml
# .github/workflows/lighthouse.yml
- name: Run Lighthouse CI
  run: |
    npm ci
    npm run build
    lhci autorun
  env:
    LHCI_GITHUB_APP_TOKEN: ${{ secrets.LHCI_GITHUB_APP_TOKEN }}
```

## Module Federation (Micro-Frontends — Advanced)

Only use Module Federation when you have **truly separate deployment units** (different teams, different release cycles). The overhead is significant for a monorepo:

```js
// next.config.js — host app (consumer)
const NextFederationPlugin = require('@module-federation/nextjs-mf');

module.exports = {
  webpack(config) {
    config.plugins.push(new NextFederationPlugin({
      name: 'host',
      remotes: {
        checkout: `checkout@https://checkout.example.com/_next/static/chunks/remoteEntry.js`,
      },
      shared: {},
    }));
    return config;
  },
};
```

```tsx
// Consuming a federated component
import dynamic from 'next/dynamic';

const CheckoutForm = dynamic(
  () => import('checkout/CheckoutForm'),
  { ssr: false }
);
```

## Anti-Patterns

```tsx
// ❌ BAD: Entire react-icons library (10,000+ SVG icons — ~2MB uncompressed)
import * as Icons from 'react-icons/fa';
<Icons.FaUser />

// ✅ GOOD: Import only what you use
import { FaUser } from 'react-icons/fa';

// ❌ BAD: moment.js — 230KB gzipped with all locales
import moment from 'moment';
moment().format('YYYY-MM-DD');

// ✅ GOOD: date-fns — 13KB for individual functions
import { format } from 'date-fns';
format(new Date(), 'yyyy-MM-dd');

// ❌ BAD: lodash default import
import _ from 'lodash';

// ✅ GOOD: lodash-es tree-shakeable
import { debounce } from 'lodash-es';

// ❌ BAD: Using console.log for debugging size issues
// (won't tell you what's causing the problem)

// ✅ GOOD: Use ANALYZE=true next build to see the actual treemap

// ❌ BAD: Loading chart library in SSR
import { LineChart } from 'recharts'; // recharts uses window

// ✅ GOOD:
const LineChart = dynamic(() => import('@/components/charts/LineChart'), { ssr: false });
```
