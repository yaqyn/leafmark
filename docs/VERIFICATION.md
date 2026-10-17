# Verification record

Verified locally on October 16, 2026. This records local results, not a GitHub Actions run or a public deployment.

## Automated checks

- `go test -race ./...` passed, including scoped writes/exports, optimistic version conflicts, reversible trash, session expiry/revocation, Secure-cookie configuration, hashed token storage, JSON validation, CSRF rejection, login throttling, readiness failure, legacy migration, and retry-safe creation.
- `go vet ./...`, `go mod verify`, `node --check static/app.js`, production binary build, and `git diff --check` passed.
- `govulncheck ./...` found no reachable vulnerabilities. The module-level warning concerns x/crypto's unimported OpenPGP package; Leafmark imports Argon2, not OpenPGP. The original Chi router was upgraded to v5.
- The frontend skill's strict static audit reported zero findings. DESIGN.md lint reported zero errors/warnings. These checks do not establish full WCAG conformance.
- Docker image built successfully. The smoke script ran with an unprivileged user, read-only root filesystem, dropped capabilities, and a named data volume. Registration, note creation, process restart, login, note persistence, and JSON export passed.
- Compose configuration validated without printing environment secrets. README local links and SVG XML parsed successfully.

## Toolchain scan reconciliation — October 17, 2026

The published October 16 CI run failed at its vulnerability check; its container step was skipped. Its Go setup read `go 1.27.1` from `go.mod`, while the Dockerfile already used Go 1.27.2. The earlier local results above do not establish that CI passed.

An isolated comparison of the same published source with govulncheck v1.9.0 reported 10 reachable standard-library vulnerabilities under Go 1.27.1 (exit 3) and zero reachable vulnerabilities under Go 1.27.2 (exit 0). The latter still reported one required-module vulnerability without an apparent call from this code; this is not a claim that every dependency is vulnerability-free. The module directive and installation requirement now use Go 1.27.2 to align CI and native builds with the existing container builder. No vulnerability checks were disabled or suppressed.

This comparison is Linux-local evidence. It does not establish that a new remote CI or container run has completed; consult the current commit's Actions results for those outcomes.

## Browser checks

Chromium checks used a disposable local SQLite database, not the existing remote Turso database. Verified account creation/sign-in, note creation/editing, pinning, archive/unarchive, trash/restore, search/no-results/clear, unsaved-navigation confirmation, modal Escape/focus restoration, and saved-note persistence after server restart.

Inspected desktop (1365×900) and phone (390×844) layouts. Long titles wrap; the phone editor exposes Back to notes. Checked the global scrollbar styling and absence of horizontal overflow at the phone width.

Forced a concurrent database update: the browser displayed the 409 conflict, retained the unsaved text, and downloaded that exact draft as Markdown. Stopped the preview server: the editor retained the failed-save draft. Follow-up hardening adds a 20-second browser request deadline and actionable connection text.

`readme-assets/workspace.webp` is a real screenshot of the disposable local notebook, converted losslessly with metadata stripped. It contains no real account credentials or private notes.

## Release boundary

The repository and local checkout use the Leafmark name. This record covers local verification; consult GitHub Actions for the published commit’s CI result. Public deployment, existing Turso schema upgrade, scheduled backups, TLS/proxy configuration, restore drill on the target host, distributed abuse controls, and automated account recovery have not been performed. The supported deployment scope is a personal, single-instance app with operator-managed HTTPS and backups. A public multi-user service needs further operational and recovery work.
