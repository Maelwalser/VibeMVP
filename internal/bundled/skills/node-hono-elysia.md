# Node.js + Hono / Elysia Skill Guide

---

## Hono

### Project Layout

```
service-name/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts          # Entry point + adapter binding
│   ├── app.ts            # Hono instance
│   ├── routes/           # Route modules
│   └── middleware/        # Custom middleware
└── wrangler.toml         # (Cloudflare Workers only)
```

### package.json Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "hono": "<version>"
  },
  "devDependencies": {
    "@types/node": "<version>",
    "typescript": "<version>",
    "@hono/node-server": "<version>"
  }
}
```

### App Setup

```typescript
// src/app.ts
import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { logger } from 'hono/logger';
import { prettyJSON } from 'hono/pretty-json';
import { userRoutes } from './routes/users';

export const app = new Hono();

// Global middleware
app.use('*', logger());
app.use('*', cors({ origin: process.env.ALLOWED_ORIGIN || '*' }));
app.use('*', prettyJSON());

// Health check
app.get('/health', (c) => c.json({ status: 'ok' }));

// Mount sub-routes
app.route('/api/users', userRoutes);

// Global error handler
app.onError((err, c) => {
  console.error(err);
  return c.json({ error: err.message || 'Internal server error' }, 500);
});

app.notFound((c) => c.json({ error: 'Not found' }, 404));
```

### Running on Different Runtimes

```typescript
// Node.js — src/index.ts
import { serve } from '@hono/node-server';
import { app } from './app';

serve({ fetch: app.fetch, port: Number(process.env.PORT) || 8080 }, (info) => {
  console.log(`Listening on :${info.port}`);
});

// Cloudflare Workers — src/index.ts
import { app } from './app';
export default app;  // Workers calls app.fetch automatically

// Bun — src/index.ts
import { app } from './app';
export default { port: 8080, fetch: app.fetch };
```

### Route / Handler Pattern

```typescript
// src/routes/users.ts
import { Hono } from 'hono';
import { UserService } from '../services/userService';

const svc = new UserService();
export const userRoutes = new Hono();

userRoutes.get('/', async (c) => {
  const users = await svc.findAll();
  return c.json({ data: users });
});

userRoutes.get('/:id', async (c) => {
  const user = await svc.findById(c.req.param('id'));
  if (!user) return c.json({ error: 'Not found' }, 404);
  return c.json({ data: user });
});

userRoutes.post('/', async (c) => {
  const body = await c.req.json();
  const user = await svc.create(body);
  return c.json({ data: user }, 201);
});

userRoutes.delete('/:id', async (c) => {
  await svc.delete(c.req.param('id'));
  return c.body(null, 204);
});
```

### Middleware

```typescript
// Auth middleware
import { MiddlewareHandler } from 'hono';
import { verifyToken } from '../services/auth';

export const requireAuth: MiddlewareHandler = async (c, next) => {
  const token = c.req.header('Authorization')?.replace('Bearer ', '');
  if (!token) return c.json({ error: 'Unauthorized' }, 401);
  try {
    c.set('user', verifyToken(token));
    await next();
  } catch {
    return c.json({ error: 'Invalid token' }, 401);
  }
};

// Apply to specific routes
userRoutes.use('/admin/*', requireAuth);
// Apply globally
app.use('*', requireAuth);
```

---

## Elysia (Bun-native)

### Project Layout

```
service-name/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts          # Entry point
│   ├── plugins/          # Elysia plugin modules
│   └── routes/           # Route handlers
└── bun.lockb
```

### package.json Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "elysia": "<version>"
  },
  "devDependencies": {
    "bun-types": "<version>"
  }
}
```

### App Setup

```typescript
// src/index.ts
import { Elysia } from 'elysia';
import { userPlugin } from './plugins/users';

const app = new Elysia()
  .use(userPlugin)
  .get('/health', () => ({ status: 'ok' }))
  .onError(({ code, error, set }) => {
    if (code === 'NOT_FOUND') {
      set.status = 404;
      return { error: 'Not found' };
    }
    set.status = 500;
    console.error(error);
    return { error: 'Internal server error' };
  })
  .listen(process.env.PORT || 8080);

console.log(`Elysia running on :${app.server?.port}`);
```

### Plugin with Typed Schema (t.Object)

```typescript
// src/plugins/users.ts
import { Elysia, t } from 'elysia';
import { UserService } from '../services/userService';

const svc = new UserService();

const CreateUserBody = t.Object({
  name:  t.String({ minLength: 1 }),
  email: t.String({ format: 'email' }),
});

const UserResponse = t.Object({
  id:    t.String(),
  name:  t.String(),
  email: t.String(),
});

export const userPlugin = new Elysia({ prefix: '/api/users' })
  .get('/', () => svc.findAll(), {
    response: t.Array(UserResponse),
  })
  .get('/:id', ({ params: { id }, error }) =>
    svc.findById(id).then((u) => u ?? error(404, 'Not found')),
    { response: UserResponse }
  )
  .post('/', ({ body }) => svc.create(body), {
    body: CreateUserBody,
    response: { 201: UserResponse },
  })
  .delete('/:id', ({ params: { id }, set }) => {
    set.status = 204;
    return svc.delete(id);
  });
```

### Guards / Middleware with .use() and derive

```typescript
import { Elysia } from 'elysia';
import { verifyToken } from '../services/auth';

// Shared auth plugin — derive adds `user` to every context in this scope
export const authPlugin = new Elysia({ name: 'auth' })
  .derive(({ headers, error }) => {
    const token = headers['authorization']?.replace('Bearer ', '');
    if (!token) throw error(401, 'Missing token');
    try {
      return { user: verifyToken(token) };
    } catch {
      throw error(401, 'Invalid token');
    }
  });

// Apply to a route group
const protectedRoutes = new Elysia()
  .use(authPlugin)
  .get('/profile', ({ user }) => user);
```

## Key Rules — Hono

- `c.json()` / `c.text()` / `c.body()` are the only correct ways to return responses — never return plain objects.
- Middleware calls `await next()` to pass control to the next handler; skipping it short-circuits the chain.
- For Cloudflare Workers, export `export default app` (not `app.fetch`) — Workers resolves `.fetch` automatically.
- Use `app.route()` to compose route groups; keep each router in its own file.

## Key Rules — Elysia

- Elysia runs on Bun — do not use Node.js-only APIs (`fs`, `http`); use Bun's built-ins instead.
- `t.Object()` schemas provide both runtime validation and TypeScript inference — always define body/response schemas.
- Name plugins with `{ name: 'plugin-name' }` when they declare decorators or derived properties so Elysia deduplicates them.
- Use `derive` to attach request-scoped computed values (e.g., authenticated user) to the context.
- Errors thrown inside handlers propagate to `onError`; use the `error()` helper for typed HTTP errors.
