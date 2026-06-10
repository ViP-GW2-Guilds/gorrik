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

### Web: dps.report Upload Integration
Upload selected logs to dps.report from within the Gorrik UI, store the returned permalink,
and display it as a link on the log row.

#### Schema
- Add `dps_report_url TEXT` column to the `logs` table (nullable)
- Generate a Drizzle migration

#### API
- Add R2 credentials to Vercel env vars (`R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`,
  `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`) so the server can fetch raw files
- Add `POST /api/logs/[id]/dps-report` route:
  - Fetch the raw `.zevtc` from R2 using stored `file_url`
  - POST to `https://dps.report/uploadContent?json=1&generator=ei` as multipart form
  - Parse the returned permalink from the JSON response
  - Write permalink back to `logs.dps_report_url`
  - Set `maxDuration = 60` on this route (download + upload can take 15–30s)
- Optionally support a `dps_report_user_token` in config/env for associating uploads
  with a dps.report account

#### UI
- **Checkbox column** (far left, no heading): visible only on rows where
  `dps_report_url` is null; clicking a row's checkbox adds/removes it from the
  selection set (component state — `Set<string>` of log IDs). No "select all".
- **"View" column** (far right): visible only on rows where `dps_report_url` is set;
  shows an external-link icon that opens the URL in a new tab. Does not interact
  with checkbox state.
- **"Upload to dps.report" button**: lives in the filter bar or above the table;
  disabled until at least one checkbox is checked; shows "Upload to dps.report (N)"
  where N is the selected count
- **Upload progress**: button changes to "Uploading N of M…" during the operation;
  uploads sequentially (one at a time) to respect dps.report rate limits; each
  completed upload removes the checkbox and adds the link icon on that row without
  a full page reload
- Checkbox state should be preserved across sidebar filter changes but cleared on
  page navigation

#### Notes
- The virtualized log list already handles 16k+ rows; checkbox state lives in a
  `Set<string>` in the `LogsTable` component, separate from the virtualizer
- No "select all" — too easy to accidentally queue thousands of uploads

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
