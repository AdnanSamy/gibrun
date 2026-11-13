# gibrun

Feature status and planned work: [Roadmap](ROADMAP.md).

## License

[MIT](LICENSE). Vendored [htmx](https://htmx.org/) is distributed under the [Zero-Clause BSD license](https://github.com/bigskysoftware/htmx/blob/master/LICENSE).

Local HTTP request editor. Requires Go 1.22+.

```sh
go run ./cmd/gibrun
```

Open the printed one-time URL. Data is stored in the OS configuration directory under `gibrun/data.json`; override with `-data`. The server listens on loopback by default. To reach it from another machine, bind to the VM's specific IP: `go run ./cmd/gibrun -listen 192.168.1.10:8080` (replace the address). Open the exact printed URL; permit the port in the VM firewall if needed.

**Security:** Non-loopback access uses unencrypted HTTP. Anyone who can observe network traffic can steal the session and credentials and execute requests through gibrun. Use only on a trusted network; prefer SSH forwarding otherwise. Credentials and imported variables are stored unencrypted; protect the data file and avoid sharing the startup URL.

## Rename and existing data

`go build ./cmd/gibrun` produces `gibrun` (`gibrun.exe` on Windows). The module, UI and command now use lowercase `gibrun`.

Default storage uses Go's `os.UserConfigDir`: Windows `%AppData%\gibrun\data.json`, Linux `$XDG_CONFIG_HOME/gibrun/data.json` (otherwise `$HOME/.config/gibrun/data.json`), macOS `$HOME/Library/Application Support/gibrun/data.json`.

On default startup only, if the new file is absent, the legacy `solarclient/data.json` under that same configuration root is validated with the existing store loader, copied into a private temporary snapshot, validated again, then published exclusively without overwriting another file. Existing new files always win. The old file remains unchanged. Corruption, permission errors or unsupported hard links stop startup visibly; no partial destination is published, so fix the problem and retry. Any explicit `-data` bypasses migration, even if it names the default path. Missing legacy data starts fresh.

Rollback: stop gibrun, launch the backup with its original data or explicit `-data`. Migration never deletes the old store; subsequent edits in the two stores are independent, not synchronized. Do not run both versions against the same file. To recover newer edits, retain both files and explicitly select the desired store; no reverse overwrite is automatic.

Files retain the existing security model: new directories request `0700`, files `0600` on Unix. Windows mode bits do not enforce equivalent isolation: restrict the configuration directory with Windows ACLs. Windows was cross-compiled only, not runtime-tested; verify `%AppData%`, ACL inheritance, NTFS hard links and retry behavior on Windows before relying on migration. Non-hard-link filesystems require an explicit `-data` path or manual copy after stopping both versions.

Browser sessionStorage reads legacy `solarclient.request-tabs.v1` only when `gibrun.request-tabs.v1` is absent; only validated saved IDs/order/selection are retained, never secrets or request content. Writes use the new key and leave the old key intact. This works only within the same browser tab and origin (including port); a changed port cannot access old browser storage. Cookies now use `gibrun_session` and `gibrun_environment`; old cookies are not accepted or migrated. Open the newly printed startup URL to authenticate, then reselect the environment.

Create collections and requests, import Postman collections or environments, select an environment when sending. Variables use `{{name}}`; environment values override collection values. Responses are limited to 10 MiB. TLS verification can be disabled explicitly per request.

Per-request timeout is saved in seconds (1–300; default 30 for new, imported, and older requests). It covers the outbound HTTP exchange through response body reading. **Cancel** stays enabled during sends, aborts the browser HTMX request, and propagates disconnection cancellation to the Go outbound context. Cancellation retains the current editor, unsaved changes, and selected files; loading controls reset. Cancel requires JavaScript. It stops waiting, not remote work: remote side effects may already have happened. With JavaScript, completed sends/saves retain the editor and selected uploads.

Responses show formatted JSON without rounding numbers, with a keyboard-accessible native raw toggle when formatting differs. Invalid/non-JSON bodies stay raw; identical formatted/raw bodies are rendered only once. Original response bytes remain unchanged in the client; the raw panel is an escaped text view, not a binary download. Duration covers the HTTP request (including redirects) through body reading, excluding formatting/rendering. Size counts actual body bytes read after automatic decompression, not compressed wire bytes.

Body format persists: raw, JSON, URL-encoded, multipart. Form fields use a JSON array (`[{"key":"name","value":"value","enabled":true}]`), preserving duplicates, disabled fields, and multiline values. For files add `"file":"reference"`, leave the fields editor, then select an upload. Postman form imports retain file references and warn to reselect; arbitrary server paths are never opened. Files are not saved; JavaScript retains selections across sends/saves and tab switches. Changing the fields editor warns before clearing selected uploads. Submissions are limited to 8 MiB; multipart parsing spills above 1 MiB to temporary files, cleaned after each request. Multipart Content-Type always uses the generated boundary.

JSON request editing and valid JSON responses have text-node-only syntax highlighting, limited to 200,000 characters (larger documents stay plain). The request overlay leaves native textarea selection/editing intact, synchronizes scrolling, and supports Tab insertion; Shift+Tab exits. Without JavaScript editors remain plain; save new file fields first to render upload inputs. Browser visual/accessibility checks remain pending.

Collections, folders, and requests support rename/move (where applicable) and confirmed deletion through sidebar context menus. Choose **Duplicate saved request** in a request's context menu, then submit the dialog: creates an independent new ID named “(copy)” in the same destination, preserving all saved fields/body/auth/TLS/timeout and file references, never uploaded bytes. Unsaved editor changes are not copied; existing tabs remain untouched. Open the copy from the sidebar. Nested folders are displayed; individual folder collapse is not implemented.

Sidebar search filters saved collection/folder/request names, HTTP methods, and literal URL host/path case-insensitively. Matching folders/collections include descendants; matching requests retain ancestor context. Clear or Escape restores navigation and collection expansion. No matches shows an accessible status. Search requires JavaScript, resets when the sidebar is replaced, and never stores the query. Authentication, bodies, headers, variables, URL userinfo/query/fragment are not indexed; malformed or unresolved-host URLs are omitted. Desktop controls use compact 32px targets and 4px corners; touch/mobile controls retain 44px targets.

Use **New environment** or an environment's **Edit** button in the sidebar; collections have **Edit variables** buttons. Accessible native dialogs edit variables as JSON arrays, preserving multiline values, duplicates, ordering, and disabled entries:

```json
[{"key":"base_url","value":"https://api.example.com","enabled":true}]
```

Set `enabled` to `false` to retain a variable without using it. Last enabled duplicate wins within each scope; enabled environment values override collection values. Environment deletion requires an explicit confirmation checkbox. Summaries show names/counts, never variable values; values are visible in editors. Failed persistence rolls back memory. Selection survives send/save/navigation through a browser-session cookie (shared across tabs); changing selection before navigation requires JavaScript. Open request tabs retain their own environment selection; deleted selections must be changed manually before sending. Other editors refresh navigation/dialogs without replacing request drafts.

Limitations: variable editing is JSON-based, not a row editor. Browser accessibility/workflow verification remains pending. Postman scripts, authentication other than Basic/Bearer/API key, binary/GraphQL bodies, nested variables, custom CA/client certificates are not supported. Unsupported authentication and body modes block sending. Disabled headers/query entries have separate editors; they are saved but never sent. Move a line between enabled and disabled editors to change its state. HTMX is vendored locally.

API-key authentication supports header/query locations, variable substitution in key name/value, and inherited Postman authentication. Authentication replaces all matching configured header values (case-insensitive) or query values (case-sensitive), including URL query parameters. Names must be nonblank; locations must be `header` or `query`. Malformed imported authentication blocks execution until corrected. Credentials are saved in the local data file; query credentials also appear in URLs and may be logged by remote servers/proxies. API-key execution errors omit transport details to prevent credential disclosure; responses from remote servers may still echo credentials.

## Request tabs

With JavaScript, saved request links open separate tabs, deduplicated by saved ID. New request always opens a fresh draft. Tabs retain live editors (body, auth, timeout, environment), selected uploads and response views in browser memory only. Browser-tab-scoped sessionStorage stores only up to 100 saved request IDs, their order and active saved selection; never names, URLs, credentials, bodies, responses or uploaded bytes. Refresh restores saved requests from authenticated same-origin server pages without sending requests; deleted/unavailable IDs are skipped. Disabled/full/corrupt storage falls back gracefully to in-memory tabs. Save updates the active identity/title; Send never saves. Arrow keys/Home/End select tabs; Delete or the labelled close button closes one. Dirty/new tabs and selected uploads require close confirmation. Reload/closing the browser warns for drafts/uploads or in-flight work; browser unload prompts are best-effort. **Refresh loses unsaved edits, draft tabs, responses and upload selections. Save first.** Saved tabs reload their latest server-saved contents, not dirty snapshots; environment selection uses the existing session cookie. A visible tab-strip warning explains this. Active draft selection is not persisted; restoration selects an available saved tab. Storage is session-only, not a durable workspace backup; no cross-window synchronization.

Tab changes and other submissions are blocked during requests; wait or use Cancel during a send. Sidebar operations preserve drafts and uploads. Completed page responses refresh destination/environment options across open tabs without replacing editor content or selections. Removed selections remain visibly marked and block submission until an existing option or explicitly Unfiled/No environment is chosen. Deleted request tabs become unsaved drafts labelled **Save as new request**; saving creates a new ID. The server rejects stale saves carrying a deleted ID rather than recreating it. Request names/content/destinations are snapshots, not synchronized with sidebar rename/move operations. Changes from other browser windows are detected only on the next page response, not pushed live. Failed submissions preserve the editor. Without JavaScript, normal full-page navigation applies. Node DOM-double checks cover reconciliation; actual browser end-to-end/accessibility verification remains pending (no browser/automation dependencies available in the development environment).
