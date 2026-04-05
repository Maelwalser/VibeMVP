# Node.js + NestJS Skill Guide

## Project Layout

```
service-name/
├── package.json
├── tsconfig.json
├── src/
│   ├── main.ts                  # Bootstrap
│   ├── app.module.ts            # Root module
│   ├── users/
│   │   ├── users.module.ts
│   │   ├── users.controller.ts
│   │   ├── users.service.ts
│   │   ├── users.repository.ts
│   │   ├── dto/
│   │   │   ├── create-user.dto.ts
│   │   │   └── update-user.dto.ts
│   │   └── entities/
│   │       └── user.entity.ts
│   ├── auth/
│   │   ├── auth.module.ts
│   │   ├── auth.guard.ts
│   │   └── jwt.strategy.ts
│   └── common/
│       ├── filters/
│       ├── interceptors/
│       └── pipes/
└── Dockerfile
```

## package.json Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never use the placeholder versions below.

```json
{
  "dependencies": {
    "@nestjs/common": "<version>",
    "@nestjs/core": "<version>",
    "@nestjs/platform-express": "<version>",
    "class-validator": "<version>",
    "class-transformer": "<version>",
    "reflect-metadata": "<version>",
    "rxjs": "<version>"
  },
  "devDependencies": {
    "@nestjs/cli": "<version>",
    "@nestjs/testing": "<version>",
    "typescript": "<version>"
  }
}
```

## Bootstrap

```typescript
// src/main.ts
import { NestFactory } from '@nestjs/core';
import { ValidationPipe } from '@nestjs/common';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  // Global validation — rejects requests with invalid DTOs
  app.useGlobalPipes(new ValidationPipe({
    whitelist: true,        // strip unknown properties
    forbidNonWhitelisted: true,
    transform: true,        // auto-cast primitives (string → number)
  }));

  app.setGlobalPrefix('api');

  const port = process.env.PORT || 8080;
  await app.listen(port);
}

bootstrap();
```

## Module + Controller + Service

```typescript
// src/users/users.module.ts
import { Module } from '@nestjs/common';
import { UsersController } from './users.controller';
import { UsersService } from './users.service';
import { UsersRepository } from './users.repository';

@Module({
  controllers: [UsersController],
  providers: [UsersService, UsersRepository],
  exports: [UsersService],   // export if other modules need UsersService
})
export class UsersModule {}

// src/users/users.controller.ts
import { Controller, Get, Post, Put, Delete, Param, Body, HttpCode, HttpStatus } from '@nestjs/common';
import { UsersService } from './users.service';
import { CreateUserDto } from './dto/create-user.dto';
import { UpdateUserDto } from './dto/update-user.dto';

@Controller('users')
export class UsersController {
  constructor(private readonly usersService: UsersService) {}

  @Get()
  findAll() {
    return this.usersService.findAll();
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.usersService.findOne(id);
  }

  @Post()
  @HttpCode(HttpStatus.CREATED)
  create(@Body() dto: CreateUserDto) {
    return this.usersService.create(dto);
  }

  @Put(':id')
  update(@Param('id') id: string, @Body() dto: UpdateUserDto) {
    return this.usersService.update(id, dto);
  }

  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  remove(@Param('id') id: string) {
    return this.usersService.remove(id);
  }
}

// src/users/users.service.ts
import { Injectable, NotFoundException } from '@nestjs/common';
import { UsersRepository } from './users.repository';
import { CreateUserDto } from './dto/create-user.dto';
import { UpdateUserDto } from './dto/update-user.dto';

@Injectable()
export class UsersService {
  constructor(private readonly repo: UsersRepository) {}

  async findAll() {
    return this.repo.findAll();
  }

  async findOne(id: string) {
    const user = await this.repo.findById(id);
    if (!user) throw new NotFoundException(`User ${id} not found`);
    return user;
  }

  async create(dto: CreateUserDto) {
    return this.repo.create(dto);
  }

  async update(id: string, dto: UpdateUserDto) {
    await this.findOne(id);  // throws NotFoundException if missing
    return this.repo.update(id, dto);
  }

  async remove(id: string) {
    await this.findOne(id);
    return this.repo.delete(id);
  }
}
```

