# SvelteKit Skill Guide

## Project Layout

```
frontend/
├── svelte.config.js
├── vite.config.ts
├── package.json
├── src/
│   ├── app.html            # HTML shell
│   ├── app.css
│   ├── lib/                # $lib alias
│   │   ├── components/
│   │   ├── stores/
│   │   └── api.ts
│   └── routes/
│       ├── +layout.svelte  # Root layout
│       ├── +layout.ts      # Root layout load
│       ├── +page.svelte    # Home page
│       ├── +page.ts        # Home load function
│       ├── users/
│       │   ├── +page.svelte
│       │   ├── +page.ts
│       │   └── [id]/
│       │       ├── +page.svelte
│       │       └── +page.ts
│       └── api/            # Server routes
│           └── users/
│               └── +server.ts
```

## Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "@sveltejs/kit": "<version>",
    "svelte": "<version>"
  },
  "devDependencies": {
    "@sveltejs/adapter-auto": "<version>",
    "typescript": "<version>",
    "vite": "<version>"
  }
}
```

## File Routing

```
src/routes/+page.svelte               → /
src/routes/users/+page.svelte         → /users
src/routes/users/[id]/+page.svelte    → /users/:id
src/routes/[...rest]/+page.svelte     → catch-all
src/routes/(auth)/login/+page.svelte  → /login  (route group, no path segment)
```

## Load Function (+page.ts)

```typescript
// src/routes/users/+page.ts
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params, url }) => {
  const res = await fetch('/api/users');
  if (!res.ok) throw new Error('Failed to load users');
  const users = await res.json();
  return { users };
};
```

```svelte
<!-- src/routes/users/+page.svelte -->
<script lang="ts">
  import type { PageData } from './$types';
  export let data: PageData;   // typed from load return
</script>

{#each data.users as user (user.id)}
  <div>{user.name}</div>
{/each}
```

## Server Actions (+page.server.ts)

```typescript
// src/routes/users/new/+page.server.ts
import type { Actions } from './$types';
import { fail, redirect } from '@sveltejs/kit';

export const actions: Actions = {
  create: async ({ request, fetch }) => {
    const formData = await request.formData();
    const name = formData.get('name')?.toString() ?? '';
    if (!name) return fail(400, { name, error: 'Name is required' });

    const res = await fetch('/api/users', {
      method: 'POST',
      body: JSON.stringify({ name }),
      headers: { 'Content-Type': 'application/json' },
    });
    if (!res.ok) return fail(500, { name, error: 'Server error' });
    throw redirect(303, '/users');
  },
};
```

```svelte
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { ActionData } from './$types';
  export let form: ActionData;
</script>

<form method="POST" action="?/create" use:enhance>
  <input name="name" value={form?.name ?? ''} />
  {#if form?.error}<p class="error">{form.error}</p>{/if}
  <button>Create</button>
</form>
```

## $page Store

```svelte
<script lang="ts">
  import { page } from '$app/stores';

  // $page.url, $page.params, $page.data, $page.error
  $: currentPath = $page.url.pathname;
  $: userId = $page.params.id;
</script>
```

## Navigation

```typescript
import { goto, invalidate, invalidateAll } from '$app/navigation';

// Navigate
await goto('/dashboard');
await goto('/users', { replaceState: true });

// Re-run load functions
await invalidate('/api/users');  // re-run loads that fetch this URL
await invalidateAll();           // re-run all load functions
```

## Svelte Component Patterns

```svelte
<script lang="ts">
  // $: reactive declarations
  export let items: string[] = [];
  $: filtered = items.filter(i => i.length > 2);
  $: count = filtered.length;

  // Two-way binding
  let inputValue = '';

  // Lifecycle
  import { onMount, onDestroy } from 'svelte';
  onMount(() => {
    console.log('mounted');
    return () => console.log('cleanup');  // optional cleanup
  });
</script>

<!-- bind:value two-way binding -->
<input bind:value={inputValue} />

<!-- {#each} with key -->
{#each filtered as item (item)}
  <p>{item}</p>
{/each}

<!-- {#if} block -->
{#if count > 0}
  <p>{count} items</p>
{:else}
  <p>No items</p>
{/if}

<!-- Event handling -->
<button on:click={() => items = [...items, inputValue]}>Add</button>
```

## Slot Patterns

```svelte
<!-- Card.svelte -->
<div class="card">
  <slot name="header" />
  <div class="body"><slot /></div>
  <slot name="footer" />
</div>

<!-- Usage -->
<Card>
  <svelte:fragment slot="header"><h2>Title</h2></svelte:fragment>
  <p>Body content</p>
  <svelte:fragment slot="footer"><button>OK</button></svelte:fragment>
</Card>
```

## Server Route (+server.ts)

```typescript
// src/routes/api/users/+server.ts
import type { RequestHandler } from './$types';
import { json, error } from '@sveltejs/kit';

export const GET: RequestHandler = async ({ url }) => {
  const users = await db.findAll();
  return json(users);
};

export const POST: RequestHandler = async ({ request }) => {
  const body = await request.json();
  if (!body.name) throw error(400, 'Name required');
  const user = await db.create(body);
  return json(user, { status: 201 });
};
```

## Key Rules

- Use `+page.ts` for universal (SSR + CSR) loads, `+page.server.ts` for server-only (DB, secrets).
- Use `use:enhance` on forms for progressive enhancement (no JS required fallback).
- `$lib` alias points to `src/lib/` — always import shared code from there.
- Reactive declarations (`$:`) re-run any time their dependencies change.
- Use `(group)` folders for layout groups that don't affect the URL.
- Always return data from `load` — never set module-level state in load functions.
- Prefer `invalidate()` over `goto()` to refresh data without full navigation.
