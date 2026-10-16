# Verification record

Verified locally on October 16, 2026. This records local results, not a GitHub Actions run or a public deployment.

## Automated checks

- `go test -race ./...` passed, including scoped writes/exports, optimistic version conflicts, reversible trash, session expiry/revocation, Secure-cookie configuration, hashed token storage, JSON validation, CSRF rejection, login throttling, readiness failure, legacy migration, and retry-safe creation.
- `go vet ./...`, `go mod verify`, `node --check static/app.js`, production binary build, and `git diff --check` passed.
- `govulncheck ./...` found no reachable vulnerabilities. The module-level warning concerns x/crypto's unimported OpenPGP package; Leafmark imports Argon2, not OpenPGP. The original Chi router was upgraded to v5.
- The frontend skill's strict static audit reported zero findings. DESIGN.md lint reported zero errors/warnings. These checks do not establish full WCAG conformance.
- Docker image built successfully. The smoke script ran with an unprivileged user, read-only root filesystem, dropped capabilities, and a named data volume. Registration, note creation, process restart, login, note persistence, and JSON export passed.
- Compose configuration validated without printing environment secrets. README local links and SVG XML parsed successfully.

## Browser checks

Chromium checks used a disposable local SQLite database, not the existing remote Turso database. Verified account creation/sign-in, note creation/editing, pinning, archive/unarchive, trash/restore, search/no-results/clear, unsaved-navigation confirmation, modal Escape/focus restoration, and saved-note persistence after server restart.

Inspected desktop (1365×900) and phone (390×844) layouts. Long titles wrap; the phone editor exposes Back to notes. Checked the global scrollbar styling and absence of horizontal overflow at the phone width.

Forced a concurrent database update: the browser displayed the 409 conflict, retained the unsaved text, and downloaded that exact draft as Markdown. Stopped the preview server: the editor retained the failed-save draft. Follow-up hardening adds a 20-second browser request deadline and actionable connection text.

`readme-assets/workspace.webp` is a real screenshot of the disposable local notebook, converted losslessly with metadata stripped. It contains no real account credentials or private notes.

## Release boundary

The changes are committed locally. Public deployment, remote repository/directory rename, remote CI execution, existing Turso schema upgrade, scheduled backups, TLS/proxy configuration, restore drill on the target host, distributed abuse controls, and automated account recovery have not been performed. The supported deployment scope is a personal, single-instance app with operator-managed HTTPS and backups. A public multi-user service needs further operational and recovery work.
