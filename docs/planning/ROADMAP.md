# Gorrik — Roadmap

## Active Backlog

### Agent: Fix Live Watcher for Nested Log Directories
`gorrik watch` only watches the top-level log directory (`fw.Add(logDir)` in `watcher/watcher.go`).
arcdps saves logs in subdirectories by encounter name and then character name, e.g.
`F:\arcdps_logs\Cairn the Indomitable\Balsamus Aran\20230707-215203.zevtc`. New logs written to
any subdirectory will be silently missed.

- Switch to recursive watching: either call `fw.Add` for every subdirectory at startup and watch
  for new directory creation events, or use `fsnotify`'s recursive watch API if available
- Ensure newly created subdirectories (arcdps creates them on first log for an encounter) are also
  watched

### Agent: Fix `gorrik setup` Paste Doubling on Windows
When running `gorrik setup` in a Windows terminal (particularly `cmd.exe`), pasting values
causes each field to be doubled in the saved `gorrik.toml`. Root cause: some Windows terminals
deliver pasted text both as a bracketed paste event and as individual keystrokes, so the
`charmbracelet/huh` TUI receives the input twice.

- Investigate whether setting a specific Windows console mode in `bubbletea` fixes it
- Fallback option: replace the TUI wizard with plain line-input (`fmt.Scan` / `bufio.Scanner`)
  on Windows, or add a `--no-tui` flag
- Until fixed, workaround is to edit `%APPDATA%\gorrik\gorrik.toml` directly

### Agent: Run as Windows Service
`gorrik watch` currently requires a terminal to stay open. It needs to run as a real Windows
service so it starts automatically at boot and survives logouts.

- Cross-compile the agent for Windows (`GOOS=windows GOARCH=amd64`)
- Install/uninstall as a service (likely via `golang.org/x/sys/windows/svc` or a wrapper like
  `kardianos/service`)
- Service should write logs to a file rather than stdout

### Agent: Bulk Import
Only 10 test logs are in the database. The full archive of ~16,621 logs (~20GB) needs to be
imported via `gorrik import`.

- Verify the `gorrik.toml` API URL is pointed at production (not localhost)
- Run `gorrik import` against the full log directory
- Confirm all logs are indexed in the DB and files uploaded to R2

### dps.report Integration
Upload logs to dps.report and store the returned permalink. Done in the agent (not
the web UI) so uploads happen from the machine that has the files — no web endpoint
to secure, no R2 → Vercel → dps.report hop.

#### Schema
- Add `dps_report_url TEXT` column to the `logs` table (nullable)
- Generate a Drizzle migration
- Add `PATCH /api/logs/[id]` route for the agent to write back the permalink

#### Agent: auto-upload in `gorrik watch`
New logs processed by the watcher are uploaded to dps.report as part of the pipeline:
parse → R2 upload → dps.report upload → API POST (with `dps_report_url` included).
One log at a time, so rate limits are not a concern.

- Add `[behaviour] upload_to_dps_report = true/false` to `gorrik.toml` (default: false
  until the user opts in)
- Add `[behaviour] dps_report_user_token = "..."` for associating uploads with an account
- dps.report API: `POST https://dps.report/uploadContent?json=1&generator=ei` as
  multipart form; parse `permalink` from JSON response

#### Agent: `gorrik backfill-dps` command
New command that backfills `dps_report_url` for logs already in the DB. Designed to
be run manually whenever the user wants to catch up historical logs.

- Walk the local log directory (same directory resolution as `gorrik import`)
- For each file, check via the API whether the log already has a `dps_report_url`
- If not, upload to dps.report and PATCH the URL back via the API
- Rate-limit deliberately (e.g. 1 upload/second) to avoid hammering dps.report
- Safe to interrupt and re-run — already-uploaded logs are skipped

#### Web UI
- **"View" column** (far right): shows an external-link icon when `dps_report_url` is
  set; opens the URL in a new tab
- No upload button, no checkboxes — uploads are agent-driven

### Web: Character Drill-Down Detail
The expanded character row on `/players` shows log count, success rate, and top spec. The
intended richer view:

- Which specific encounters that character attended
- Kill times per encounter

The data volume may make this better suited to a dedicated character detail page or modal
rather than an inline row expansion.

### Web: Tags
The `logs` table already has a `tags text[]` column. Nothing has been built yet:

- Display tags on log rows
- Filter logs by tag in the sidebar or filter bar
- UI for adding/editing/removing tags on a log

---

## Deferred

### Web: dps.report Integration
Trigger a dps.report upload from within the Gorrik UI and store the permalink on the log
record. Lets users jump straight to full combat analysis from the log list.

### GW2 API Integration
Pull guild data, member lists, and character names via the GW2 API. Prerequisite for the
Guilds tab and for enriching player profiles with guild tags.

### Guilds Tab
A view grouping logs and players by guild. Requires GW2 API integration.

### Weekly Clears Tab
Show which encounters were cleared in the current or past weekly reset window. Requires
reset timer logic and encounter scheduling data.

### Multi-User / Sharing
Open the app to guildmates. Currently a personal tool — each instance is a single-user
deployment. Sharing would require auth and per-user data isolation.
