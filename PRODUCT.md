# Leafmark product contract

Leafmark is a personal plain-text notebook extended from Boot.dev's Notely CI/CD project. Each account owns its notes. There is no collaboration, public sharing, billing, or organization access.

## Access and data
- Every note read, write, and export is scoped to the authenticated user. Registration is controlled by ALLOW_SIGNUP; turn it off after creating accounts for a private deployment.
- Browser authentication uses a seven-day, revocable HttpOnly/SameSite=Strict cookie; production requires HTTPS and Secure cookies. Passwords use salted Argon2id. No email addresses are collected. There is no automated account recovery: keep credentials in a password manager and export notes regularly.
- Existing course users may claim their notes with their existing key and a new username/password. Claiming invalidates the old key. The previous API-key authentication endpoints are retired.
- Trash and archive are reversible. Nothing automatically purges notes; there is no permanent-delete action in this release. Export includes all notes, including archive and trash. Operators control database retention and backups.
- Concurrent edits use version checks. A stale write returns 409 and preserves the editor; export the draft before reloading.

## Scope
Create and edit title/body, pin, archive, trash, restore, server search and pagination, JSON export, Markdown draft download, keyboard save, responsive interface, logout, and legacy migration. Notes are plain text; HTML and Markdown are never executed.

## Operations
Single instance by default; local SQLite or existing remote libSQL/Turso. Authentication throttling is in memory and deliberately ignores forwarded IP headers: a reverse proxy must enforce its own per-client limits. No claim of high availability, end-to-end encryption, automated recovery, or public deployment is made.
