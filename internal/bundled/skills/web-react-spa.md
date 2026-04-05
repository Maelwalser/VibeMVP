# React SPA Skill Guide

## Project Layout

```
frontend/
├── package.json
├── tsconfig.json
├── vite.config.ts
├── index.html
├── src/
│   ├── main.tsx            # Entry point
│   ├── App.tsx             # Root component + router setup
│   ├── routes/             # Route-level components
│   ├── components/
│   │   ├── ui/             # Reusable primitives
│   │   └── features/       # Feature-specific components
│   ├── hooks/              # Custom hooks
│   ├── stores/             # State (Zustand / Context)
│   ├── lib/
│   │   ├── api.ts          # API client
│   │   └── utils.ts
│   └── types/
│       └── api.ts
```

## Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "react": "<version>",
    "react-dom": "<version>",
    "react-router-dom": "<version>"
  },
  "devDependencies": {
    "@types/react": "<version>",
    "@types/react-dom": "<version>",
    "@vitejs/plugin-react": "<version>",
    "typescript": "<version>",
    "vite": "<version>"
  }
}
```

## Router Setup

```typescript
// src/App.tsx
import { BrowserRouter, Routes, Route, Outlet } from 'react-router-dom';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<HomePage />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="users/:id" element={<UserDetailPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
        <Route element={<ProtectedRoute />}>
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function Layout() {
  return (
    <>
      <NavBar />
      <main><Outlet /></main>
    </>
  );
}
```

## Protected Route

```typescript
// src/components/ProtectedRoute.tsx
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';

export function ProtectedRoute() {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) return <div>Loading...</div>;
  if (!user) return <Navigate to="/login" state={{ from: location }} replace />;
  return <Outlet />;
}
```

## Navigation Hooks

```typescript
import { useParams, useNavigate, useLocation } from 'react-router-dom';

function UserDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();

  const goBack = () => navigate(-1);
  const goToUser = (userId: string) => navigate(`/users/${userId}`);
  const goWithState = () => navigate('/dashboard', { state: { from: 'detail' } });

  return <div>User {id}</div>;
}
```

## Code Splitting

```typescript
// src/App.tsx
import { lazy, Suspense } from 'react';
import { Routes, Route } from 'react-router-dom';

const DashboardPage = lazy(() => import('./routes/DashboardPage'));
const SettingsPage = lazy(() => import('./routes/SettingsPage'));

function App() {
  return (
    <Suspense fallback={<PageSpinner />}>
      <Routes>
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Routes>
    </Suspense>
  );
}
```

## useEffect Cleanup

```typescript
function DataComponent({ userId }: { userId: string }) {
  const [data, setData] = useState<User | null>(null);

  useEffect(() => {
    let cancelled = false;
    const controller = new AbortController();

    async function fetch() {
      try {
        const res = await getUser(userId, { signal: controller.signal });
        if (!cancelled) setData(res);
      } catch (err) {
        if (!cancelled) console.error(err);
      }
    }

    fetch();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [userId]);

  return data ? <UserCard user={data} /> : null;
}
```

## Performance: useMemo / useCallback / React.memo

```typescript
// Memoize expensive computation
const sorted = useMemo(
  () => [...items].sort((a, b) => a.name.localeCompare(b.name)),
  [items]
);

// Stable callback reference
const handleDelete = useCallback((id: string) => {
  setItems(prev => prev.filter(i => i.id !== id));
}, []);

// Skip re-render when props unchanged
const ItemRow = React.memo(function ItemRow({ item, onDelete }: Props) {
  return <li>{item.name} <button onClick={() => onDelete(item.id)}>x</button></li>;
});
```

## Stable Keys in Lists

```typescript
// ALWAYS use stable unique IDs as keys, never array index
items.map(item => <ItemRow key={item.id} item={item} onDelete={handleDelete} />)
```

## Key Rules

- Functional components with hooks only — no class components.
- Co-locate state as close as possible to where it is used.
- Extract repeated logic into custom hooks in `hooks/`.
- Always clean up effects (abort controllers, event listeners, timers).
- Use `React.memo` + `useCallback` only when profiling shows a real issue.
- Use stable keys (IDs) in lists — never array index when list can reorder.
- Lazy-load heavy routes with `React.lazy` + `Suspense`.
- All environment variables via `import.meta.env.VITE_*` (Vite) or `process.env.REACT_APP_*` (CRA).
