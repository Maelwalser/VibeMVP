# Node.js + Express Skill Guide

## Project Layout

```
service-name/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts          # Entry point
│   ├── app.ts            # Express app setup
│   ├── routes/           # Router modules
│   ├── handlers/         # Request handlers
│   ├── middleware/        # Custom middleware
│   ├── services/         # Business logic
│   └── repositories/     # Data access
└── Dockerfile
```

## package.json Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "express": "<version>",
    "cors": "<version>",
    "helmet": "<version>",
    "express-rate-limit": "<version>"
  },
  "devDependencies": {
    "@types/express": "<version>",
    "@types/cors": "<version>",
    "@types/node": "<version>",
    "typescript": "<version>",
    "ts-node": "<version>"
  }
}
```

## Server Setup

```typescript
// src/app.ts
import express, { Application } from 'express';
import cors from 'cors';
import helmet from 'helmet';
import { userRouter } from './routes/users';
import { errorHandler } from './middleware/errorHandler';

export function createApp(): Application {
  const app = express();

  // Body parsing
  app.use(express.json());
  app.use(express.urlencoded({ extended: true }));

  // Security middleware
  app.use(helmet());
  app.use(cors({ origin: process.env.ALLOWED_ORIGIN }));

  // Health check
  app.get('/health', (_req, res) => res.json({ status: 'ok' }));

  // Routes
  app.use('/api/users', userRouter);

  // 404 handler (after all routes)
  app.use((_req, res) => res.status(404).json({ error: 'Not found' }));

  // Error handler must be last (4 args)
  app.use(errorHandler);

  return app;
}

// src/index.ts
import { createApp } from './app';

const app = createApp();
const port = process.env.PORT || 8080;
app.listen(port, () => console.log(`Listening on :${port}`));
```

## Route / Handler Pattern

```typescript
// src/routes/users.ts
import { Router } from 'express';
import { UserHandler } from '../handlers/userHandler';
import { UserService } from '../services/userService';

const svc = new UserService();
const handler = new UserHandler(svc);

export const userRouter = Router();

userRouter.get('/',        asyncWrap(handler.list.bind(handler)));
userRouter.get('/:id',     asyncWrap(handler.getById.bind(handler)));
userRouter.post('/',       asyncWrap(handler.create.bind(handler)));
userRouter.put('/:id',     asyncWrap(handler.update.bind(handler)));
userRouter.delete('/:id',  asyncWrap(handler.remove.bind(handler)));

// src/handlers/userHandler.ts
import { Request, Response } from 'express';
import { UserService } from '../services/userService';

export class UserHandler {
  constructor(private svc: UserService) {}

  async list(_req: Request, res: Response): Promise<void> {
    const users = await this.svc.findAll();
    res.json({ data: users });
  }

  async getById(req: Request, res: Response): Promise<void> {
    const user = await this.svc.findById(req.params.id);
    if (!user) {
      res.status(404).json({ error: 'User not found' });
      return;
    }
    res.json({ data: user });
  }

  async create(req: Request, res: Response): Promise<void> {
    const user = await this.svc.create(req.body);
    res.status(201).json({ data: user });
  }

  async update(req: Request, res: Response): Promise<void> {
    const user = await this.svc.update(req.params.id, req.body);
    res.json({ data: user });
  }

  async remove(req: Request, res: Response): Promise<void> {
    await this.svc.delete(req.params.id);
    res.status(204).send();
  }
}
```

## Async Handler Wrapper

Catches Promise rejections and forwards them to the error handler — avoids try/catch in every handler.

```typescript
// src/middleware/asyncWrap.ts
import { Request, Response, NextFunction, RequestHandler } from 'express';

type AsyncHandler = (req: Request, res: Response, next: NextFunction) => Promise<void>;

export function asyncWrap(fn: AsyncHandler): RequestHandler {
  return (req, res, next) => {
    fn(req, res, next).catch(next);
  };
}
```

## Middleware

```typescript
// src/middleware/auth.ts
import { Request, Response, NextFunction } from 'express';
import { verifyToken } from '../services/auth';

export function requireAuth(req: Request, res: Response, next: NextFunction): void {
  const token = req.headers.authorization?.replace('Bearer ', '');
  if (!token) {
    res.status(401).json({ error: 'Missing token' });
    return;
  }
  try {
    const payload = verifyToken(token);
    (req as any).user = payload;
    next();
  } catch {
    res.status(401).json({ error: 'Invalid token' });
  }
}

// Apply to a single route
userRouter.delete('/:id', requireAuth, asyncWrap(handler.remove.bind(handler)));

// Apply to all routes in a router
userRouter.use(requireAuth);
```

## Error Handling

The 4-argument signature tells Express this is an error handler — it must come last in `app.use()` calls.

```typescript
// src/middleware/errorHandler.ts
import { Request, Response, NextFunction } from 'express';

export class AppError extends Error {
  constructor(public statusCode: number, message: string) {
    super(message);
    this.name = 'AppError';
  }
}

export function errorHandler(
  err: Error,
  _req: Request,
  res: Response,
  _next: NextFunction
): void {
  if (err instanceof AppError) {
    res.status(err.statusCode).json({ error: err.message });
    return;
  }

  // Log unexpected errors server-side; never expose internals
  console.error('Unhandled error:', err);
  res.status(500).json({ error: 'Internal server error' });
}

// Throwing from a handler (asyncWrap forwards to errorHandler)
async create(req: Request, res: Response): Promise<void> {
  if (!req.body.name) {
    throw new AppError(400, 'Name is required');
  }
  const user = await this.svc.create(req.body);
  res.status(201).json({ data: user });
}
```

## Key Rules

- Always use `asyncWrap` for async handlers — unhandled Promise rejections crash the process without it.
- Error handler signature must have exactly 4 parameters `(err, req, res, next)`.
- Register the error handler after all routes and other `app.use()` calls.
- Use `express.json()` before any route that reads `req.body`.
- Never call `next()` after sending a response.
- Read config from environment variables; validate required vars at startup.
