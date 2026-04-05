# Remix + Astro Skill Guide

## Remix

### Project Layout

```
frontend/
├── package.json
├── remix.config.js
├── tsconfig.json
├── app/
│   ├── root.tsx            # Root layout + error boundary
│   ├── entry.client.tsx
│   ├── entry.server.tsx
│   ├── routes/
│   │   ├── _index.tsx      # / (home)
│   │   ├── users._index.tsx    # /users
│   │   ├── users.$id.tsx       # /users/:id
│   │   └── users.new.tsx       # /users/new
│   ├── components/
│   ├── lib/
│   │   └── db.server.ts    # .server → never sent to client
│   └── styles/
```

### Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "@remix-run/node": "<version>",
    "@remix-run/react": "<version>",
    "@remix-run/serve": "<version>",
    "react": "<version>",
    "react-dom": "<version>"
  }
}
```

### Loader (Data Fetching)

```typescript
// app/routes/users._index.tsx
import { json, type LoaderFunctionArgs } from '@remix-run/node';
import { useLoaderData, Link } from '@remix-run/react';

export async function loader({ request }: LoaderFunctionArgs) {
  const url = new URL(request.url);
  const q = url.searchParams.get('q') ?? '';

  const users = await db.users.search(q);
  return json({ users, q });
}

export default function UsersPage() {
  const { users, q } = useLoaderData<typeof loader>();

  return (
    <div>
      <h1>Users</h1>
      <ul>
        {users.map(u => (
          <li key={u.id}><Link to={`/users/${u.id}`}>{u.name}</Link></li>
        ))}
      </ul>
    </div>
  );
}
```

### Action (Mutations)

```typescript
import { redirect, type ActionFunctionArgs } from '@remix-run/node';
import { Form, useActionData } from '@remix-run/react';

export async function action({ request }: ActionFunctionArgs) {
  const formData = await request.formData();
  const name = formData.get('name')?.toString() ?? '';
  const email = formData.get('email')?.toString() ?? '';

  if (!name || !email) {
    return json({ errors: { name: !name, email: !email } }, { status: 422 });
  }

  await db.users.create({ name, email });
  return redirect('/users');
}

export default function NewUserPage() {
  const actionData = useActionData<typeof action>();

  return (
    <Form method="post">
      <input name="name" required />
      {actionData?.errors?.name && <p>Name is required</p>}
      <input name="email" type="email" required />
      {actionData?.errors?.email && <p>Email is required</p>}
      <button type="submit">Create</button>
    </Form>
  );
}
```

### useFetcher (Non-Navigating Mutations)

```typescript
import { useFetcher } from '@remix-run/react';

function LikeButton({ postId }: { postId: string }) {
  const fetcher = useFetcher();
  const isLiking = fetcher.state !== 'idle';

  return (
    <fetcher.Form method="post" action={`/posts/${postId}/like`}>
      <button disabled={isLiking}>
        {isLiking ? 'Liking...' : 'Like'}
      </button>
    </fetcher.Form>
  );
}
```

### Nested Routes with Outlet

```typescript
// app/routes/users.tsx — layout for all /users/* routes
import { Outlet, NavLink } from '@remix-run/react';

export default function UsersLayout() {
  return (
    <div className="layout">
      <nav>
        <NavLink to="/users" end>All Users</NavLink>
        <NavLink to="/users/new">New User</NavLink>
      </nav>
      <main>
        <Outlet />    {/* child route renders here */}
      </main>
    </div>
  );
}
```

### Error Boundary

```typescript
import { isRouteErrorResponse, useRouteError } from '@remix-run/react';

