# Node.js + Fastify Skill Guide

## Project Layout

```
service-name/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts          # Entry point
│   ├── app.ts            # Fastify instance + plugin registration
│   ├── plugins/          # fastify-plugin decorated shared utilities
│   ├── routes/           # Route modules (registered as plugins)
│   ├── schemas/          # JSON Schema / TypeBox definitions
│   ├── services/         # Business logic
│   └── repositories/     # Data access
└── Dockerfile
```

## package.json Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "fastify": "<version>",
    "@fastify/cors": "<version>",
    "@fastify/helmet": "<version>",
    "@fastify/rate-limit": "<version>",
    "@sinclair/typebox": "<version>"
  },
  "devDependencies": {
    "@types/node": "<version>",
    "typescript": "<version>",
    "ts-node": "<version>"
  }
}
```

## Server Setup

```typescript
// src/app.ts
import Fastify, { FastifyInstance } from 'fastify';
import cors from '@fastify/cors';
import helmet from '@fastify/helmet';
import rateLimit from '@fastify/rate-limit';
import { userRoutes } from './routes/users';

export async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: true });

  // Security plugins
  await app.register(helmet);
  await app.register(cors, { origin: process.env.ALLOWED_ORIGIN });
  await app.register(rateLimit, { max: 100, timeWindow: '1 minute' });

  // Domain routes
  await app.register(userRoutes, { prefix: '/api/users' });

  // Health check
  app.get('/health', async () => ({ status: 'ok' }));

  return app;
}

// src/index.ts
import { buildApp } from './app';

const app = await buildApp();
const port = Number(process.env.PORT) || 8080;

try {
  await app.listen({ port, host: '0.0.0.0' });
} catch (err) {
  app.log.error(err);
  process.exit(1);
}
```

## Route / Handler Pattern with JSON Schema Validation

```typescript
// src/schemas/user.ts
import { Type, Static } from '@sinclair/typebox';

export const CreateUserBody = Type.Object({
  name:  Type.String({ minLength: 1 }),
  email: Type.String({ format: 'email' }),
});
export type CreateUserBodyType = Static<typeof CreateUserBody>;

export const UserParams = Type.Object({
  id: Type.String({ format: 'uuid' }),
});
export type UserParamsType = Static<typeof UserParams>;

export const UserResponse = Type.Object({
  id:    Type.String(),
  name:  Type.String(),
  email: Type.String(),
});

// src/routes/users.ts
import { FastifyPluginAsync } from 'fastify';
import { CreateUserBody, CreateUserBodyType, UserParams, UserParamsType, UserResponse } from '../schemas/user';
import { UserService } from '../services/userService';

export const userRoutes: FastifyPluginAsync = async (app) => {
  const svc = new UserService();

  app.get('/', {
    schema: { response: { 200: Type.Array(UserResponse) } },
    handler: async () => svc.findAll(),
  });

  app.get<{ Params: UserParamsType }>('/:id', {
    schema: { params: UserParams, response: { 200: UserResponse } },
    handler: async (req, reply) => {
      const user = await svc.findById(req.params.id);
      if (!user) return reply.status(404).send({ error: 'Not found' });
      return user;
    },
  });

  app.post<{ Body: CreateUserBodyType }>('/', {
    schema: { body: CreateUserBody, response: { 201: UserResponse } },
    handler: async (req, reply) => {
      const user = await svc.create(req.body);
      return reply.status(201).send(user);
    },
  });
};
```

## Shared Decorators with fastify-plugin

`fastify-plugin` breaks Fastify's plugin encapsulation so decorators are visible across the whole app.

```typescript
// src/plugins/db.ts
import fp from 'fastify-plugin';
import { FastifyPluginAsync } from 'fastify';
import { Pool } from 'pg';

declare module 'fastify' {
  interface FastifyInstance {
    db: Pool;
  }
}

const dbPlugin: FastifyPluginAsync = async (app) => {
  const pool = new Pool({ connectionString: process.env.DATABASE_URL });
  app.decorate('db', pool);
  app.addHook('onClose', async () => pool.end());
};

export default fp(dbPlugin);

// Usage in any route after registration:
// app.db.query('SELECT ...')
```

## Lifecycle Hooks

```typescript
// onRequest — runs before body parsing; useful for auth
app.addHook('onRequest', async (req, reply) => {
  const token = req.headers.authorization?.replace('Bearer ', '');
  if (!token) {
    return reply.status(401).send({ error: 'Unauthorized' });
  }
  (req as any).user = verifyToken(token);
});

// preHandler — runs after parsing; input is available
app.addHook('preHandler', async (req) => {
  app.log.info({ url: req.url }, 'incoming request');
});

// onSend — modify or inspect the response before sending
app.addHook('onSend', async (_req, reply, payload) => {
  reply.header('X-Response-Time', Date.now().toString());
  return payload;
});
```

## Error Handling

```typescript
// src/errors.ts
export class AppError extends Error {
  constructor(public statusCode: number, message: string) {
    super(message);
    this.name = 'AppError';
  }
}

// Global error handler set on the Fastify instance
app.setErrorHandler((err, _req, reply) => {
  if (err instanceof AppError) {
    return reply.status(err.statusCode).send({ error: err.message });
  }
  // Fastify validation errors have statusCode 400
  if (err.statusCode === 400) {
    return reply.status(400).send({ error: err.message });
  }
  app.log.error(err);
  return reply.status(500).send({ error: 'Internal server error' });
});
```

## Key Rules

- Register plugins with `await app.register()` — order matters; plugins registered later cannot see decorators from sibling plugins unless wrapped with `fastify-plugin`.
- Always define `schema.body`, `schema.params`, and `schema.response` — Fastify serializes responses 2–3× faster when a response schema is provided.
- Use TypeBox (`@sinclair/typebox`) for schemas so TypeScript types and JSON Schema stay in sync.
- Lifecycle order: `onRequest` → `preParsing` → `preValidation` → `preHandler` → handler → `onSend` → `onResponse`.
- Never throw inside `onSend` hooks; return modified payload instead.
- Read all config from environment variables; validate required vars at startup before calling `app.listen()`.
