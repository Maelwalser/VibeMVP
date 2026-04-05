package deps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var WellKnownNpmPackages = map[string]string{
	// ── Next.js / React ────────────────────────────────────────────
	"next":               "15.3.0",
	"react":              "19.1.0",
	"react-dom":          "19.1.0",
	"@types/react":       "19.1.0",
	"@types/react-dom":   "19.1.0",
	"eslint-config-next": "15.3.0",

	// ── Vue / Nuxt ─────────────────────────────────────────────────
	"vue":          "3.5.13",
	"nuxt":         "3.16.1",
	"@nuxt/ui":     "3.0.2",
	"pinia":        "2.3.1",
	"@vueuse/core": "12.7.0",

	// ── Svelte / SvelteKit ─────────────────────────────────────────
	"svelte":                       "5.25.3",
	"@sveltejs/kit":                "2.20.1",
	"@sveltejs/vite-plugin-svelte": "5.0.3",

	// ── Angular ────────────────────────────────────────────────────
	"@angular/core":                 "19.2.5",
	"@angular/common":               "19.2.5",
	"@angular/forms":                "19.2.5",
	"@angular/router":               "19.2.5",
	"@angular/platform-browser":     "19.2.5",
	"@angular/cli":                  "19.2.5",
	"@angular-devkit/build-angular": "19.2.5",

	// ── Express / Node.js ──────────────────────────────────────────
	"express":        "4.21.2",
	"@types/express": "4.17.21",
	"cors":           "2.8.5",
	"@types/cors":    "2.8.17",
	"helmet":         "8.0.0",
	"morgan":         "1.10.0",
	"@types/morgan":  "1.9.9",
	"dotenv":         "16.4.7",

	// ── Fastify ────────────────────────────────────────────────────
	"fastify":          "5.3.2",
	"@fastify/cors":    "10.1.0",
	"@fastify/jwt":     "9.1.0",
	"@fastify/swagger": "9.5.0",

	// ── NestJS ─────────────────────────────────────────────────────
	"@nestjs/core":             "11.1.0",
	"@nestjs/common":           "11.1.0",
	"@nestjs/platform-express": "11.1.0",
	"@nestjs/config":           "4.0.2",
	"@nestjs/jwt":              "11.0.0",
	"@nestjs/typeorm":          "11.0.0",
	"@nestjs/swagger":          "11.1.4",
	"@nestjs/testing":          "11.1.0",

	// ── Hono ───────────────────────────────────────────────────────
	"hono": "4.7.5",

	// ── tRPC ───────────────────────────────────────────────────────
	"@trpc/server": "11.1.0",
	"@trpc/client": "11.1.0",

	// ── Remix ──────────────────────────────────────────────────────
	"@remix-run/node":    "2.15.3",
	"@remix-run/react":   "2.15.3",
	"@remix-run/dev":     "2.15.3",
	"@remix-run/express": "2.15.3",

	// ── Astro ──────────────────────────────────────────────────────
	"astro":             "5.6.1",
	"@astrojs/react":    "4.2.0",
	"@astrojs/tailwind": "5.1.4",

	// ── Solid.js / Qwik ────────────────────────────────────────────
	"solid-js":              "1.9.5",
	"@solidjs/router":       "0.15.3",
	"@builder.io/qwik":      "1.14.0",
	"@builder.io/qwik-city": "1.14.0",

	// ── Database / ORM ─────────────────────────────────────────────
	"drizzle-orm":    "0.41.0",
	"drizzle-kit":    "0.30.4",
	"prisma":         "6.5.0",
	"@prisma/client": "6.5.0",
	"typeorm":        "0.3.21",
	"pg":             "8.13.3",
	"@types/pg":      "8.11.11",
	"mysql2":         "3.12.0",

	// ── Auth ───────────────────────────────────────────────────────
	"next-auth":           "4.24.11",
	"jsonwebtoken":        "9.0.2",
	"@types/jsonwebtoken": "9.0.9",
	"bcryptjs":            "3.0.2",
	"@types/bcryptjs":     "2.4.6",

	// ── TypeScript / build ─────────────────────────────────────────
	"typescript":  "5.7.2",
	"@types/node": "22.10.0",
	"vite":        "6.2.5",
	"vitest":      "3.1.1",
	"tsx":         "4.19.3",
	"ts-node":     "10.9.2",

	// ── Styling ────────────────────────────────────────────────────
	"tailwindcss":          "3.4.17",
	"@tailwindcss/postcss": "4.0.17",
	"postcss":              "8.5.1",
	"autoprefixer":         "10.4.20",
	"tailwind-merge":       "2.5.5",

	// ── Linting ────────────────────────────────────────────────────
	"eslint": "9.17.0",

	// ── UI / components ────────────────────────────────────────────
	"lucide-react":             "0.468.0",
	"framer-motion":            "12.6.3",
	"clsx":                     "2.1.1",
	"class-variance-authority": "0.7.1",

	// ── State / data fetching ──────────────────────────────────────
	"@tanstack/react-query": "5.62.3",
	"@tanstack/vue-query":   "5.62.3",
	"zustand":               "5.0.2",
	"axios":                 "1.7.9",
	"swr":                   "2.3.3",

	// ── Forms / validation ─────────────────────────────────────────
	"zod":                 "3.24.1",
	"react-hook-form":     "7.54.2",
	"@hookform/resolvers": "3.9.1",
	"valibot":             "1.4.0",

	// ── Testing ────────────────────────────────────────────────────
	"@testing-library/react":      "16.2.0",
	"@testing-library/user-event": "14.5.2",
	"@playwright/test":            "1.51.1",
	"jest":                        "29.7.0",
	"@types/jest":                 "29.5.14",

	// ── Runtime / tooling ──────────────────────────────────────────
	"@types/bun": "1.2.8",
}

// LibraryAPIDocs holds the exported API surface of commonly-misused libraries.
// Injected into agent prompts to prevent hallucinated types/functions.
//
// Each entry is keyed by a lowercase technology name that matches against
// the task's technology stack.
func resolveNpmVersion(ctx context.Context, pkg, fallback string) string {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rawURL := "https://registry.npmjs.org/" + url.PathEscape(pkg) + "/latest"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fallback
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fallback
	}
	var result struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Version == "" {
		return fallback
	}
	return result.Version
}

// resolveAllNpmVersions fetches the latest versions for all packages in the fallback
// map concurrently, returning a new map with resolved (or fallback) versions.
func resolveAllNpmVersions(ctx context.Context) map[string]string {
	type result struct {
		pkg, version string
	}
	ch := make(chan result, len(WellKnownNpmPackages))
	var wg sync.WaitGroup
	for pkg, fallback := range WellKnownNpmPackages {
		wg.Add(1)
		go func(p, fb string) {
			defer wg.Done()
			ch <- result{p, resolveNpmVersion(ctx, p, fb)}
		}(pkg, fallback)
	}
	wg.Wait()
	close(ch)
	resolved := make(map[string]string, len(WellKnownNpmPackages))
	for r := range ch {
		resolved[r.pkg] = r.version
	}
	return resolved
}

// resolveGoDevToolVersion fetches the latest version of a Go dev tool from the Go
// module proxy, then reads its go.mod to find the minimum Go version it requires.
// Both version and MinGoVersion are updated from live registry data.
// Falls back to t on any error.
