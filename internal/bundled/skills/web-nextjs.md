# Next.js App Router Skill Guide

## Project Structure

```
frontend/
├── package.json
├── next.config.ts
├── tsconfig.json
├── tailwind.config.ts
├── src/
│   ├── app/
│   │   ├── layout.tsx          # Root layout (always Server Component)
│   │   ├── page.tsx            # Home page
│   │   ├── loading.tsx         # Suspense boundary
│   │   ├── error.tsx           # Error boundary ('use client')
│   │   ├── not-found.tsx       # 404
│   │   ├── (auth)/             # Route group — no path segment
│   │   │   ├── login/page.tsx
│   │   │   └── register/page.tsx
│   │   ├── dashboard/
│   │   │   ├── layout.tsx
│   │   │   └── page.tsx
│   │   └── api/                # Route handlers
│   │       └── users/
│   │           └── route.ts
│   ├── components/
│   │   ├── ui/
│   │   └── features/
│   ├── lib/
│   │   ├── api.ts
│   │   └── utils.ts
│   └── types/
│       └── api.ts
```

## Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "next": "<version>",
    "react": "<version>",
    "react-dom": "<version>"
  },
  "devDependencies": {
    "@types/node": "<version>",
    "@types/react": "<version>",
    "typescript": "<version>",
    "tailwindcss": "<version>"
  }
}
```

## Server Components (Default)

```typescript
// src/app/users/page.tsx — async Server Component
import { getUserList } from '@/lib/api';

export const metadata = { title: 'Users' };

export default async function UsersPage() {
  const users = await getUserList();   // direct async, no useEffect

  return (
    <main>
      <h1>Users ({users.length})</h1>
      <ul>
        {users.map(u => (
          <li key={u.id}>{u.name}</li>
        ))}
      </ul>
    </main>
  );
}
```

## Client Components

```typescript
'use client';
// src/components/features/SearchBar.tsx
import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';

export function SearchBar() {
  const [query, setQuery] = useState('');
  const [isPending, startTransition] = useTransition();
  const router = useRouter();

  function search(q: string) {
    setQuery(q);
    startTransition(() => {
      router.push(`/users?q=${encodeURIComponent(q)}`);
    });
  }

  return (
    <input
      value={query}
      onChange={e => search(e.target.value)}
      placeholder={isPending ? 'Searching...' : 'Search users'}
    />
  );
}
```

## Server Actions

```typescript
'use server';
// src/app/users/actions.ts
import { revalidatePath } from 'next/cache';
import { redirect } from 'next/navigation';
import { z } from 'zod';

const CreateUserSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
});

export async function createUser(prevState: unknown, formData: FormData) {
  const parsed = CreateUserSchema.safeParse({
    name: formData.get('name'),
    email: formData.get('email'),
  });

  if (!parsed.success) {
    return { errors: parsed.error.flatten().fieldErrors };
  }

  await db.users.create(parsed.data);
  revalidatePath('/users');
  redirect('/users');
}
```

## useFormStatus / useFormState

```typescript
'use client';
import { useFormState, useFormStatus } from 'react-dom';
import { createUser } from './actions';

function SubmitButton() {
  const { pending } = useFormStatus();
  return <button disabled={pending}>{pending ? 'Saving...' : 'Create'}</button>;
}

export function CreateUserForm() {
  const [state, action] = useFormState(createUser, null);

  return (
    <form action={action}>
      <input name="name" required />
      {state?.errors?.name && <p>{state.errors.name}</p>}
      <input name="email" type="email" required />
      {state?.errors?.email && <p>{state.errors.email}</p>}
      <SubmitButton />
    </form>
  );
}
```

## Route Handlers

```typescript
// src/app/api/users/route.ts
import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest) {
  const { searchParams } = request.nextUrl;
  const q = searchParams.get('q') ?? '';
  const users = await db.users.search(q);
  return NextResponse.json(users);
}

export async function POST(request: NextRequest) {
  const body = await request.json();
  const user = await db.users.create(body);
  return NextResponse.json(user, { status: 201 });
}
```

## Cache Invalidation

```typescript
import { revalidatePath, revalidateTag } from 'next/cache';

// Revalidate a specific path
revalidatePath('/users');
revalidatePath('/users/[id]', 'page');

// Tag-based revalidation
// Tag when fetching:
const users = await fetch('/api/users', { next: { tags: ['users'] } });
// Invalidate by tag:
revalidateTag('users');
```

## next/image

```typescript
import Image from 'next/image';

// Fixed size
<Image src={user.avatar} alt={user.name} width={48} height={48} />

// Responsive fill (parent must have position: relative + dimensions)
<div style={{ position: 'relative', width: '100%', height: 400 }}>
  <Image src="/hero.jpg" alt="Hero" fill sizes="(max-width: 768px) 100vw, 50vw" />
</div>
```

## generateStaticParams (SSG)

```typescript
// src/app/users/[id]/page.tsx
export async function generateStaticParams() {
  const users = await db.users.findAll();
  return users.map(u => ({ id: u.id }));
}

export default async function UserPage({ params }: { params: { id: string } }) {
  const user = await db.users.findById(params.id);
  if (!user) notFound();
  return <UserProfile user={user} />;
}
```

## Loading / Error / Not-Found

```typescript
// src/app/users/loading.tsx (auto Suspense boundary)
export default function Loading() {
  return <div className="skeleton" aria-label="Loading users..." />;
}

// src/app/users/error.tsx (must be 'use client')
'use client';
export default function Error({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div>
      <p>Error: {error.message}</p>
      <button onClick={reset}>Retry</button>
    </div>
  );
}

// src/app/users/not-found.tsx
export default function NotFound() {
  return <p>User not found.</p>;
}
```

## Environment Variables

```
NEXT_PUBLIC_API_URL=...   # exposed to browser
DATABASE_URL=...          # server only (never sent to client)
```

## Key Rules

- All components are Server Components by default — add `'use client'` only when needed.
- `'use server'` marks a module or function as a Server Action (called from client).
- Server Actions must validate all input with Zod or equivalent before DB access.
- Use `loading.tsx` + `error.tsx` files for automatic Suspense/ErrorBoundary wrappers.
- Use `revalidatePath`/`revalidateTag` instead of client-side cache invalidation.
- Never expose server-only env vars without `NEXT_PUBLIC_` prefix.
- Use `next/image` for all images — it handles optimization and lazy loading.
- Call `notFound()` from `next/navigation` to render the `not-found.tsx` component.
