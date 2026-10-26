# SpecTest Web UI

The SpecTest Web UI is the browser dashboard bundled into the `spectest`
binary. It is built with Vite, React 19, TypeScript, and React Router, and it
talks to the Go backend over relative `/api` routes.

The Go server embeds the production build and serves it from `spectest ui`.
There is no separate mock or data mode at runtime: every screen uses the real
backend API.

## Quick start

Run the UI from a SpecTest workspace (the directory containing `spectest.yaml`
or your test suite):

```bash
spectest ui
```

The command serves the UI and API at `http://127.0.0.1:8088` and opens the
default browser. It uses the current directory as the workspace.

```bash
spectest ui --host 0.0.0.0 --port 9000 --no-open
```

| Flag | Default | Purpose |
| --- | --- | --- |
| `--host` | `127.0.0.1` | Address to bind |
| `--port`, `-p` | `8088` | Port to listen on; an open port is chosen if busy |
| `--no-open` | `false` | Do not launch the browser automatically |

The UI can read and update workspace files and LLM credentials. Only bind to a
non-loopback host on a trusted network.

## Development

Requirements:

- Node.js 20 or newer
- npm
- Go toolchain for the API server

Start the API server in one terminal:

```bash
go run ./cmd/spectest ui --no-open
```

Start Vite in another terminal:

```bash
cd web
npm install
npm run dev
```

Open `http://localhost:5173`. Vite proxies `/api` to
`http://127.0.0.1:8088` as configured in `vite.config.ts`. No environment
files or mode flags are required.

## Production build and embedding

The release workflow builds the web app and copies it into the Go embed
directory before compiling the binary. To reproduce that locally:

```bash
cd web
npm ci
npm run build
cd ..

mkdir -p internal/adapter/ui/dist
cp -r web/dist/* internal/adapter/ui/dist/

make build
```

`make build` writes `bin/spectest`. The server embeds
`internal/adapter/ui/dist` with `//go:embed dist/*` and serves it alongside the
API from `spectest ui`.

`npm run preview` only serves the static build; it does not proxy `/api`. Use
`spectest ui` when you need a working API.

## Scripts

| Script | Purpose |
| --- | --- |
| `npm run dev` | Vite dev server on port 5173, proxying `/api` to port 8088 |
| `npm run build` | Production build into `dist` |
| `npm run preview` | Serve the static build without an API |
| `npm run typecheck` | TypeScript check for the app and Vite config |
| `npm run test` | Vitest in watch mode |
| `npm run test:run` | Vitest once |

## Testing

Component tests run with Vitest, jsdom, and a Node-side Mock Service Worker
setup. MSW is a test double only: it is not loaded by the browser and does not
ship in the production bundle. The network layer is covered through the same
typed client used by the pages.

```bash
npm run test:run
npm run typecheck
npm run build
```

## Stale mock service worker

Older builds could register a browser service worker at
`/mockServiceWorker.js`. After upgrading, that worker may still intercept
`/api` requests and return old mock data. Unregister it once per origin:

1. Open DevTools and go to **Application > Service Workers**.
2. Unregister the worker for the UI origin.
3. Run a hard reload (or clear site data if it persists).

From the browser console:

```js
navigator.serviceWorker
  .getRegistrations()
  .then((registrations) => registrations.forEach((item) => item.unregister()));
```

## Project layout

```text
src/
  api/         Typed fetch client, wire types, contract mappers, SSE parser
  components/  Layout, data, run, suite, workspace, markdown, and UI primitives
  hooks/       useApi, useRunStream, useDebouncedValue
  lib/         Formatting and URL helpers
  mocks/       MSW handlers, fixtures, and in-memory store for tests
  pages/       One component per route
  styles/      Tokens plus base, layout, component, and page CSS
  test/        Vitest setup and render helper
```

## Routes

| Route | Screen |
| --- | --- |
| `/` | Dashboard: health, suite size, last run, history, logs |
| `/suite` | Suite plan and scenarios |
| `/generate` | Generate or update a suite |
| `/run` | Start a run, follow SSE progress, inspect the report |
| `/history` | Spec update history |
| `/workspace` | Workspace files and uploads |
| `/docs` | Markdown documents |
| `/settings` | LLM provider, endpoint, model, and connection test |
