# HTTP API

All `/v1` responses use `Cache-Control: no-store`. JSON mutations require `Content-Type: application/json`; unknown fields, multiple JSON values, and bodies above 128 KiB are rejected. Authentication is a session cookie, not an API key. Cross-origin unsafe requests are rejected by Go's CrossOriginProtection. No cross-origin API access is enabled.

| Method and path | Behavior |
| --- | --- |
| `GET /v1/healthz` | Process liveness |
| `GET /v1/readyz` | Database ping, 503 on failure |
| `POST /v1/auth/register` | `{name, username, password}`; creates account and session |
| `POST /v1/auth/login` | `{username, password}`; starts a seven-day session |
| `POST /v1/auth/claim` | `{username, password, legacy_key}`; upgrades a course account and retires key |
| `POST /v1/auth/logout` | Revokes current session; clears cookie |
| `GET /v1/users` | Current user's ID and display name; never a credential |
| `GET /v1/notes` | `{notes, total, page, page_size}`; 30 per page |
| `POST /v1/notes` | `{id?, title, note, pinned?}`; returns created note, 201 |
| `PUT /v1/notes/{id}` | Full `{title, note, pinned, archived, trashed, version}` update |
| `GET /v1/export` | Downloads `{format, exported_at, notes}` including archive/trash |

List query parameters: `view=all|pinned|archive|trash`, `q` (max 200 bytes), `page` (positive integer). Search is literal and case-insensitive for ASCII; SQLite's built-in lower does not provide full Unicode case folding. Order is pinned first, then most recently changed, then ID.

Create may include a client-generated UUID. Retrying an identical create with the same ID returns the saved note without making a duplicate; a different payload returns 409.

Note representation: `id`, `title`, `note`, `created_at`, `updated_at`, `pinned`, `archived`, `trashed`, `version`. Titles permit 160 Unicode characters, note bodies 100,000 bytes, and title or body must contain text. Timestamps use UTC RFC3339. Update is scoped by note ID, user ID, and expected version in one SQL statement. Successful updates increment version; stale writes return 409. Another account's note returns 404.

Errors use `{ "error": "Actionable message" }`: 400 invalid input, 401 session/credentials missing, 403 registration closed or cross-origin mutation, 404 missing owned note/route, 409 stale version or unavailable username, 415 wrong content type, 429 authentication throttled, 503 dependency failure. `Retry-After` accompanies 429.

Usernames normalize to lowercase ASCII and permit 3–40 letters, digits, dashes, or underscores. Passwords permit 12–256 UTF-8 bytes at account creation. No password is returned or logged. A fresh random session token goes only into the HttpOnly cookie; the database stores its SHA-256 digest.
