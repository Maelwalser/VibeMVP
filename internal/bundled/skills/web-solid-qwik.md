# Solid.js + Qwik Skill Guide

## Solid.js

### Project Layout

```
frontend/
├── package.json
├── vite.config.ts
├── src/
│   ├── index.tsx           # Entry point
│   ├── App.tsx             # Root + router
│   ├── routes/
│   ├── components/
│   ├── stores/
│   └── lib/api.ts
```

### Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "solid-js": "<version>",
    "@solidjs/router": "<version>"
  },
  "devDependencies": {
    "vite-plugin-solid": "<version>",
    "typescript": "<version>",
    "vite": "<version>"
  }
}
```

### Fine-grained Reactivity

```typescript
import { createSignal, createMemo, createEffect, batch } from 'solid-js';

// createSignal — reactive primitive
const [count, setCount] = createSignal(0);
const [name, setName] = createSignal('');

// Read: count()   Write: setCount(n => n + 1)

// createMemo — derived value (only recomputes when deps change)
const doubled = createMemo(() => count() * 2);
const greeting = createMemo(() => `Hello, ${name() || 'World'}`);

// createEffect — side effect (runs when deps change)
createEffect(() => {
  console.log('count is now', count());
  // No cleanup return needed — Solid uses ownership system
});

// batch multiple updates (single re-render)
batch(() => {
  setCount(10);
  setName('Alice');
});
```

### createStore (Nested State)

```typescript
import { createStore, produce } from 'solid-js/store';

interface State { users: User[]; loading: boolean; error: string | null; }

const [state, setState] = createStore<State>({
  users: [],
  loading: false,
  error: null,
});

// Granular path update
setState('loading', true);
setState('users', users => [...users, newUser]);

// Immer-style with produce
setState(produce(s => {
  s.users.push(newUser);
  s.loading = false;
}));
```

### createResource (Async Data)

```typescript
import { createResource, Suspense, ErrorBoundary } from 'solid-js';

function UserList() {
  const [users] = createResource(fetchUsers);
  return (
    <ErrorBoundary fallback={err => <p>Error: {err.message}</p>}>
      <Suspense fallback={<p>Loading...</p>}>
        <For each={users()} fallback={<p>No users</p>}>
          {user => <UserRow user={user} />}
        </For>
      </Suspense>
    </ErrorBoundary>
  );
}

// With reactive source (refetches when id changes)
function UserDetail(props: { id: string }) {
  const [user, { refetch }] = createResource(
    () => props.id,
    id => fetchUser(id)
  );
  return <div>{user()?.name}</div>;
}
```

### Show / For Components

```typescript
import { Show, For, Switch, Match } from 'solid-js';

// Show (conditional)
<Show when={user()} fallback={<p>Loading...</p>}>
  {user => <UserCard user={user()} />}
</Show>

// For (list with keying)
<For each={items()} fallback={<p>Empty</p>}>
  {(item, index) => <ItemRow item={item} index={index()} />}
</For>

// Switch/Match
<Switch fallback={<NotFound />}>
  <Match when={page() === 'home'}><Home /></Match>
  <Match when={page() === 'about'}><About /></Match>
</Switch>
```

### Lazy / Code Splitting

```typescript
import { lazy, Suspense } from 'solid-js';
import { Route } from '@solidjs/router';

const Dashboard = lazy(() => import('./routes/Dashboard'));

<Route path="/dashboard" component={() => (
  <Suspense fallback={<Spinner />}>
    <Dashboard />
  </Suspense>
)} />
```

### SolidJS Key Rules

- `count()` reads signal; `setCount(v)` writes — never destructure to lose reactivity.
- `createMemo` wraps expensive computations; do NOT compute inside JSX directly.
- Reactive data in JSX must be accessed as function calls: `{count()}` not `{count}`.
- Use `For` instead of `.map()` — it avoids full list re-renders.
- `createStore` granular updates are more efficient than `createSignal` for objects.

---

## Qwik

### Project Layout

```
frontend/
├── package.json
├── vite.config.ts
├── src/
│   ├── entry.ssr.tsx       # SSR entry
│   ├── root.tsx            # Root component
│   └── routes/
│       ├── layout.tsx      # Root layout
│       ├── index.tsx       # Home
│       └── users/
│           ├── index.tsx
│           └── [id]/
│               └── index.tsx
```

### Key Dependencies

```json
{
  "dependencies": {
    "@builder.io/qwik": "^1.10.0",
    "@builder.io/qwik-city": "^1.10.0"
  }
}
```

### Resumability Concept

Qwik serializes component state + event handlers into HTML. On the client, no JavaScript runs until interaction. Handlers load lazily via QRL (Qwik Resource Locator) on first use — this is "resumability" (vs hydration).

### Signals and Stores

```typescript
import { component$, useSignal, useStore, $ } from '@builder.io/qwik';

export const Counter = component$(() => {
  const count = useSignal(0);
  const user = useStore({ name: '', email: '' });

  // $ suffix = QRL — handler is lazy loaded
  const increment = $(() => { count.value++; });

  return (
    <div>
      <p>{count.value}</p>
      <button onClick$={increment}>+</button>
      <input bind:value={user.name} />
    </div>
  );
});
```

### onClick$ / onChange$ Handlers

```typescript
export const Form = component$(() => {
  const value = useSignal('');

  return (
    <div>
      {/* Inline $ handler */}
      <button onClick$={() => console.log('clicked')}>Click</button>

      {/* Change handler */}
      <input
        value={value.value}
        onChange$={(e, el) => { value.value = el.value; }}
      />
    </div>
  );
});
```

### server$ and routeLoader$

```typescript
// src/routes/users/index.tsx
import { component$ } from '@builder.io/qwik';
import { routeLoader$, routeAction$, Form } from '@builder.io/qwik-city';

// Runs on server, result serialized to client
export const useUsers = routeLoader$(async ({ env }) => {
  const users = await db.findAll();
  return users;
});

export const useCreateUser = routeAction$(async (data, { redirect }) => {
  await db.create(data);
  throw redirect(302, '/users');
});

export default component$(() => {
  const users = useUsers();
  const createUser = useCreateUser();

  return (
    <div>
      <ul>
        {users.value.map(u => <li key={u.id}>{u.name}</li>)}
      </ul>
      <Form action={createUser}>
        <input name="name" required />
        <button type="submit">Add</button>
      </Form>
    </div>
  );
});
```

### Qwik Key Rules

- Every exported function that crosses server/client boundary must be wrapped in `$()`.
- `component$`, `useTask$`, `onClick$` — the `$` suffix is mandatory, not optional.
- `useSignal` for primitives, `useStore` for objects — both are serializable.
- Avoid importing large libraries at the top level — Qwik lazy-loads via QRL.
- `routeLoader$` is the canonical way to fetch data for a route on the server.
- `server$` wraps arbitrary server functions called from the client.
