# gibrun feature roadmap

## Rename compatibility

- [x] Lowercase `gibrun` UI/module/binary and `cmd/gibrun`; native OS configuration directory.
- [x] Default-only legacy `solarclient` store migration: validated exclusive snapshot, no overwrite/deletion, visible errors, safe retry. Explicit `-data` bypasses it.
- [x] Saved tab ID compatibility fallback; new cookies require the fresh startup URL. See [migration and rollback](README.md#rename-and-existing-data).
- [ ] Windows runtime migration/ACL verification (cross-compilation only); real-browser tab migration checks.

Personal, lightweight Postman replacement for everyday CRUD APIs.

Checked items are implemented, not a claim of full Postman compatibility. Browser workflow verification remains ongoing.

## Phase 1 — Everyday CRUD and migration

- [x] Go backend, Go templates, embedded HTMX; no frontend build pipeline.
- [x] Responsive dark UI with request editor tabs and response panel; desktop resizable split, stacked on narrow or short screens.
- [x] HTTP method, URL, headers, query parameters, raw/JSON payload editing.
- [x] Explicit save destination and saved/unsaved status; sending does not save.
- [x] Collection creation, rename, confirmed cascade deletion.
- [x] Nested folder display, creation, rename, move, confirmed cascade deletion.
- [x] Request save, rename, move, delete; right-click context menus.
- [x] Folder cycle and destination validation.
- [x] Postman v2.0/v2.1 collection and environment import.
- [x] Import counts, unsupported-feature warnings, missing collection-variable review.
- [x] Imported inherited authentication; Basic and Bearer execution.
- [x] Collection/environment variable substitution.
- [x] Disabled headers/query parameters preserved, editable, excluded from execution.
- [x] Response status, headers, raw body.
- [x] Backend timeout, context cancellation, response-size limit.
- [x] Local JSON persistence with atomic replacement.
- [x] Loopback default; explicit VM-IP binding with session/host/origin checks.
- [ ] Verify migration with real sanitized Postman exports.
- [x] Create/edit/delete environments and variables (JSON editors; confirmed deletion).
- [x] Edit collection variables, including disabled entries.
- [x] Preserve selected environment across send/save/navigation (browser session cookie; shared across tabs).
- [x] URL-encoded request bodies (JSON-array field editor).
- [x] Multipart fields and file uploads (8 MiB submission cap; reselection required).
- [x] Postman form fields/file references preserved; file reselection warnings.
- [x] Safe JSON syntax highlighting in request editor and responses; plain fallback above 200,000 characters.
- [x] Formatted JSON response with native raw toggle when formatting differs; invalid/non-JSON stays raw.
- [x] Response duration through body reading and actual body byte count (after automatic decompression).
- [x] Saved per-request timeout (1–300 seconds; legacy default 30) and visible Cancel button; HTMX abort propagates through Go context, retains unsaved editor/files. Remote side effects cannot be undone; browser workflow verification pending.
- [x] API-key authentication: header/query, variables, inherited Postman import; malformed configurations block execution.
- [x] Request draft protection across tab/sidebar operations; close/unload warning, in-flight navigation lock.

## Phase 2 — Faster daily workflow

- [x] Multiple request tabs; saved-ID deduplication, independent editors/uploads/responses, keyboard navigation, close/unload guards. Refresh restores up to 100 saved IDs/order/active saved selection from browser-tab-scoped sessionStorage via authenticated same-origin pages; no secrets/content stored in browser storage. Drafts/unsaved edits/responses/uploads still lost, with visible warning.
- [x] Duplicate saved requests through context menu/dialog; independent ID and field slices, same destination, all saved fields preserved; no uploaded bytes or unsaved changes copied.
- [x] Sidebar collection/folder/request search by name, method, literal URL host/path; case-insensitive ancestor context, matching descendants, Clear/Escape, empty status. Browser-only, no query persistence; auth/body/variables and URL credentials/query/fragment excluded.
- [x] Compact desktop buttons/controls (32px, thin borders, 4px corners); touch/mobile targets remain 44px. Visual/browser accessibility verification pending.
- [ ] Bounded request history and replay, excluding secrets by default.
- [ ] Import/export cURL.
- [ ] gibrun backup/export.
- [ ] Workflow keyboard shortcuts.
- [ ] Request/folder ordering.
- [ ] Collapsible nested folders.

## Phase 3 — Demand-driven additions

Implement only when an actual workflow requires them.

- [ ] OAuth 2.0 token acquisition.
- [ ] Custom CA and client certificates.
- [ ] GraphQL-specific tooling.
- [ ] WebSocket client.
- [ ] Collection runner and response assertions.

## Quality and performance

- [ ] Browser workflow tests and accessibility checks.
- [ ] Import and execution benchmarks.
- Keep flat records, contiguous slices, and ID/index references instead of pointer trees.
- Return small values directly; avoid unnecessary payload copies.
- Prefer Go standard library, native browser features, existing HTMX.
- Optimize measured bottlenecks; no speculative pooling or custom allocators.
- Consider SQLite only when history/search requirements outgrow JSON storage.

## Current limitations and security

- Postman scripts/tests are not executed.
- Unsupported authentication and body modes block execution.
- Nested variable expansion is unsupported.
- Variable editing currently uses JSON, not a row-based editor; disabled entries, ordering, duplicates, and multiline values are preserved.
- Environment cookie initializes new tabs; each open request tab retains its own selection.
- JavaScript preserves request drafts/uploads across sidebar operations; page responses refresh all tabs' collection/environment options. Removed selections visibly block submission until explicitly changed. Deleted requests become labelled unsaved drafts saved only under new IDs; stale server saves are rejected. Request content/name/destination snapshots are not synchronized with sidebar rename/move. No live cross-window synchronization; only saved tab IDs/order/active selection survive refresh (100-tab limit, sessionStorage optional). Deleted/unavailable IDs are skipped; dirty snapshots, draft tabs, responses, selected files and per-tab environment selections do not persist. Unload warnings are best-effort. Reconciliation has Node DOM-double checks; real browser/accessibility checks remain pending (browser tooling unavailable).
- Credentials and imported variable values are stored unencrypted locally.
- Non-loopback access uses unencrypted HTTP; use a trusted network or SSH forwarding.
- Go strings, slices, maps, and HTTP internals still use indirection; zero pointer chasing is not a literal guarantee.

## Out of scope

Cloud sync, accounts, collaboration, plugin infrastructure.

## Recommended next sequence

1. Real-export migration verification and browser form/highlighting checks.
2. Remaining Phase 1 workflow safeguards.
