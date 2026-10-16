# Leafmark UX contract

## Business sources
PRODUCT.md owns account permissions, data lifecycle, access, export, registration, retention, and recovery. No billing or legal acceptance flows exist.

## Canonical ownership
| Capability | Canonical owner | Source of truth | Allowed variants | Verification |
| --- | --- | --- | --- | --- |
| Form | static/app.js auth and editor handlers | PRODUCT.md | sign in, register, migrate, edit | browser and server tests |
| Scrollbar | static/app.css global baseline | DESIGN.md | pane geometry | browser computed style |
| Toast | static/app.js announce | this contract | success, error | live status region |
| CRUD | static/app.js saveNote and changeState | PRODUCT.md | stay in editor, return on archive/trash | server_test.go and browser |
| Dialog | static/app.js ask, native dialog | this contract | discard, trash | keyboard and focus |

## Behavior ledger
Sign in/create/migrate → pending with duplicate prevention → workspace; errors remain in the form and focus the related field. All forms disable native validation popups and preserve input. Credentials are masked with show/hide controls.

New note → editor → explicit Save note → remain in editor with saved status. Opening another note, switching view, paging, reloading notes, or signing out while dirty opens the shared discard dialog. Actual page unload uses beforeunload. Failed writes preserve drafts in memory; no note content or credentials are persisted to browser storage. Export draft remains available during a failed or stale session. Reauthentication can be opened in a separate tab.

Trash → named confirmation → reversible move → list; restore → All notes. Archive → reversible move → list. Pins keep the editor open. Search is 300ms debounced, IME-safe, aborts superseded reads, and clears immediately. Query, view, and page deliberately stay out of URL history because they may expose private note text and notebook activity.

Loading has a stable list area. Empty, no-results, offline, session-expired, conflict, and failure states have explicit text. Session expiry preserves the draft. Paginated list uses 30 notes per page with previous/next controls and clamps after mutations.

English UI; dates use the browser's local timezone with explicit full timestamps in title text. Native dialog provides modal focus containment/Escape; app restores trigger focus. Mobile editor back action returns to notes. Shortcuts ignore IME composition. Ctrl/Cmd+S saves; every shortcut has a visible button.

Controls are disabled in static markup until app.js binds their actions. CSP forbids inline JavaScript.
