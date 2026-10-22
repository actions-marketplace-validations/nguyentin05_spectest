## Description

Please provide a concise description of what this PR introduces, fixes, or refactors.

- **Related Issue:** Fixes #(issue) / Relates to #(issue)
- **Type of Change:**
  - [ ] :sparkles: New feature (`feat`)
  - [ ] :bug: Bug fix (`fix`)
  - [ ] :recycle: Refactoring (`refactor`)
  - [ ] :pencil: Documentation update (`docs`)
  - [ ] :zap: Performance improvement (`perf`)
  - [ ] :white_check_mark: Tests & QA (`test`)
  - [ ] :wrench: Build / CI tooling (`ci` / `build`)

## Architecture & Design Checklist

Please confirm that your changes respect the Clean Architecture boundaries:
- [ ] **Domain Layer**: Does not import external infrastructure packages.
- [ ] **UseCase Layer**: Interacts with infrastructure solely via Domain Port interfaces.
- [ ] **Zero-Token Philosophy**: Re-used deterministic Go algorithms where applicable before invoking LLMs.

## Verification & Testing

Explain how you tested these changes:
1. `make test` pass: [ ] Yes [ ] N/A
2. `make lint` pass: [ ] Yes [ ] N/A
3. Manual test commands run:
   ```bash
   # Paste command here
   ```

## Checklist

- [ ] My code adheres to the [Go Code Guidelines](https://go.dev/doc/effective_go) and project styling.
- [ ] I have added or updated unit tests covering the new functionality.
- [ ] I have updated relevant documentation in `docs/` or `README.md` if necessary.
- [ ] All CI checks and linter passes without warnings.
