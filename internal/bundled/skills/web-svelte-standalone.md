# Svelte Standalone Skill Guide

## Project Layout

```
frontend/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── index.html
├── src/
│   ├── main.ts             # Entry: mount App
│   ├── App.svelte          # Root component
│   ├── components/
│   ├── stores/             # Writable/readable stores
│   └── lib/
│       └── api.ts
```

## Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "svelte": "<version>"
  },
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "<version>",
    "typescript": "<version>",
    "vite": "<version>"
  }
}
```

## Entry Point

```typescript
// src/main.ts
import { mount } from 'svelte';
import App from './App.svelte';

const app = mount(App, { target: document.getElementById('app')! });
export default app;
```

## Stores

```typescript
// src/stores/counter.ts
import { writable, readable, derived, get } from 'svelte/store';

// Writable store
export const count = writable(0);
export const name = writable('');

// Read-only store (e.g., clock)
export const time = readable(new Date(), (set) => {
  const interval = setInterval(() => set(new Date()), 1000);
  return () => clearInterval(interval);   // cleanup
});

// Derived store
export const greeting = derived(name, ($name) => `Hello, ${$name || 'World'}!`);

// Combined derived
export const summary = derived([count, name], ([$count, $name]) => ({
  label: $name,
  value: $count,
}));

// Read current value imperatively
const current = get(count);
```

## Custom Store Factory

```typescript
// src/stores/createList.ts
import { writable } from 'svelte/store';

export function createList<T extends { id: string }>(initial: T[] = []) {
  const { subscribe, update, set } = writable<T[]>(initial);

  return {
    subscribe,
    add: (item: T) => update(list => [...list, item]),
    remove: (id: string) => update(list => list.filter(i => i.id !== id)),
    reset: () => set(initial),
  };
}

// Usage
export const tasks = createList<Task>();
tasks.add({ id: '1', name: 'First' });
tasks.remove('1');
```

## $-Prefix Auto-Subscription

```svelte
<script lang="ts">
  import { count, greeting, tasks } from './stores';
  // $count auto-subscribes and unsubscribes on destroy
</script>

<p>{$greeting}</p>
<p>Count: {$count}</p>
<button on:click={() => count.update(n => n + 1)}>+</button>

{#each $tasks as task (task.id)}
  <p>{task.name}</p>
{/each}
```

## Event Modifiers

```svelte
<!-- Prevent default -->
<a href="/foo" on:click|preventDefault={() => handleClick()}>Link</a>

<!-- Stop propagation -->
<div on:click|stopPropagation={() => {}}>Inner</div>

<!-- Once: handler fires only once -->
<button on:click|once={() => init()}>Init</button>

<!-- Passive: improves scroll performance -->
<div on:touchstart|passive={handleTouch}>...</div>

<!-- Chain modifiers -->
<form on:submit|preventDefault|stopPropagation={handleSubmit}>...</form>
```

## Named Slots

```svelte
<!-- Modal.svelte -->
<div class="modal-overlay" on:click|self={close}>
  <div class="modal">
    <header><slot name="title">Default Title</slot></header>
    <div class="body"><slot /></div>
    <footer><slot name="actions" /></footer>
  </div>
</div>

<!-- Usage -->
<Modal {close}>
  <svelte:fragment slot="title">Confirm Delete</svelte:fragment>
  <p>Are you sure?</p>
  <svelte:fragment slot="actions">
    <button on:click={close}>Cancel</button>
    <button on:click={confirm}>Delete</button>
  </svelte:fragment>
</Modal>
```

## Transitions

```svelte
<script lang="ts">
  import { fade, fly, slide, scale } from 'svelte/transition';
  import { quintOut } from 'svelte/easing';
  let visible = true;
</script>

{#if visible}
  <div transition:fade={{ duration: 200 }}>Fades in/out</div>
{/if}

{#if visible}
  <div in:fly={{ y: 20, duration: 300 }} out:fade>Flies in, fades out</div>
{/if}

{#each items as item (item.id)}
  <div animate:flip={{ duration: 200 }}>{item.name}</div>
{/each}
```

## createEventDispatcher (Child → Parent)

```svelte
<!-- ItemCard.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let item: { id: string; name: string };

  const dispatch = createEventDispatcher<{
    select: { id: string };
    delete: { id: string };
  }>();
</script>

<div>
  <span>{item.name}</span>
  <button on:click={() => dispatch('select', { id: item.id })}>Select</button>
  <button on:click={() => dispatch('delete', { id: item.id })}>Delete</button>
</div>
```

```svelte
<!-- Parent.svelte -->
<ItemCard
  {item}
  on:select={e => handleSelect(e.detail.id)}
  on:delete={e => handleDelete(e.detail.id)}
/>
```

## Component Lifecycle

```svelte
<script lang="ts">
  import { onMount, onDestroy, beforeUpdate, afterUpdate } from 'svelte';

  onMount(() => {
    // DOM is ready
    return () => { /* cleanup */ };
  });

  onDestroy(() => { /* final cleanup */ });
</script>
```

## Key Rules

- Use `$store` syntax in templates — it auto-subscribes and prevents memory leaks.
- Read store value imperatively with `get(store)` only outside components.
- Custom store factories encapsulate all mutation logic — consumers use the API.
- Always return a cleanup function from `readable` if it sets up intervals/listeners.
- Use event modifiers instead of calling `e.preventDefault()` manually.
- `createEventDispatcher` generics document the event payload types.
- Transitions on `{#each}` blocks need `animate:flip` for smooth reorders.
