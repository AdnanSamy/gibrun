# gibrun

Lightweight personal Postman replacement. Project name is always lowercase `gibrun`.

## Stack and layout

- Go 1.22+, standard library backend, `html/template`, embedded HTMX and handwritten JavaScript/CSS.
- `cmd/gibrun`: startup, flags, legacy data migration.
- `internal/client`: outbound HTTP, timeout/cancellation, response limits/metrics.
- `internal/postman`: collection/environment import.
- `internal/store`: flat records, local JSON persistence.
- `internal/web`: handlers, templates, organization, variables, response rendering.
- `internal/web/static`: application JavaScript and Node-based checks; vendored `htmx.min.js`.
- `ROADMAP.md`: feature status; update when behavior changes.

## Commands

Run from this directory:

```sh
go run ./cmd/gibrun
go test ./...
go vet ./...
node --test internal/web/static/*_test.cjs
git diff --check
```

For VM access, use `-listen VM_IP:8080`; default binding is loopback. Open the exact printed startup URL. Never put real startup tokens or credentials in documentation/tests.

## Coding rules

- Minimum working diff; standard library/native browser features first. No new dependency without a demonstrated need.
- Plain structs/functions; no OOP hierarchies, speculative abstractions, or frameworks.
- Flat slices and ID/index references; avoid pointer trees and unnecessary heap indirection. Return small values directly. Measure before optimizing.
- Explicit data flow: pass state and required inputs to helpers. Keep shared mutable closure state minimal; event adapters may capture owned state.
- Clear indentation, one statement per line. Avoid compressed inline logic. Do not hand-edit vendored HTMX.
- Preserve existing work. Investigate unfamiliar changes before editing; do not reset, delete, or overwrite them.
- Leave a runnable regression check for nontrivial behavior. Do not weaken tests merely to accommodate refactoring.

## Behavior and security

- Send never saves. Preserve unsaved drafts, selected files, tab state, and cancellation behavior during UI changes.
- Browser storage holds only saved request IDs/order/selection, never bodies, credentials, responses, or upload bytes.
- Render untrusted data through escaped templates or DOM text nodes, never untrusted `innerHTML`.
- Validate IDs, destinations, forms, auth settings, and limits at boundaries. Preserve session-token, Host, and Origin protections.
- Preserve disabled headers/query/variables on save; exclude disabled fields from execution.
- Unsupported imports/auth/body modes must warn or block, never silently change request semantics.
- Never read imported upload paths from the server filesystem; require explicit file selection.
- Cancellation cannot undo remote side effects. Avoid claims otherwise.
- Credentials are stored locally in plaintext. Non-loopback HTTP is unencrypted; document these limits.

## Persistence and migration

- Data uses `os.UserConfigDir()` plus `gibrun/data.json`; explicit `-data` overrides the location.
- Windows: `%AppData%\\gibrun\\data.json`; Linux: `$XDG_CONFIG_HOME/gibrun/data.json` or `~/.config/gibrun/data.json`; macOS: `~/Library/Application Support/gibrun/data.json`.
- Default-path migration preserves legacy `solarclient` data and never overwrites existing destination data. Explicit `-data` bypasses automatic migration.
- Keep saves/migrations failure-safe and retries idempotent. Never delete the old project/data without explicit permission.
- Windows ACL/runtime behavior requires Windows verification; cross-compilation alone is not runtime proof.

## Verification notes

- At the rename checkpoint, `TestTLSAndCancellation` and `TestSendTimeoutAndLegacy` failed in both the original and migrated projects. Reproduce and investigate current failures; do not assume they remain baseline failures or hide them.
- Node/Go checks do not prove visual or accessibility behavior. Report actual browser verification separately.
- Do not commit unless requested. Keep migration/refactor changes distinct from unrelated feature work.
