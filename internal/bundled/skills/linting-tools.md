# Linting Tools Skill Guide

## Overview

Linters enforce code quality, consistency, and catch bugs before runtime. Configure linters in version-controlled files so the entire team runs the same rules.

---

## Go — golangci-lint

```bash
# Install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run
golangci-lint run ./...
golangci-lint run --fix ./...    # auto-fix where possible
```

```yaml
# .golangci.yml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  enable:
    - errcheck        # check all errors are handled
    - gosimple        # simplify code
    - govet           # go vet checks
    - ineffassign     # detect ineffectual assignments
    - staticcheck     # static analysis
    - unused          # detect unused code
    - gofmt           # formatting
    - goimports       # import organization
    - godot           # comments end with period
    - exhaustive      # exhaustive enum switches
    - wrapcheck       # errors from external packages must be wrapped
    - noctx           # HTTP requests must use context
    - revive          # fast linter, replaces golint
    - gocritic        # opinionated checks
    - cyclop          # cyclomatic complexity
    - funlen          # function length
    - maintidx        # maintainability index

linters-settings:
  funlen:
    lines: 60
    statements: 40

  cyclop:
    max-complexity: 10

  govet:
    enable:
      - shadow         # detect shadowed variables

  revive:
    rules:
      - name: exported
        arguments:
          - disableStutteringCheck

issues:
  exclude-rules:
    - path: "_test.go"
      linters:
        - funlen       # test functions are allowed to be longer
        - wrapcheck
  max-issues-per-linter: 0
  max-same-issues: 0
```

---

## Python — Ruff

Ruff replaces flake8, isort, pyupgrade, and more — runs at Rust speed.

```bash
# Install
pip install ruff
# Or via uv: uv add --dev ruff

# Run
ruff check .             # lint
ruff check --fix .       # auto-fix
ruff format .            # format (replaces black)
ruff format --check .    # format check only
```

```toml
# pyproject.toml
[tool.ruff]
target-version = "py312"
line-length = 100

[tool.ruff.lint]
select = [
    "E",    # pycodestyle errors
    "W",    # pycodestyle warnings
    "F",    # pyflakes
    "I",    # isort
    "B",    # flake8-bugbear
    "C4",   # flake8-comprehensions
    "UP",   # pyupgrade
    "N",    # pep8 naming
    "S",    # flake8-bandit (security)
    "RUF",  # ruff-specific rules
    "TCH",  # type-checking imports
    "ANN",  # annotations
    "ERA",  # eradicate (commented-out code)
    "TRY",  # tryceratops (exception handling)
]
ignore = [
    "ANN101",   # missing self annotation
    "ANN102",   # missing cls annotation
    "S101",     # assert statements allowed in tests
    "TRY003",   # allow long exception messages
]

[tool.ruff.lint.per-file-ignores]
"tests/**/*.py" = ["S", "ANN"]   # relaxed rules in tests
"migrations/**/*.py" = ["E501"]  # long lines OK in migrations

[tool.ruff.lint.isort]
known-first-party = ["myapp"]

[tool.ruff.format]
quote-style = "double"
indent-style = "space"
```

---

## TypeScript/JavaScript — ESLint (Flat Config)

```bash
npm install --save-dev eslint @eslint/js typescript-eslint eslint-plugin-unicorn
```

```javascript
// eslint.config.js (flat config — ESLint v9+)
import js from "@eslint/js";
import ts from "typescript-eslint";
import unicorn from "eslint-plugin-unicorn";

export default ts.config(
  js.configs.recommended,
  ...ts.configs.strictTypeChecked,
  ...ts.configs.stylisticTypeChecked,

  {
    plugins: {
      unicorn,
    },
    languageOptions: {
      parserOptions: {
        project: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      // TypeScript
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
      "@typescript-eslint/prefer-nullish-coalescing": "error",
      "@typescript-eslint/prefer-optional-chain": "error",
      "@typescript-eslint/no-non-null-assertion": "error",
      "@typescript-eslint/consistent-type-imports": ["error", { prefer: "type-imports" }],

      // Unicorn (modern JS practices)
      "unicorn/prefer-module": "error",
      "unicorn/no-array-for-each": "error",
      "unicorn/prefer-node-protocol": "error",
      "unicorn/no-null": "off",   // null is useful with SQL

      // General
      "no-console": "warn",
      "eqeqeq": ["error", "always"],
    },
  },

  {
    // Relax rules for test files
    files: ["**/*.test.ts", "**/*.spec.ts"],
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
      "no-console": "off",
    },
  },

  {
    // Ignore generated files and build output
    ignores: ["dist/**", "node_modules/**", "**/*.generated.ts", "coverage/**"],
  },
);
```

```json
// package.json scripts
{
  "scripts": {
    "lint": "eslint .",
    "lint:fix": "eslint . --fix",
    "typecheck": "tsc --noEmit"
  }
}
```

---

## Java — Checkstyle

```bash
# Maven plugin
mvn checkstyle:check
```

```xml
<!-- pom.xml -->
<plugin>
  <groupId>org.apache.maven.plugins</groupId>
  <artifactId>maven-checkstyle-plugin</artifactId>
  <version><!-- use version from Dependency & API Reference --></version>
  <configuration>
    <configLocation>checkstyle.xml</configLocation>
    <failsOnError>true</failsOnError>
    <violationSeverity>warning</violationSeverity>
  </configuration>
  <executions>
    <execution>
      <goals><goal>check</goal></goals>
    </execution>
  </executions>
</plugin>
```

