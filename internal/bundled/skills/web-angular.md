# Angular Skill Guide

## Project Layout

```
frontend/
├── angular.json
├── tsconfig.json
├── package.json
├── src/
│   ├── main.ts             # Bootstrap
│   ├── app/
│   │   ├── app.config.ts   # Application config (standalone)
│   │   ├── app.routes.ts   # Root routes
│   │   ├── app.component.ts
│   │   └── features/
│   │       ├── users/
│   │       │   ├── users.routes.ts
│   │       │   ├── user-list/
│   │       │   └── user-detail/
│   │       └── dashboard/
│   ├── shared/
│   │   ├── components/
│   │   ├── services/
│   │   └── models/
│   └── environments/
│       ├── environment.ts
│       └── environment.prod.ts
```

## Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "@angular/animations": "<version>",
    "@angular/common": "<version>",
    "@angular/core": "<version>",
    "@angular/forms": "<version>",
    "@angular/router": "<version>",
    "rxjs": "<version>"
  }
}
```

## Standalone Component (Modern)

```typescript
// src/app/features/users/user-list/user-list.component.ts
import { Component, OnInit, inject } from '@angular/core';
import { AsyncPipe, NgFor, NgIf } from '@angular/common';
import { RouterLink } from '@angular/router';
import { UserService } from '../user.service';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

@Component({
  selector: 'app-user-list',
  standalone: true,
  imports: [AsyncPipe, NgFor, NgIf, RouterLink],
  template: `
    <ul *ngIf="users$ | async as users; else loading">
      <li *ngFor="let user of users; trackBy: trackById">
        <a [routerLink]="['/users', user.id]">{{ user.name }}</a>
      </li>
    </ul>
    <ng-template #loading><p>Loading...</p></ng-template>
  `,
})
export class UserListComponent {
  private userService = inject(UserService);
  users$ = this.userService.getUsers();

  trackById(_: number, user: { id: string }) {
    return user.id;
  }
}
```

## Input / Output

```typescript
import { Component, Input, Output, EventEmitter, input, output } from '@angular/core';

// Modern signal-based (Angular 17+)
@Component({ standalone: true, selector: 'app-card', template: `...` })
export class CardComponent {
  title = input.required<string>();
  description = input('');              // optional with default
  selected = output<string>();

  select() { this.selected.emit(this.title()); }
}

// Classic decorator style
@Component({ standalone: true, selector: 'app-badge', template: `...` })
export class BadgeComponent {
  @Input() label = '';
  @Output() clicked = new EventEmitter<void>();
}
```

## Angular Signals

```typescript
import { signal, computed, effect } from '@angular/core';

@Component({ standalone: true, template: `<p>{{ doubled() }}</p>` })
export class CounterComponent {
  count = signal(0);
  doubled = computed(() => this.count() * 2);

  constructor() {
    effect(() => {
      console.log('count changed to', this.count());
    });
  }

  increment() { this.count.update(n => n + 1); }
  reset()     { this.count.set(0); }
}
```

## Service with BehaviorSubject

```typescript
import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable } from 'rxjs';
import { tap } from 'rxjs/operators';
import { environment } from '@env/environment';

interface User { id: string; name: string; email: string; }

@Injectable({ providedIn: 'root' })
export class UserService {
  private readonly api = `${environment.apiUrl}/users`;
  private usersSubject = new BehaviorSubject<User[]>([]);

  users$ = this.usersSubject.asObservable();

  constructor(private http: HttpClient) {}

  getUsers(): Observable<User[]> {
    return this.http.get<User[]>(this.api).pipe(
      tap(users => this.usersSubject.next(users))
    );
  }

  createUser(data: Omit<User, 'id'>): Observable<User> {
    return this.http.post<User>(this.api, data);
  }
}
```

## takeUntilDestroyed

```typescript
import { Component, OnInit, inject } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

@Component({ standalone: true, template: `...` })
export class SearchComponent implements OnInit {
  private service = inject(UserService);
  results: string[] = [];

  // takeUntilDestroyed must be called in constructor or field init
  private destroy$ = takeUntilDestroyed();

  ngOnInit() {
    this.service.search$.pipe(this.destroy$).subscribe(r => {
      this.results = r;
    });
  }
}
```

## Reactive Forms

```typescript
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { NgIf } from '@angular/common';

@Component({
  standalone: true,
  imports: [ReactiveFormsModule, NgIf],
  template: `
    <form [formGroup]="form" (ngSubmit)="submit()">
      <input formControlName="email" />
      <span *ngIf="form.get('email')?.invalid && form.get('email')?.touched">
        Invalid email
      </span>
      <input formControlName="password" type="password" />
      <button type="submit" [disabled]="form.invalid">Submit</button>
    </form>
  `,
})
export class LoginFormComponent {
  private fb = inject(FormBuilder);

  form = this.fb.group({
    email:    ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required, Validators.minLength(8)]],
  });

  submit() {
    if (this.form.valid) console.log(this.form.value);
  }
}
```

## Routing (App Router)

```typescript
// src/app/app.routes.ts
import { Routes } from '@angular/router';
import { authGuard } from './guards/auth.guard';

export const routes: Routes = [
  { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
  { path: 'login', loadComponent: () => import('./features/auth/login.component').then(m => m.LoginComponent) },
  {
    path: 'dashboard',
    canActivate: [authGuard],
    loadComponent: () => import('./features/dashboard/dashboard.component').then(m => m.DashboardComponent),
  },
  {
    path: 'users',
    canActivate: [authGuard],
    loadChildren: () => import('./features/users/users.routes').then(m => m.USERS_ROUTES),
  },
  { path: '**', loadComponent: () => import('./features/not-found/not-found.component').then(m => m.NotFoundComponent) },
];
```

## Async Pipe Pattern

```typescript
// Always prefer async pipe over manual subscriptions
// Template: users$ | async as users
// Auto-subscribes and unsubscribes when component is destroyed
```

## Key Rules

- Use `standalone: true` for all new components — avoid NgModule for new code.
- Use `inject()` instead of constructor injection in standalone components.
- Prefer Angular Signals (`signal`/`computed`/`effect`) over RxJS for simple local state.
- Use `async` pipe in templates — never subscribe manually without `takeUntilDestroyed`.
- Use `trackBy` in `*ngFor` to avoid unnecessary DOM re-renders.
- Lazy-load feature routes with `loadChildren` / `loadComponent`.
- Put shared services in `shared/services/` with `providedIn: 'root'`.
- Use `environment.ts` for environment-specific config, not hardcoded URLs.
