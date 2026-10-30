<div align="center">

<img src="assets/banner_black.png#gh-light-mode-only" alt="SpecTest" width="400">
<img src="assets/banner_white.png#gh-dark-mode-only" alt="SpecTest" width="400">

### LLM-based API Testing Engine for Multi-Platform Workflows

*Automate end-to-end API testing powered by cost-optimized LLMs deployed across multiple platforms.*

[![License](https://img.shields.io/badge/license-Apache--2.0-6d5bff?style=flat-square)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square)](go.mod)
[![Docs](https://img.shields.io/badge/docs-live-6d5bff?style=flat-square)](https://nguyentin05.github.io/spectest/)
[![CI](https://img.shields.io/badge/CI-Passing-brightgreen?style=flat-square&logo=githubactions)](https://github.com/nguyentin05/spectest/actions/workflows/ci.yml)
[![Release](https://img.shields.io/badge/Release-v1.0.0-blue?style=flat-square&logo=git)](https://github.com/nguyentin05/spectest/releases)
[![Issues](https://img.shields.io/github/issues/nguyentin05/spectest?style=flat-square&color=orange)](https://github.com/nguyentin05/spectest/issues)
[![Discussions](https://img.shields.io/badge/Community-Discussions-purple?style=flat-square&logo=github)](https://github.com/nguyentin05/spectest/discussions)

</div>

> **TL;DR:** **SpecTest** is an open-source, cost-efficient AI test engineering engine that turns API specs (OpenAPI specifications + PRD) into executable test suites, catches real backend bugs, and automatically updates tests when your API changes. By offloading deterministic work to Go (**Token-Optimized First**) and supporting both Cloud LLMs and local models, it cuts AI costs by up to 90% and runs anywhere from a single binary: **Local CLI & Web UI, CI/CD Pipelines, On-Premise Servers, and AI Agents (MCP)**.

---

## What is SpecTest?

SpecTest is an open-source API test engineering engine that automates the E2E testing workflow. It converts OpenAPI specs and PRD documents into executable test scenarios, generates test data, executes deep API checks, diagnoses failures, and updates tests when APIs change. Packaged as a single Go binary centered around a two-file contract (`spectest.yaml` and `spectest.log.yaml`), the same core engine runs across local CLI, built-in Web UI, CI/CD pipelines, and Agentic AI workflows.

To keep AI costs low and test runs predictable, SpecTest offloads all deterministic work to Go. Spec compression, auth-token wiring, boundary and security fuzzing, assertion evaluation, and API version diffing run natively with zero LLM tokens. LLMs (Cloud APIs or self-hosted local models) are called only where semantic reasoning matters: designing business test cases, explaining failed runs, and repairing steps affected by API changes.

---

## Core Capabilities

### Test Scenarios Generation
- **Token-optimized preprocessing:** Normalizes specifications, isolates per-operation schema snippets, and strips redundant metadata to focus context on core logic while minimizing prompt tokens.
- **Streamlined generation pipeline:** Six-stage architecture (Facts → Contract Reconciliation → Seed/Setup → Scenarios → Analysis → Persist) ensuring resilient suite generation.
- **Context window chunking:** Splits large specifications into focused domain segments to keep prompts compact and improve LLM reasoning.
- **Hybrid test data:** Combines LLM business scenarios with deterministic boundary and security payloads generated natively with zero token cost.

### Test Execution & Failure Diagnosis
- **Automated environment lifecycle:** Boots services, waits for health checks, executes scenarios, and cleanly tears down target environments (live endpoints or Docker Compose).
- **Setup DAG dependency resolution:** Resolves setup steps as a directed acyclic graph, isolating scenario runtime scopes and extracting response variables (`save: token: body.access_token`).
- **Multi-tier verification:** Validates responses with HTTP status codes, latency thresholds, schema structure, security constraints, and business logic assertions.
- **AI defect triage:** Optional root-cause diagnosis inspecting failed test cases, explaining underlying faults, and suggesting fixes without leaking secrets.

### Spec-Drift Detection & Incremental Updates
- **Deterministic spec diffing:** Compares API versions natively to detect breaking changes, modified fields, and new endpoints with zero LLM tokens.
- **Targeted test patching:** Isolates only the scenarios and setups affected by API changes, prompting the LLM to repair them while preserving existing passing tests.
- **User customization preservation:** Automatically preserves locked scenarios (`locked: true`) and custom user-authored skip reasons.
- **Cryptographic audit history:** Tracks test updates as append-only entries in `spectest.log.yaml` with SHA-256 hash integrity validation.

---

## Supported Platforms

| Platform | Integration | Description |
| :--- | :--- | :--- |
| **Local Development** | CLI & Web UI | Native single binary for Linux, macOS, and Windows with a local interactive Web UI dashboard. |
| **Cloud CI/CD & Deploy** | GitHub Actions, GitLab CI, Jenkins, Docker | Official setup action, pipeline templates, and multi-arch container images (`ghcr.io`). |
| **On-Premise Server** | Webhook Daemon & Docker | Self-hosted test runner for private enterprise networks triggered by Git webhooks. |
| **AI Coding Agents** | MCP Server & Agent Skills | Native Model Context Protocol (MCP) server over stdio for Claude Code, Cursor, Cline, and Windsurf. |
| **LLM Providers** | Cloud APIs & Local Models | Pluggable support for Google Gemini, OpenAI, Anthropic, and self-hosted Ollama. |

---

## Documentation

- **[Quick Start Guide](docs/quickstart.md)**: Installation, LLM provider setup, and running your first test suite.
- **[System Architecture](docs/architecture.md)**: Clean Architecture layers, data models, and core execution flows.
- **[CI/CD Deployment Guide](deploy/README.md)**: Production-ready templates for GitHub Actions, GitLab CI, and Jenkins.
- **[Contributing Guide](CONTRIBUTING.md)**: Development setup, coding standards, and pull request workflows.
- **[Security Policy](SECURITY.md)**: Credential handling, redaction rules, and vulnerability reporting.

---

## Quickstart

### Prerequisites
- **Go 1.25+** (or download prebuilt binaries from [Releases](https://github.com/nguyentin05/spectest/releases))
- An OpenAPI 3.x document (`openapi.yaml` or `openapi.json`)
- *(Optional)* Product Requirement Document (PRD) or business requirements in Markdown (`requirements.md`)
- *(Optional)* An API key for an LLM provider (Google Gemini, OpenAI, Anthropic) or a running local instance of [Ollama](https://ollama.ai)

---

### 1. Local Development (CLI & Web UI)

#### Installation
```bash
# Install via cURL (Linux & macOS)
curl -fsSL https://raw.githubusercontent.com/nguyentin05/spectest/main/install.sh | sh

# Or install via Go
go install github.com/nguyentin05/spectest/cmd/spectest@latest

# Or download prebuilt release binaries from GitHub Releases:
# https://github.com/nguyentin05/spectest/releases/latest
```

#### Configure LLM Provider
```bash
# Initialize LLM provider (Google Gemini, OpenAI, Anthropic, or Ollama)
spectest config init --provider gemini --model gemini-2.5-flash

# Securely set API key (input is masked from terminal echo)
spectest config set api-key

# View active configuration (credentials are masked)
spectest config list
```

#### Generate a Test Suite
```bash
# Bootstrap a new suite from OpenAPI specification
spectest generate --mode init -a auth=openapi.yaml -o ./tests

# Bootstrap with multiple microservices and business requirements
spectest generate --mode init -a auth=auth-api.yaml -a billing=billing-api.yaml -r requirements.md -o ./tests

# Incrementally sync test suite with updated OpenAPI spec (dry-run preview)
spectest generate --mode update -a auth=openapi.yaml -o ./tests --dry-run

# Apply incremental update to existing suite
spectest generate --mode update -a auth=openapi.yaml -o ./tests
```

#### Run Tests
```bash
# Execute suite against a live endpoint
spectest run -f ./tests/spectest.yaml -t http://localhost:8080

# Multi-target microservices
spectest run -f ./tests/spectest.yaml -t auth=http://localhost:8001 -t billing=http://localhost:8002

# Run with automated Docker Compose lifecycle and AI failure triage
spectest run -c docker-compose.test.yml -f ./tests/spectest.yaml -t http://localhost:8080 --triage

# Run targeted regression (only scenarios affected by the latest update)
spectest run -c docker-compose.test.yml -f ./tests/spectest.yaml -t http://localhost:8080 --only-changed

# Export SARIF report for GitHub Code Scanning
spectest run -f ./tests/spectest.yaml -t http://localhost:8080 --sarif reports/spectest.sarif
```

#### Launch Built-in Web UI Dashboard
```bash
spectest ui
```
Opens the local interactive React dashboard at `http://127.0.0.1:8088`.

---

### 2. CI/CD & Deployment (GitHub Actions, GitLab CI, Jenkins, Docker)

Ready-to-use pipeline templates are maintained in the **[`deploy/`](deploy/)** directory:

#### GitHub Actions
Use the official composite action (`nguyentin05/spectest@v1`) with automatic multi-platform binary installation, SHA-256 verification, and tool caching:

```yaml
# .github/workflows/spectest.yml
name: API Regression Quality Gate

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  test:
    name: Run SpecTest Suite
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup SpecTest CLI
        uses: nguyentin05/spectest@v1
        with:
          version: "latest"

      - name: Execute SpecTest Suite
        run: |
          spectest run \
            -f tests/spectest.yaml \
            -t http://localhost:8080 \
            --sarif reports/spectest.sarif
```

#### GitLab CI, Jenkins & Docker
- **GitLab CI:** Use [`deploy/gitlab-ci/.gitlab-ci.yml`](deploy/gitlab-ci/.gitlab-ci.yml) to install the binary or use `ghcr.io/nguyentin05/spectest:latest`.
- **Jenkins:** Use [`deploy/jenkins/Jenkinsfile`](deploy/jenkins/Jenkinsfile) with automatic CPU architecture detection (`amd64`/`arm64`).
- **Docker:** Run directly using the multi-arch container image:
  ```bash
  docker run --rm -v "$PWD:/workspace" ghcr.io/nguyentin05/spectest:latest run -f /workspace/spectest.yaml -t http://host.docker.internal:8080
  ```

For detailed setup instructions, secret management, and platform guides, see **[`deploy/README.md`](deploy/README.md)**.

---

### 3. AI Coding Agents (MCP Server & Agent Skills)

SpecTest provides a native **Model Context Protocol (MCP)** server over standard I/O (stdio) for AI coding assistants (Claude Code, Cursor, Cline, Windsurf).

#### Launch MCP Server
```bash
spectest mcp serve --root .
```

#### Configure in Claude Desktop or Cursor
Add SpecTest to your MCP settings (`claude_desktop_config.json` or `.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "spectest": {
      "command": "spectest",
      "args": ["mcp", "serve", "--root", "."]
    }
  }
}
```

#### Available MCP Tools
- `spectest_status`: Inspect workspace readiness and LLM capabilities.
- `inspect_suite`: Read scenarios, setup blocks, and verify cryptographic hash integrity.
- `validate_suite`: Validate suite structure, DAG dependencies, and syntax.
- `plan_update`: Deterministically preview spec diffs and update impact without mutating disk or consuming tokens.
- `start_update`: Incrementally patch scenarios and setups based on OpenAPI changes.
- `start_run`: Execute full test runs or targeted regressions (`mode: "only_changed"`).
- `read_report`: Retrieve redacted execution reports.

#### Agent Skill

Install it into your coding agent (Claude Code, Cursor, Codex, OpenCode, and other agents supported by the [`skills` CLI](https://github.com/vercel-labs/skills)):

```bash
# Install for the current project
npx skills add nguyentin05/spectest

# Install globally, available to every project on this machine
npx skills add nguyentin05/spectest -g
```

---

## Community & Bug Reports

We actively track bugs, feature requests, and community feedback:

- **Report a Bug or Suggest a Feature:** Found an issue or have an enhancement request? Please [open an issue on GitHub](https://github.com/nguyentin05/spectest/issues/new/choose) using our issue templates.
- **Discussions & Q&A:** Join the conversation, share test suites, and ask questions on [GitHub Discussions](https://github.com/nguyentin05/spectest/discussions).
- **Contributing:** Contributions are welcome! Read our [Contributing Guide](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md) to get started.

---

## License

Distributed under the **[Apache License, Version 2.0](LICENSE)**. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for full license text and third-party attributions.