export function ErrorBoundary() {
  const error = useRouteError();

  if (isRouteErrorResponse(error)) {
    return <p>{error.status}: {error.data}</p>;
  }
  if (error instanceof Error) {
    return <p>Error: {error.message}</p>;
  }
  return <p>Unknown error</p>;
}
```

### Remix Key Rules

- `loader` runs on server for GET; `action` runs on server for POST/PUT/DELETE.
- `Form` provides progressive enhancement — works without JS.
- Use `useFetcher` for mutations that don't cause navigation (inline forms, toggles).
- `.server.ts` files are never bundled for the client.
- `defer()` + `Await` component for streaming slow data.
- Always `redirect()` after a successful action (PRG pattern).

---

## Astro

### Project Layout

```
frontend/
├── astro.config.mjs
├── tsconfig.json
├── package.json
├── src/
│   ├── layouts/
│   │   └── BaseLayout.astro
│   ├── pages/
│   │   ├── index.astro         # /
│   │   ├── about.astro         # /about
│   │   └── blog/
│   │       ├── index.astro     # /blog
│   │       └── [slug].astro    # /blog/:slug
│   ├── components/
│   │   ├── Header.astro
│   │   └── Counter.tsx         # React island
│   └── content/
│       ├── config.ts           # Collection schemas
│       └── blog/
│           └── *.md
```

### astro.config.mjs

```javascript
import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import tailwind from '@astrojs/tailwind';

export default defineConfig({
  integrations: [react(), tailwind()],
  output: 'hybrid',             // 'static' | 'server' | 'hybrid'
});
```

### Astro Component

```astro
---
// Frontmatter: runs at build time (or request time for SSR)
import BaseLayout from '../layouts/BaseLayout.astro';
import { getCollection } from 'astro:content';
import Counter from '../components/Counter.tsx';

const posts = await getCollection('blog');
const { title = 'Home' } = Astro.props;
---

<BaseLayout title={title}>
  <h1>Blog Posts</h1>
  {posts.map(post => (
    <a href={`/blog/${post.slug}`}>{post.data.title}</a>
  ))}

  <!-- React island: loads React + hydrates on visible -->
  <Counter client:visible initialCount={0} />
</BaseLayout>
```

### Island Directives

| Directive | When JS loads |
|-----------|--------------|
| `client:load` | Immediately on page load |
| `client:idle` | When browser is idle (requestIdleCallback) |
| `client:visible` | When element enters viewport |
| `client:media="(max-width: 768px)"` | When media query matches |
| `client:only="react"` | Client-only, skip SSR (use sparingly) |

### Content Collections

```typescript
// src/content/config.ts
import { defineCollection, z } from 'astro:content';

const blog = defineCollection({
  type: 'content',
  schema: z.object({
    title: z.string(),
    date: z.coerce.date(),
    tags: z.array(z.string()).default([]),
    draft: z.boolean().default(false),
  }),
});

export const collections = { blog };
```

```astro
---
// src/pages/blog/[slug].astro
import { getCollection, getEntry } from 'astro:content';
import type { GetStaticPaths } from 'astro';

export const getStaticPaths: GetStaticPaths = async () => {
  const posts = await getCollection('blog', p => !p.data.draft);
  return posts.map(post => ({ params: { slug: post.slug }, props: { post } }));
};

const { post } = Astro.props;
const { Content } = await post.render();
---

<article>
  <h1>{post.data.title}</h1>
  <Content />
</article>
```

### Hybrid Rendering

```astro
---
// Mark a page as server-rendered (in 'hybrid' output mode)
export const prerender = false;

// Access request data
const url = new URL(Astro.request.url);
const id = Astro.params.id;
const user = await db.users.findById(id);
if (!user) return Astro.redirect('/404');
---
<h1>{user.name}</h1>
```

### Astro Key Rules

- Astro components have zero JS by default — add `client:*` only for interactive islands.
- Use `client:visible` for below-fold components, `client:load` for above-fold critical UI.
- Content Collections with Zod schemas provide type-safe frontmatter.
- `output: 'hybrid'` lets you opt individual pages into SSR with `export const prerender = false`.
- Always filter draft posts in `getCollection` predicates, not after fetching.
- Use `.astro` for layouts and static content; `.tsx/.vue/.svelte` for interactive islands.
