---
name: frontend-error-boundaries
description: Frontend error boundaries and error handling — React class ErrorBoundary, Next.js App Router error files, Sentry integration, Suspense composition, Vue error handling, and fallback UI patterns.
origin: vibemenu
---

# Frontend Error Boundaries and Error Handling

React render errors aren't caught by `try/catch` or `useEffect`. Error boundaries are class components that intercept render-phase errors and display a fallback UI instead of crashing the whole page.

## When to Activate

- A component crash takes down the entire page (white screen of death)
- Implementing error tracking with Sentry in a React/Next.js app
- Designing graceful degradation for widget failures
- Adding `error.tsx` / `global-error.tsx` in Next.js App Router

## React Class ErrorBoundary (Required — Hooks Cannot Catch Render Errors)

```tsx
import React from 'react';
import * as Sentry from '@sentry/react';

interface ErrorBoundaryProps {
  fallback: React.ComponentType<{ error: Error; reset: () => void }>;
  children: React.ReactNode;
  /** Optional context tags sent to Sentry */
  context?: Record<string, string>;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    // Update state so the next render shows the fallback UI
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo): void {
    // Report to Sentry with component stack for debugging
    Sentry.captureException(error, {
      extra: {
        componentStack: info.componentStack,
        ...this.props.context,
      },
    });
  }

  reset = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): React.ReactNode {
    if (this.state.hasError && this.state.error) {
      const Fallback = this.props.fallback;
      return <Fallback error={this.state.error} reset={this.reset} />;
    }
    return this.props.children;
  }
}
```

### Fallback UI Component

```tsx
interface ErrorFallbackProps {
  error: Error;
  reset: () => void;
}

export function ErrorFallback({ error, reset }: ErrorFallbackProps) {
  // Generate a reference ID for support — never show raw stack traces
  const errorRef = React.useMemo(
    () => Math.random().toString(36).substring(2, 10).toUpperCase(),
    []
  );

  const isNetworkError = error.message.toLowerCase().includes('network') ||
    error.message.toLowerCase().includes('fetch');

  return (
    <div className="flex flex-col items-center justify-center p-8 text-center space-y-4">
      <h2 className="text-lg font-semibold text-destructive">Something went wrong</h2>
      <p className="text-sm text-muted-foreground max-w-sm">
        {isNetworkError
          ? 'A network error occurred. Check your connection and try again.'
          : 'An unexpected error occurred. If this persists, contact support.'}
      </p>
      <p className="text-xs text-muted-foreground font-mono">
        Reference: {errorRef}
      </p>
      {isNetworkError && (
        <button
          onClick={reset}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm"
        >
          Try again
        </button>
      )}
    </div>
  );
}
```

### Usage

```tsx
// Wrap individual sections — failure in one widget doesn't crash the page
export function DashboardPage() {
  return (
    <div className="grid grid-cols-2 gap-4">
      <ErrorBoundary fallback={ErrorFallback} context={{ section: 'revenue-chart' }}>
        <RevenueChart />
      </ErrorBoundary>

      <ErrorBoundary fallback={ErrorFallback} context={{ section: 'user-table' }}>
        <UserTable />
      </ErrorBoundary>

      {/* Root boundary for catastrophic failures */}
      <ErrorBoundary fallback={FullPageError}>
        <MainContent />
      </ErrorBoundary>
    </div>
  );
}
```

## Suspense + ErrorBoundary Composition

ErrorBoundary must **wrap** Suspense, not be inside it:

```tsx
// ✅ CORRECT: ErrorBoundary wraps Suspense
<ErrorBoundary fallback={ErrorFallback}>
  <Suspense fallback={<LoadingSpinner />}>
    <AsyncDataComponent />
  </Suspense>
</ErrorBoundary>

// ❌ WRONG: Suspense wraps ErrorBoundary (Suspense doesn't catch errors)
<Suspense fallback={<LoadingSpinner />}>
  <ErrorBoundary fallback={ErrorFallback}>
    <AsyncDataComponent />
  </ErrorBoundary>
</Suspense>
```

```tsx
// Complete async component pattern
import { Suspense } from 'react';
import { ErrorBoundary } from '@/components/ErrorBoundary';

export function UserProfile({ userId }: { userId: string }) {
  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <div>
          <p>Failed to load profile</p>
          <button onClick={reset}>Retry</button>
        </div>
      )}
    >
      <Suspense fallback={<ProfileSkeleton />}>
        <UserProfileData userId={userId} />
      </Suspense>
    </ErrorBoundary>
  );
}
```

