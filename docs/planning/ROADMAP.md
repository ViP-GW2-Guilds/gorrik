# Gorrik — Roadmap

## Active Backlog

### Agent: Re-import Broken Encounter Logs After Parser Fix
The parser's result detection was wrong for four raid encounters. The code has been fixed
(commit on `main`), but the ~2,570 already-imported records have stale results in the DB:

Confirmed kill counts from arcdps Logs Manager (ground truth for verifying fixes):

| Encounter | DB logs | True kills | Notes |
|---|---|---|---|
| Soulless Horror | 1,146 | 174 | gadget fix — verify after re-import |
| Deimos | 720 | 237 | gadget fix — verify after re-import |
| Xera | 290 | 141 | phase-1 fix — high confidence |
| Kaineng Overlook | 145 | 55 | gadget fix — verify after re-import |
| Xunlai Jade Junkyard | 141 | 55 | needs custom detection (team transformation at 50%) |
| Guardian's Glade | 93 | 1 | gadget fix — verify after re-import |
| Harvest Temple | 88 | 34 | currently `resultUnknown`; kills exist but need custom detection |
| Kanaxai | 27 | 7 | gadget fix — verify after re-import |
| Eparch | 19 | 9 | gadget fix — verify after re-import |

**Statues of Grenth** (502 DB logs): arcdps Logs Manager records these as *three separate
encounters* — Broken King, Eater of Souls, Statue of Darkness. Our current definition treats
them as one encounter with four species as triggers, which is wrong. Needs to be split into
three encounter entries with correct individual trigger/species IDs per sub-boss. Counts
unknown until split.

**Ai, Keeper of the Peak** (58 DB logs): arcdps Logs Manager records three variants —
Elemental, Dark, Both Phases. Our trigger 23254 may only match one variant. Currently
`resultUnknown` which is still correct; kill recovery requires splitting the encounter
definition and implementing phase-based detection.

**Xunlai Jade Junkyard** and **Harvest Temple** need custom detection logic —
re-importing won't fix them. Both need their own implementation effort.

**What was fixed:**
- Xera: was requiring phase 1 form (16246) to die; it teleports away. Now only requires
  phase 2 (16286) to die.
- Deimos / Soulless Horror / Statues of Grenth: their kill targets are gadget-type agents
  in the EVTC format (`prof >> 16 == 0xFFFF`). `isNPC()` was excluding them from address
  collection, so ChangeDead events were never matched. Now both NPCs and gadgets are
  collected for `mainBossAddrs`.

**Note:** Soulless Horror and Statues of Grenth may still show 0 kills after re-import if
the gadget fix doesn't fully cover them — in that case further investigation with kill logs
is needed. Deimos and Xera fixes are higher confidence.

**How to re-import:**
1. Build a new Windows binary from the current `main` branch and transfer to the Windows
   machine (see "Agent: Run as Windows Service" for cross-compile command).
2. Connect to the Neon DB and delete the stale records:
   ```sql
   DELETE FROM logs
   WHERE encounter_name IN ('Deimos', 'Xera', 'Soulless Horror', 'Statues of Grenth');
   ```
3. On the Windows machine, run `gorrik import` against the full log directory.
   - R2 uploads are skipped (HeadObject check; files already uploaded).
   - Parser re-runs on every file; only the deleted encounters get re-inserted.
   - This is safe to interrupt and re-run.

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

**Before running backfill-dps:** arcdps Log Manager already has dps.report URLs stored
for logs previously uploaded through it. Mining those first avoids re-uploading thousands
of files unnecessarily. Both systems key off the same base filename, so matching is trivial.

- arcdps Log Manager stores its data in `%APPDATA%\arcdps Log Manager\LogDataCache.json`
- Format: JSON, `Version: 2`, top-level object `LogsByFilename` keyed by full Windows path
- dps.report URL lives at `LogsByFilename[path].DpsReportEIUpload.Url` (null if not uploaded)
- 938 logs already have URLs as of the data export
- Some logs appear under two keys (OneDrive path + F:\ path) with identical URLs —
  deduplicate by base filename when matching
- Matching strategy: `filepath.Base(key)` → match against `logs.filename` in Gorrik DB
- Add a `gorrik import-dps-urls --cache <path>` command that parses the cache file,
  matches by base filename, and PATCHes any found URLs via the API without touching dps.report
- Run this first, then `backfill-dps` for anything still missing
- `Settings.json` in the same directory contains the user's `DpsReportUserToken` —
  could optionally read it from there rather than requiring manual config

#### Web UI
- **"View" column** (far right): shows an external-link icon when `dps_report_url` is
  set; opens the URL in a new tab
- No upload button, no checkboxes — uploads are agent-driven

### Web: Encounters Page UI Polish
Small display improvements to the encounter stats table:

- **Kills column**: always render the kill percentage on its own line below the `kills / total`
  fraction. Currently the percentage wraps or not depending on column width, which is
  inconsistent across encounters.
- **Fastest column**: show the date of the fastest kill on a new line below the time
  (same treatment as the kill percentage — secondary info, visually subordinate).
  The current query uses `MIN(duration_ms)` which discards which log that came from.
  Fix: use `DISTINCT ON (encounter_name) ... ORDER BY encounter_name, duration_ms ASC`
  on the success-only rows to retrieve the full row including `logged_at`, then join
  that back to the main aggregation.

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
