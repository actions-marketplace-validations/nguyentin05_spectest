# SpecTest Web UI

Mock-first control center for SpecTest. The app is built with Vite, React 19,
TypeScript, and React Router, and it talks to the Go backend through a single
typed API client. Every endpoint has a Mock Service Worker implementation, so
the UI is fully usable without a backend and can switch to the real one without
component changes.

## Requirements

- Node.js 20 or newer
- npm
- Real mode: the SpecTest UI server (`spectest ui`) listening on
  `http://localhost:4173`

## Run locally

```bash
cd web
npm install
npm run dev
```

Open `http://localhost:5173`. Mock mode is the default and needs no backend.

### Real backend

Start the Go UI server first:

```bash
spectest ui
```

It serves the API at `http://localhost:4173/api`. Then run:

```bash
cd web
npm run dev:real
```

Vite proxies `/api` to `http://localhost:4173`, as configured in
`vite.config.ts`.

Production builds:

```bash
npm run build       # mock mode
npm run build:real  # real backend mode
npm run preview     # serve the last build
```

## How the switch works

`src/lib/mode.ts` resolves the mode once:

1. `VITE_API_MODE=mock|real` when set.
2. Otherwise Vite's `--mode real` (used by `dev:real` and `build:real`) means
   real.
3. Everything else means mock.

Components only call `api.*` from `src/api/endpoints.ts`. The MSW worker is
dynamically imported in `src/main.tsx` only when the mode is mock, so mock code
is not loaded at runtime in real mode (the chunk is still emitted by the build).
Switching modes requires no component edits.

## Mock endpoint map

| Method | Endpoint | Mock behavior |
| --- | --- | --- |
| GET | `/api/health` | Fixed health and work directory fixture |
| GET | `/api/logs` | In-memory session log, newest last |
| GET | `/api/config` | Masked LLM configuration |
| POST | `/api/config` | Updates the in-memory configuration |
| POST | `/api/config/test` | Runs a delayed success; an API key of `invalid` returns 401 |
| POST | `/api/server/healthcheck` | Ready or down result plus latency |
| GET | `/api/specs/workspace` | Fixture workspace files |
| POST | `/api/specs/upload` | Multipart file or raw `{ file_name, content }`, added in memory |
| GET | `/api/specs/plan?path=` | Fixture plan: 2 scenarios, servers, setups |
| GET | `/api/specs/history?dir=` | Fixture base snapshot plus update blocks |
| POST | `/api/generate` | Delayed generated-suite result |
| POST | `/api/run` | Starts a scripted run and returns its `run_id` |
| GET | `/api/run/stream` | SSE: start, scenario_start, step_result, complete, cancelled |
| POST | `/api/run/cancel` | Cancels the active run; 409 when idle |
| GET | `/api/run/report` | 404 until a run ends, then 5 report records |
| GET | `/api/markdown/read?path=` | Fixture Markdown; 404 for unknown paths |

Mock code lives in `src/mocks`. Handlers are grouped by feature under
`src/mocks/handlers`, fixtures and in-memory state are in `fixtures.ts` and
`store.ts`.

The mock run has 2 scenarios and 5 HTTP steps. `mockState.runDelayMs` controls
stream pacing; cancelling builds a partial report. Mock state resets on reload.

## Scripts

| Script | Purpose |
| --- | --- |
| `npm run dev` | Vite dev server in mock mode on port 5173 |
| `npm run dev:real` | Vite dev server in real mode (proxies `/api` to :4173) |
| `npm run build` | Production build in mock mode |
| `npm run build:real` | Production build in real mode |
| `npm run preview` | Preview the last build |
| `npm run typecheck` | TypeScript check for app and Vite config |
| `npm run test` | Vitest in watch mode |
| `npm run test:run` | Vitest once |

## Project layout

```
src/
  api/         Typed fetch client, wire types, SSE parser
  components/  Layout, data, run, suite, workspace, markdown, and UI primitives
  hooks/       useApi, useRunStream, useDebouncedValue
  lib/         Mode flag, formatting, URL helpers
  mocks/       MSW worker, handlers, fixtures, in-memory store
  pages/       One component per route
  styles/      Tokens plus base, layout, component, and page CSS
  test/        Vitest setup and render helper
```

## Known gaps

- Mock actions are in memory only. They never write to the Go backend or disk,
  and a page reload resets the workspace, logs, configuration, and runs.
- The 10 MB upload limit is enforced in the browser only; the backend must
  enforce its own limit.
- Long lists are not virtualized (workspace files, logs, and JSON arrays).
- Automated tests are component-level with Vitest, jsdom, and MSW. There is no
  E2E, browser, or visual-regression suite.
- Streamed `step_result` events do not include request or response payloads;
  the full payloads appear in the detailed report after a run ends.
- The main bundle is roughly 495 kB raw (~151 kB gzip) with no route-level code
  splitting. Split routes if first-load size becomes a concern.

## Verification

```bash
npm run test:run
npm run typecheck
npm run build
npm run build:real
```

Both builds must succeed. Real-mode checks require `spectest ui` on port 4173.