## Next.js App Router Error Files

### `app/error.tsx` — Segment-Level Error Boundary

```tsx
// app/error.tsx (or app/dashboard/error.tsx for segment-specific)
'use client'; // Error components must be client components

import { useEffect } from 'react';

interface ErrorProps {
  error: Error & { digest?: string }; // digest = server-side error ID
  reset: () => void;                   // retry: re-renders the segment
}

export default function Error({ error, reset }: ErrorProps) {
  useEffect(() => {
    // Log to error tracking service
    console.error('Segment error:', error);
  }, [error]);

  return (
    <div className="flex flex-col items-center gap-4 p-8">
      <h2 className="text-xl font-bold">Something went wrong</h2>
      {error.digest && (
        <p className="text-sm text-muted-foreground font-mono">
          Error ID: {error.digest}
        </p>
      )}
      <button
        onClick={reset}
        className="px-4 py-2 bg-primary text-white rounded"
      >
        Try again
      </button>
    </div>
  );
}
```

### `app/global-error.tsx` — Root Layout Error Boundary

Replaces the root layout when it catches an error — must include `<html>` and `<body>`:

```tsx
// app/global-error.tsx
'use client';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en">
      <body>
        <div className="min-h-screen flex items-center justify-center">
          <div className="text-center space-y-4">
            <h1 className="text-2xl font-bold">Critical Error</h1>
            <p className="text-muted-foreground">
              The application encountered a fatal error.
            </p>
            {error.digest && (
              <p className="font-mono text-sm">ID: {error.digest}</p>
            )}
            <button onClick={reset} className="px-4 py-2 bg-primary text-white rounded">
              Reload
            </button>
          </div>
        </div>
      </body>
    </html>
  );
}
```

### `app/not-found.tsx` — 404 Handler

```tsx
// app/not-found.tsx
import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
      <h2 className="text-2xl font-bold">404 — Page Not Found</h2>
      <p className="text-muted-foreground">The page you're looking for doesn't exist.</p>
      <Link href="/" className="underline text-primary">
        Go home
      </Link>
    </div>
  );
}
```

Trigger programmatically:

```tsx
// In a Server Component or Server Action
import { notFound } from 'next/navigation';

async function ProductPage({ params }: { params: { id: string } }) {
  const product = await getProduct(params.id);
  if (!product) notFound(); // renders not-found.tsx
  return <ProductDetail product={product} />;
}
```

### `app/loading.tsx` — Streaming Loading State

```tsx
// app/loading.tsx — wraps the page in a Suspense boundary automatically
export default function Loading() {
  return (
    <div className="space-y-4 p-8">
      <div className="h-8 w-48 animate-pulse bg-muted rounded" />
      <div className="h-4 w-full animate-pulse bg-muted rounded" />
      <div className="h-4 w-3/4 animate-pulse bg-muted rounded" />
    </div>
  );
}
```

## Sentry Integration

```bash
npm install @sentry/nextjs
npx @sentry/wizard@latest -i nextjs
```

```tsx
// sentry.client.config.ts
import * as Sentry from '@sentry/nextjs';

Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  environment: process.env.NODE_ENV,
  tracesSampleRate: process.env.NODE_ENV === 'production' ? 0.1 : 1.0,
  replaysSessionSampleRate: 0.1,
  replaysOnErrorSampleRate: 1.0,
  integrations: [
    Sentry.replayIntegration({
      maskAllText: true,  // mask PII in session replays
      blockAllMedia: false,
    }),
  ],
});
```

```tsx
// Use Sentry's built-in ErrorBoundary for convenience
import { ErrorBoundary as SentryErrorBoundary } from '@sentry/react';

export function App() {
  return (
    <SentryErrorBoundary
      fallback={({ error, resetError }) => (
        <ErrorFallback error={error as Error} reset={resetError} />
      )}
      beforeCapture={(scope) => {
        scope.setTag('section', 'app-root');
      }}
    >
      <Router />
    </SentryErrorBoundary>
  );
}
```

```tsx
// Manual error capture in event handlers (ErrorBoundary doesn't catch these)
async function handleFormSubmit(data: FormData) {
  try {
    await submitForm(data);
  } catch (error) {
    Sentry.captureException(error, {
      extra: { formAction: 'user-signup' },
    });
    setErrorMessage('Submission failed. Please try again.');
  }
}
```

## Vue 3 Error Handling

### Options API (`errorCaptured`)

