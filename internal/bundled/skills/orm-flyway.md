# Flyway Skill Guide

## Naming Convention

```
V{version}__{description}.sql    → Versioned migration (runs once)
R__{description}.sql             → Repeatable migration (runs when content changes)
U{version}__{description}.sql    → Undo migration (Flyway Teams only)
```

Examples:
```
V1__create_users_table.sql
V2__add_user_role_column.sql
V3__add_posts_table.sql
R__create_reporting_views.sql
```

Rules:
- Double underscore between version and description.
- Version: integer, decimal (1.2), or timestamp (20240115123045).
- Description: words separated by underscores or spaces.
- Never edit or rename a migration once applied — Flyway will refuse to run.

## flyway_schema_history Table

Flyway creates and manages this table automatically. It records:
- installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success

Do not manually modify this table.

## Spring Boot Integration

```xml
<!-- pom.xml -->
<dependency>
    <groupId>org.flywaydb</groupId>
    <artifactId>flyway-core</artifactId>
</dependency>
<dependency>
    <groupId>org.flywaydb</groupId>
    <artifactId>flyway-database-postgresql</artifactId>
</dependency>
```

```yaml
# application.yml
spring:
  datasource:
    url: ${DATABASE_URL}
    username: ${DB_USER}
    password: ${DB_PASSWORD}
  flyway:
    enabled: true
    locations: classpath:db/migration
    baseline-on-migrate: true      # For existing DBs without history table
    baseline-version: 0
    validate-on-migrate: true      # Reject modified migrations
    out-of-order: false            # Reject out-of-sequence migrations
    clean-disabled: true           # NEVER allow flyway:clean in production
```

Spring Boot auto-runs migrations on startup before the application context is ready.

## Maven Plugin

```xml
<plugin>
    <groupId>org.flywaydb</groupId>
    <artifactId>flyway-maven-plugin</artifactId>
    <configuration>
        <url>${env.DATABASE_URL}</url>
        <user>${env.DB_USER}</user>
        <password>${env.DB_PASSWORD}</password>
        <locations>
            <location>classpath:db/migration</location>
        </locations>
    </configuration>
</plugin>
```

```bash
mvn flyway:info      # Show migration status
mvn flyway:migrate   # Apply pending migrations
mvn flyway:validate  # Check checksums match
mvn flyway:repair    # Fix failed migration entries in history table
mvn flyway:clean     # DROP ALL OBJECTS — dev only!
```

## Gradle Plugin

> ⚠ **Use version from the "Dependency & API Reference" section** of your task prompt — never hardcode version strings.

```kotlin
// build.gradle.kts
plugins {
    id("org.flywaydb.flyway") version "<version>"
}

flyway {
    url = System.getenv("DATABASE_URL")
    user = System.getenv("DB_USER")
    password = System.getenv("DB_PASSWORD")
    locations = arrayOf("classpath:db/migration")
    cleanDisabled = true
}
```

```bash
./gradlew flywayInfo
./gradlew flywayMigrate
./gradlew flywayValidate
```

## CLI Usage

```bash
# Download CLI
wget -qO- https://download.red-gate.com/maven/release/com/redgate/flyway/flyway-commandline/10.8.1/flyway-commandline-10.8.1-linux-x64.tar.gz | tar xvz

./flyway -url=jdbc:postgresql://localhost/mydb \
         -user=postgres \
         -password=secret \
         migrate
```

## CI/CD Integration

```yaml
# GitHub Actions
- name: Run Flyway migrations
  run: |
    docker run --rm \
      -e FLYWAY_URL=jdbc:postgresql://${{ env.DB_HOST }}/mydb \
      -e FLYWAY_USER=${{ secrets.DB_USER }} \
      -e FLYWAY_PASSWORD=${{ secrets.DB_PASSWORD }} \
      -v ${{ github.workspace }}/src/main/resources/db/migration:/flyway/sql \
      redgate/flyway migrate
```

## Migration Examples

```sql
-- V1__create_users_table.sql
CREATE TABLE users (
    id         UUID         NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    name       VARCHAR(255),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_users_email ON users (email);
```

```sql
-- V2__add_user_role.sql
ALTER TABLE users ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'USER';
CREATE INDEX idx_users_role ON users (role);
```

```sql
-- V3__add_posts_table.sql
CREATE TABLE posts (
    id         UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    title      VARCHAR(500) NOT NULL,
    content    TEXT,
    published  BOOLEAN     NOT NULL DEFAULT false,
    author_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_posts_author ON posts (author_id);
```

```sql
-- R__create_reporting_views.sql
-- Repeatable: re-runs whenever this file's checksum changes
CREATE OR REPLACE VIEW user_post_counts AS
SELECT u.id, u.email, COUNT(p.id) AS post_count
FROM users u
LEFT JOIN posts p ON p.author_id = u.id
GROUP BY u.id, u.email;
```

## Rollback Strategy

Flyway Community Edition does not support automatic rollbacks. Use the forward-fix approach:

```sql
-- V4__revert_user_role.sql  (rollback = new forward migration)
ALTER TABLE users DROP COLUMN IF EXISTS role;
```

For production rollback capability, use Flyway Teams undo migrations:

```sql
-- U2__add_user_role.sql (paired undo for V2)
ALTER TABLE users DROP COLUMN role;
```

## Error Recovery

```bash
# When a migration fails mid-way, fix the script then:
mvn flyway:repair   # Removes failed entry from history table
# Re-apply
mvn flyway:migrate
```

## Anti-Patterns

- Never edit a migration file that has been applied anywhere — Flyway checksum validation will reject it.
- Never rename applied migration files — they are tracked by name in the history table.
- Never run `flyway:clean` in any environment with valuable data.
- Do not use `out-of-order=true` in production — it allows time-travel bugs.
- Keep each migration focused: one logical change per file, DDL and DML separate.