## DTO Validation with class-validator

```typescript
// src/users/dto/create-user.dto.ts
import { IsString, IsEmail, IsNotEmpty, MinLength } from 'class-validator';

export class CreateUserDto {
  @IsString()
  @IsNotEmpty()
  @MinLength(2)
  name: string;

  @IsEmail()
  email: string;
}

// src/users/dto/update-user.dto.ts
import { PartialType } from '@nestjs/mapped-types';
import { CreateUserDto } from './create-user.dto';

export class UpdateUserDto extends PartialType(CreateUserDto) {}
```

## Guards (Authentication / Authorization)

```typescript
// src/auth/auth.guard.ts
import { CanActivate, ExecutionContext, Injectable, UnauthorizedException } from '@nestjs/common';
import { verifyToken } from './jwt.strategy';

@Injectable()
export class AuthGuard implements CanActivate {
  canActivate(context: ExecutionContext): boolean {
    const request = context.switchToHttp().getRequest();
    const token = request.headers.authorization?.replace('Bearer ', '');

    if (!token) throw new UnauthorizedException('Missing token');

    try {
      request.user = verifyToken(token);
      return true;
    } catch {
      throw new UnauthorizedException('Invalid token');
    }
  }
}

// Apply to a single route
@Get('profile')
@UseGuards(AuthGuard)
getProfile(@Request() req) {
  return req.user;
}

// Apply globally in main.ts
app.useGlobalGuards(new AuthGuard());
```

## Interceptors

```typescript
// src/common/interceptors/logging.interceptor.ts
import { Injectable, NestInterceptor, ExecutionContext, CallHandler } from '@nestjs/common';
import { Observable, tap } from 'rxjs';

@Injectable()
export class LoggingInterceptor implements NestInterceptor {
  intercept(context: ExecutionContext, next: CallHandler): Observable<unknown> {
    const req = context.switchToHttp().getRequest();
    const start = Date.now();

    return next.handle().pipe(
      tap(() => {
        console.log(`${req.method} ${req.url} — ${Date.now() - start}ms`);
      }),
    );
  }
}

// Register globally
app.useGlobalInterceptors(new LoggingInterceptor());
```

## Exception Filters

```typescript
// src/common/filters/http-exception.filter.ts
import { ExceptionFilter, Catch, ArgumentsHost, HttpException } from '@nestjs/common';
import { Response } from 'express';

@Catch(HttpException)
export class HttpExceptionFilter implements ExceptionFilter {
  catch(exception: HttpException, host: ArgumentsHost) {
    const ctx = host.switchToHttp();
    const response = ctx.getResponse<Response>();
    const status = exception.getStatus();
    const body = exception.getResponse();

    response.status(status).json({
      statusCode: status,
      error: typeof body === 'string' ? body : (body as any).message,
      timestamp: new Date().toISOString(),
    });
  }
}

// Register globally
app.useGlobalFilters(new HttpExceptionFilter());
```

## Error Handling

```typescript
// Use built-in NestJS exceptions — they map to HTTP status codes automatically
throw new NotFoundException('User not found');       // 404
throw new BadRequestException('Invalid payload');    // 400
throw new UnauthorizedException('Token expired');    // 401
throw new ForbiddenException('Access denied');       // 403
throw new ConflictException('Email already exists'); // 409
throw new InternalServerErrorException();            // 500
```

## Key Rules

- Dependency injection is constructor-based — declare dependencies as `private readonly` constructor parameters.
- Every class used as a provider must be decorated with `@Injectable()`.
- Every controller must be declared in a module's `controllers` array; every service/repository in `providers`.
- `ValidationPipe` with `whitelist: true` is essential — it strips extra properties before they reach your handlers.
- Use `PartialType` from `@nestjs/mapped-types` for update DTOs to avoid duplicating validation decorators.
- Guards return `boolean` or throw an exception; they never send responses directly.
- Module imports/exports control visibility: a service is only available in modules that import the module that exports it.
- Read all config from environment variables; use `@nestjs/config` (ConfigModule) for structured config access.
