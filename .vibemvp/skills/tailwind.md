# Tailwind CSS Skill Guide

## Configuration (tailwind.config.ts)

```typescript
import type { Config } from 'tailwindcss';

const config: Config = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#f0f9ff',
          500: '#0ea5e9',
          900: '#0c4a6e',
        },
      },
    },
  },
  plugins: [],
};
export default config;
```

## globals.css

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

## Common Patterns

### Button

```tsx
<button className="rounded-md bg-primary-500 px-4 py-2 text-white hover:bg-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 transition-colors">
  Click me
</button>
```

### Card

```tsx
<div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
  {children}
</div>
```

### Responsive Grid

```tsx
<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
  {items.map(item => <Card key={item.id} {...item} />)}
</div>
```

### Form Input

```tsx
<input
  className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
  type="text"
/>
```

## Dark Mode

Add `darkMode: 'class'` to config, then toggle the `dark` class on `<html>`.

```tsx
<div className="bg-white text-gray-900 dark:bg-gray-900 dark:text-white">
```

## cn() Utility (with clsx + tailwind-merge)

```typescript
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```
