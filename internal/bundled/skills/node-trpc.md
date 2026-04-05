# Node.js + tRPC Skill Guide

## Project Layout

```
service-name/
├── package.json
├── tsconfig.json
├── src/
│   ├── server/
│   │   ├── index.ts          # HTTP server entry point
│   │   ├── trpc.ts           # tRPC instance + base procedures
│   │   ├── context.ts        # Context factory
│   │   └── routers/
│   │       ├── index.ts      # Root router (merges all sub-routers)
│   │       └── users.ts      # Users sub-router
│   └── client/
│       └── trpc.ts           # Type-safe client
└── shared/
    └── types.ts              # Shared types (optional)
```

## package.json Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "@trpc/server": "<version>",
    "zod": "<version>",
    "express": "<version>",
    "@trpc/client": "<version>"
  },
  "devDependencies": {
    "@types/express": "<version>",
    "@types/node": "<version>",
    "typescript": "<version>"
  }
}
```

## tRPC Instance + Base Procedures

```typescript
// src/server/trpc.ts
import { initTRPC, TRPCError } from '@trpc/server';
import type { Context } from './context';

const t = initTRPC.context<Context>().create();

// Exportable building blocks
export const router      = t.router;
export const publicProcedure   = t.procedure;

// Protected procedure — reusable middleware that checks auth
export const protectedProcedure = t.procedure.use(({ ctx, next }) => {
  if (!ctx.user) {
    throw new TRPCError({ code: 'UNAUTHORIZED', message: 'Not authenticated' });
  }
  return next({ ctx: { ...ctx, user: ctx.user } });
});
```

## Context

```typescript
// src/server/context.ts
import { CreateExpressContextOptions } from '@trpc/server/adapters/express';
import { verifyToken } from '../services/auth';

export interface Context {
  user: { id: string; email: string } | null;
}

export function createContext({ req }: CreateExpressContextOptions): Context {
  const token = req.headers.authorization?.replace('Bearer ', '');
  if (!token) return { user: null };
  try {
    return { user: verifyToken(token) };
  } catch {
    return { user: null };
  }
}
```

## Routers

```typescript
// src/server/routers/users.ts
import { z } from 'zod';
import { router, publicProcedure, protectedProcedure } from '../trpc';
import { UserService } from '../../services/userService';

const svc = new UserService();

export const usersRouter = router({
  // Query: returns a list
  list: publicProcedure.query(async () => {
    return svc.findAll();
  }),

  // Query with input validation
  byId: publicProcedure
    .input(z.object({ id: z.string().uuid() }))
    .query(async ({ input }) => {
      const user = await svc.findById(input.id);
      if (!user) throw new TRPCError({ code: 'NOT_FOUND', message: 'User not found' });
      return user;
    }),

  // Mutation: create
  create: protectedProcedure
    .input(z.object({
      name:  z.string().min(1),
      email: z.string().email(),
    }))
    .mutation(async ({ input }) => {
      return svc.create(input);
    }),

  // Mutation: update (partial input)
  update: protectedProcedure
    .input(z.object({
      id:    z.string().uuid(),
      name:  z.string().min(1).optional(),
      email: z.string().email().optional(),
    }))
    .mutation(async ({ input: { id, ...data } }) => {
      return svc.update(id, data);
    }),

  // Mutation: delete
  remove: protectedProcedure
    .input(z.object({ id: z.string().uuid() }))
    .mutation(async ({ input }) => {
      await svc.delete(input.id);
      return { success: true };
    }),
});

// src/server/routers/index.ts
import { router } from '../trpc';
import { usersRouter } from './users';

export const appRouter = router({
  users: usersRouter,
});

// Export the router type — import this on the client side
export type AppRouter = typeof appRouter;
```

## HTTP Server (Express adapter)

```typescript
// src/server/index.ts
import express from 'express';
import { createExpressMiddleware } from '@trpc/server/adapters/express';
import { appRouter } from './routers';
import { createContext } from './context';

const app = express();

app.use('/trpc', createExpressMiddleware({
  router: appRouter,
  createContext,
  onError({ error, path }) {
    // Log server errors; client errors (4xx) are expected
    if (error.code === 'INTERNAL_SERVER_ERROR') {
      console.error(`tRPC error on ${path}:`, error);
    }
  },
}));

app.get('/health', (_req, res) => res.json({ status: 'ok' }));

const port = process.env.PORT || 8080;
app.listen(port, () => console.log(`Server on :${port}`));
```

## Type-Safe Client

```typescript
// src/client/trpc.ts
import { createTRPCClient, httpBatchLink } from '@trpc/client';
import type { AppRouter } from '../server/routers';  // import TYPE only

export const trpc = createTRPCClient<AppRouter>({
  links: [
    httpBatchLink({
      url: process.env.API_URL || 'http://localhost:8080/trpc',
      headers() {
        const token = localStorage.getItem('token');
        return token ? { Authorization: `Bearer ${token}` } : {};
      },
    }),
  ],
});

// Usage — fully typed, no type assertions needed
const users = await trpc.users.list.query();
const user  = await trpc.users.byId.query({ id: '123e4567-...' });
await trpc.users.create.mutate({ name: 'Alice', email: 'alice@example.com' });
```

## Standalone HTTP Server (no Express)

```typescript
import { createHTTPServer } from '@trpc/server/adapters/standalone';
import { appRouter } from './routers';
import { createContext } from './context';

const server = createHTTPServer({ router: appRouter, createContext });
server.listen(8080, () => console.log('tRPC standalone server on :8080'));
```

## Error Handling

```typescript
import { TRPCError } from '@trpc/server';

// tRPC error codes map to HTTP status codes automatically
throw new TRPCError({ code: 'NOT_FOUND',          message: 'User not found' });   // 404
throw new TRPCError({ code: 'BAD_REQUEST',         message: 'Invalid input' });   // 400
throw new TRPCError({ code: 'UNAUTHORIZED',        message: 'Not authenticated' }); // 401
throw new TRPCError({ code: 'FORBIDDEN',           message: 'Access denied' });   // 403
throw new TRPCError({ code: 'CONFLICT',            message: 'Already exists' });  // 409
throw new TRPCError({ code: 'INTERNAL_SERVER_ERROR', message: 'Server error' });  // 500

// Wrap unknown errors
.mutation(async ({ input }) => {
  try {
    return await svc.create(input);
  } catch (err) {
    throw new TRPCError({
      code: 'INTERNAL_SERVER_ERROR',
      message: 'Failed to create user',
      cause: err,
    });
  }
})
```

## Key Rules

- Import `AppRouter` as a **type-only import** on the client (`import type`) — never import server code into the client bundle.
- Zod schemas are the single source of truth for input validation; tRPC automatically returns a `BAD_REQUEST` error when validation fails.
- Use `t.procedure` middleware (`.use()`) for cross-cutting concerns like auth — define them once in `trpc.ts` and compose.
- `httpBatchLink` batches multiple concurrent calls into a single HTTP request — use it in production clients.
- Procedures are tree-shakeable: unused procedures do not affect bundle size on the client.
- Read all config from environment variables; validate required vars at startup.