```xml
<!-- checkstyle.xml -->
<?xml version="1.0"?>
<!DOCTYPE module PUBLIC "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
  "https://checkstyle.org/dtds/configuration_1_3.dtd">
<module name="Checker">
  <property name="severity" value="warning"/>

  <module name="TreeWalker">
    <module name="ConstantName"/>
    <module name="LocalVariableName"/>
    <module name="MemberName"/>
    <module name="MethodName"/>
    <module name="PackageName"/>
    <module name="TypeName"/>

    <module name="AvoidStarImport"/>
    <module name="UnusedImports"/>

    <module name="MethodLength">
      <property name="max" value="60"/>
    </module>
    <module name="ParameterNumber">
      <property name="max" value="7"/>
    </module>

    <module name="EmptyBlock"/>
    <module name="LeftCurly"/>
    <module name="NeedBraces"/>
    <module name="RightCurly"/>

    <module name="MagicNumber">
      <property name="ignoreNumbers" value="-1, 0, 1, 2"/>
    </module>

    <module name="JavadocMethod">
      <property name="scope" value="public"/>
    </module>
  </module>

  <module name="FileLength">
    <property name="max" value="800"/>
  </module>
  <module name="LineLength">
    <property name="max" value="120"/>
  </module>
</module>
```

---

## Kotlin — ktlint

```bash
# Via gradle task
./gradlew ktlintCheck
./gradlew ktlintFormat
```

```kotlin
// build.gradle.kts — use versions from "Dependency & API Reference" section
plugins {
    id("org.jlleitschuh.gradle.ktlint") version "<version>"
}

ktlint {
    version.set("<version>")
    android.set(false)
    outputToConsole.set(true)
    reporters {
        reporter(org.jlleitschuh.gradle.ktlint.reporter.ReporterType.PLAIN)
        reporter(org.jlleitschuh.gradle.ktlint.reporter.ReporterType.CHECKSTYLE)
    }
}
```

```ini
# .editorconfig — ktlint reads this
[*.{kt,kts}]
ij_kotlin_imports_layout = *
ktlint_standard_no-wildcard-imports = enabled
ktlint_standard_max-line-length = disabled
```

---

## Rust — Clippy

```bash
# Run Clippy
cargo clippy -- -D warnings      # treat warnings as errors

# With fixes
cargo clippy --fix --allow-dirty -- -D warnings
```

```rust
// Crate-level deny in lib.rs or main.rs
#![deny(clippy::all)]
#![deny(clippy::pedantic)]
#![deny(clippy::nursery)]
#![allow(clippy::module_name_repetitions)]  // allow per-rule exemption

// Inline allow for specific case
#[allow(clippy::too_many_arguments)]
pub fn complex_function(...) {}
```

```toml
# .cargo/config.toml
[target.'cfg(all())']
rustflags = ["-D", "warnings"]   # all warnings are errors

# Clippy.toml — configure specific lints
[Clippy]
cognitive-complexity-threshold = 15
too-many-arguments-threshold = 7
```

---

## Ruby — RuboCop

```bash
bundle exec rubocop
bundle exec rubocop -A    # auto-correct
```

```yaml
# .rubocop.yml
AllCops:
  NewCops: enable
  SuggestExtensions: false
  Exclude:
    - "db/schema.rb"
    - "bin/**/*"
    - "node_modules/**/*"

inherit_gem:
  rubocop-rails: config/rails.yml
  rubocop-rspec: config/rspec.yml
  rubocop-performance: config/performance.yml

Metrics/MethodLength:
  Max: 20

Metrics/ClassLength:
  Max: 200

Metrics/AbcSize:
  Max: 20

Style/Documentation:
  Enabled: false    # don't require class docs in small projects

Layout/LineLength:
  Max: 120
```

---

## PHP — PHP-CS-Fixer

```bash
# Install
composer require --dev friendsofphp/php-cs-fixer

# Run
vendor/bin/php-cs-fixer fix --dry-run --diff    # preview
vendor/bin/php-cs-fixer fix                      # apply
```

```php
// .php-cs-fixer.php
<?php

$finder = PhpCsFixer\Finder::create()
    ->in(__DIR__ . '/src')
    ->in(__DIR__ . '/tests')
    ->exclude('var')
    ->exclude('vendor');

return (new PhpCsFixer\Config())
    ->setRules([
        '@PSR12' => true,
        '@PHP82Migration' => true,
        'array_syntax' => ['syntax' => 'short'],
        'ordered_imports' => ['sort_algorithm' => 'alpha'],
        'no_unused_imports' => true,
        'trailing_comma_in_multiline' => true,
        'phpdoc_align' => true,
        'declare_strict_types' => true,
        'strict_param' => true,
    ])
    ->setFinder($finder)
    ->setUsingCache(true);
```

---

## CI Integration (GitHub Actions)

```yaml
# .github/workflows/lint.yml
name: Lint

on: [push, pull_request]

jobs:
  golangci-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  ruff:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: chartboost/ruff-action@v1

  eslint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: "20" }
      - run: npm ci
      - run: npm run lint
      - run: npm run typecheck
```

## Rules

- Commit linter config files — never rely on defaults that may change between versions
- Run linters in CI on every PR — block merges on lint failures
- Use `--fix` / auto-correct in local development; never auto-fix in CI (it hides the underlying issue)
- Pin linter versions in CI to prevent unexpected rule changes breaking builds
- Disable rules file-by-file with inline comments as a last resort — prefer fixing or configuring globally
