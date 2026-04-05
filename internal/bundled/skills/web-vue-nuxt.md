# Vue 3 + Nuxt 3 Skill Guide

## Project Layout (Nuxt 3)

```
frontend/
├── nuxt.config.ts
├── tsconfig.json
├── package.json
├── app.vue                 # Root component
├── pages/                  # File-based routing
│   ├── index.vue
│   ├── users/
│   │   ├── index.vue       # /users
│   │   └── [id].vue        # /users/:id
├── components/             # Auto-imported components
├── composables/            # Auto-imported composables
├── stores/                 # Pinia stores
├── server/
│   └── api/                # Nitro server routes
├── middleware/              # Route middleware
└── plugins/                # Nuxt plugins
```

## Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "nuxt": "<version>",
    "@pinia/nuxt": "<version>",
    "pinia": "<version>"
  },
  "devDependencies": {
    "typescript": "<version>",
    "@nuxt/devtools": "<version>"
  }
}
```

## nuxt.config.ts

```typescript
export default defineNuxtConfig({
  modules: ['@pinia/nuxt'],
  runtimeConfig: {
    apiSecret: '',                    // server-only
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE ?? 'http://localhost:8080',
    },
  },
  typescript: { strict: true },
});
```

## Vue 3 Composition API

```vue
<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue';

// Primitive reactivity
const count = ref(0);
const name = ref('');

// Object reactivity
const user = reactive({ id: '', email: '' });

// Derived value
const displayName = computed(() => name.value.trim() || 'Anonymous');

// Watcher
watch(count, (newVal, oldVal) => {
  console.log(`count changed: ${oldVal} → ${newVal}`);
});

// Watch multiple sources
watch([count, name], ([c, n]) => {
  console.log(c, n);
});

function increment() {
  count.value++;
}
</script>

<template>
  <button @click="increment">{{ count }}</button>
  <p>{{ displayName }}</p>
</template>
```

## Pinia Store

```typescript
// stores/useUserStore.ts
import { defineStore } from 'pinia';

interface User { id: string; email: string; name: string; }

export const useUserStore = defineStore('user', {
  state: () => ({
    current: null as User | null,
    list: [] as User[],
    loading: false,
  }),
  getters: {
    isAuthenticated: (state) => state.current !== null,
    userById: (state) => (id: string) => state.list.find(u => u.id === id),
  },
  actions: {
    async fetchUsers() {
      this.loading = true;
      try {
        this.list = await $fetch('/api/users');
      } finally {
        this.loading = false;
      }
    },
    setCurrentUser(user: User | null) {
      this.current = user;
    },
  },
});

// Usage in component
const store = useUserStore();
await store.fetchUsers();
console.log(store.isAuthenticated);
```

## Vue Router (standalone)

```typescript
import { useRouter, useRoute } from 'vue-router';

const router = useRouter();
const route = useRoute();

// Params: route.params.id
// Query:  route.query.search
// Navigate
router.push('/users/123');
router.push({ name: 'user-detail', params: { id: '123' } });
router.replace('/login');
router.back();
```

## Nuxt 3 File-based Routing

```
pages/index.vue           → /
pages/users/index.vue     → /users
pages/users/[id].vue      → /users/:id
pages/[...slug].vue       → catch-all
```

```vue
<!-- pages/users/[id].vue -->
<script setup lang="ts">
const route = useRoute();
const id = computed(() => route.params.id as string);
</script>
```

## Data Fetching (SSR)

```vue
<script setup lang="ts">
// SSR-aware: runs on server + client
const { data: users, pending, error } = await useAsyncData(
  'users',
  () => $fetch('/api/users')
);

// Simple fetch with automatic key
const { data: user } = await useFetch(`/api/users/${id.value}`, {
  watch: [id],   // re-fetch when id changes
});
</script>
```

## Composables (Auto-imported)

```typescript
// composables/useCounter.ts
export function useCounter(initial = 0) {
  const count = ref(initial);
  const increment = () => count.value++;
  const reset = () => { count.value = initial; };
  return { count: readonly(count), increment, reset };
}

// Usage in any component — no import needed
const { count, increment } = useCounter(10);
```

## definePageMeta

```vue
<script setup lang="ts">
definePageMeta({
  layout: 'dashboard',
  middleware: ['auth'],
  title: 'Dashboard',
});
</script>
```

## Route Middleware

```typescript
// middleware/auth.ts
export default defineNuxtRouteMiddleware((to) => {
  const store = useUserStore();
  if (!store.isAuthenticated) {
    return navigateTo('/login');
  }
});
```

## Key Rules

- Use `<script setup lang="ts">` — it is the modern standard.
- Prefer `ref` for primitives, `reactive` for objects/collections.
- Store all global state in Pinia; local UI state stays in the component.
- `useAsyncData` keys must be unique per page to avoid SSR hydration conflicts.
- Composables must start with `use` to be auto-imported by Nuxt.
- Never mutate Pinia state directly outside actions.
- Use `readonly()` when exposing ref values from composables.