```ts
// Parent component catches errors from children
export default {
  errorCaptured(error: Error, instance: ComponentPublicInstance | null, info: string) {
    console.error('Captured error:', error, info);
    // Report to tracking
    Sentry.captureException(error, { extra: { vueInfo: info } });
    // Return false to stop propagation (prevent reaching global handler)
    return false;
  },
};
```

### Composition API (`onErrorCaptured`)

```ts
import { onErrorCaptured, ref } from 'vue';

export default {
  setup() {
    const hasError = ref(false);
    const errorMessage = ref('');

    onErrorCaptured((error: Error, _instance, info: string) => {
      hasError.value = true;
      errorMessage.value = 'An error occurred in this section';
      Sentry.captureException(error, { extra: { vueInfo: info } });
      return false; // stop propagation
    });

    return { hasError, errorMessage };
  },
};
```

### Global Vue Error Handler

```ts
// main.ts
import { createApp } from 'vue';
import * as Sentry from '@sentry/vue';
import App from './App.vue';

const app = createApp(App);

// Global error handler — catches errors not caught by component-level handlers
app.config.errorHandler = (error, instance, info) => {
  console.error('Global Vue error:', error);
  Sentry.captureException(error, {
    extra: {
      vueInfo: info,
      componentName: instance?.$options?.name,
    },
  });
};

app.mount('#app');
```

### Vue Error Boundary Component

```vue
<!-- components/ErrorBoundary.vue -->
<template>
  <slot v-if="!hasError" />
  <slot v-else name="fallback" :error="error" :reset="reset" />
</template>

<script setup lang="ts">
import { ref } from 'vue';

const hasError = ref(false);
const error = ref<Error | null>(null);

function reset() {
  hasError.value = false;
  error.value = null;
}

// Capture errors from child components
onErrorCaptured((err: Error) => {
  hasError.value = true;
  error.value = err;
  return false;
});
</script>
```

```vue
<!-- Usage -->
<ErrorBoundary>
  <DataTable />
  <template #fallback="{ error, reset }">
    <p>Table failed to load: {{ error.message }}</p>
    <button @click="reset">Retry</button>
  </template>
</ErrorBoundary>
```

## Error Boundary Placement Strategy

```
App (root ErrorBoundary — catch catastrophic failures)
└── Layout
    ├── Sidebar (ErrorBoundary — sidebar crash doesn't kill main content)
    │   └── NavigationMenu
    └── Main
        ├── RevenueWidget (ErrorBoundary — one widget crash is isolated)
        │   └── Suspense → RevenueChart
        ├── UsersWidget (ErrorBoundary)
        │   └── Suspense → UserTable
        └── ActivityFeed (ErrorBoundary)
            └── Suspense → FeedList
```

**Rules:**
1. One boundary at the application root for unrecoverable errors
2. One boundary per major page section so widget failures are isolated
3. Never wrap individual buttons or small components — too granular, overhead without benefit
4. Always pair with Suspense for async components

## Anti-Patterns

```tsx
// ❌ BAD: Showing raw stack traces to users (information leakage)
<p>{error.stack}</p>

// ✅ GOOD: Show error reference ID, log details server-side
<p>Error ID: {errorRef} — contact support if this persists</p>

// ❌ BAD: Single root error boundary (entire app crashes on widget error)
<ErrorBoundary>
  <EntireApp />
</ErrorBoundary>

// ✅ GOOD: Granular boundaries per section
<ErrorBoundary><Sidebar /></ErrorBoundary>
<ErrorBoundary><MainContent /></ErrorBoundary>

// ❌ BAD: Using try/catch to catch render errors
function Component() {
  try {
    return <BuggyChild />; // try/catch cannot catch errors thrown during JSX rendering
  } catch (e) {
    return <Fallback />;   // this never runs for render errors
  }
}

// ✅ GOOD: Use class ErrorBoundary (getDerivedStateFromError)

// ❌ BAD: Treating all errors as non-retryable
// Network errors ARE retryable; null reference errors are NOT

// ✅ GOOD: Detect error type and show retry button only for network errors
const isRetryable = error.message.includes('network') || error.message.includes('fetch');

// ❌ BAD: ErrorBoundary inside Suspense
<Suspense fallback={<Loading />}>
  <ErrorBoundary fallback={<Error />}>
    <AsyncComponent />
  </ErrorBoundary>
</Suspense>

// ✅ GOOD: ErrorBoundary wraps Suspense
<ErrorBoundary fallback={<Error />}>
  <Suspense fallback={<Loading />}>
    <AsyncComponent />
  </Suspense>
</ErrorBoundary>
```
