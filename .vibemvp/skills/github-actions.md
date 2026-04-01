# GitHub Actions Skill Guide

## File Location

All workflow files go in `.github/workflows/`.

## Go Service CI Pipeline

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Run tests
        run: go test -race -cover ./...

      - name: Run vet
        run: go vet ./...

  build:
    name: Build and Push
    runs-on: ubuntu-latest
    needs: test
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4

      - name: Log in to container registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          push: true
          tags: ghcr.io/${{ github.repository }}:${{ github.sha }}
```

## Secrets Management

Store secrets in GitHub repository settings under Settings → Secrets and variables → Actions.

Reference in workflow:
```yaml
env:
  DATABASE_URL: ${{ secrets.DATABASE_URL }}
```

## Matrix Builds

```yaml
strategy:
  matrix:
    go-version: ['1.21', '1.22']
    os: [ubuntu-latest, macos-latest]
runs-on: ${{ matrix.os }}
```

## Caching Dependencies

```yaml
- uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    restore-keys: ${{ runner.os }}-go-
```

## Deploy Stage Pattern

```yaml
deploy:
  name: Deploy
  runs-on: ubuntu-latest
  needs: build
  environment: production
  steps:
    - name: Deploy to production
      run: |
        # Add deployment commands here
        echo "Deploying ${{ github.sha }}"
```
