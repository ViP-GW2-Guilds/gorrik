# Gorrik — Roadmap

## Active Backlog

### Agent: Re-parse Broken Encounter Logs After Parser Fixes
Many encounters had wrong result detection when the bulk of the ~17,900 logs were imported.
The parser has since been fixed for 20+ encounters (`91f2d16`, `074013e`, `dfaf1b0`, `c9d2064`,
`99fafe4`, `1c5c3b4`), but the already-imported records still carry the old results.

**Mechanism (shipped):** `gorrik reparse` re-parses local files and updates matching records
in place — no delete, no re-upload, `dps_report_url` / favourites / tags preserved. Run
`gorrik reparse --dir <path> --dry-run` first, then without `--dry-run`, once per local
directory (`F:\arcdps_logs` and the live `arcdps.cbtlogs`).

**Still open — encounters that need parser work, not just a re-parse:**
- **Statues of Grenth**: split into Broken King / Eater of Souls / Statue of Darkness landed
  (`dfaf1b0`); verify counts after re-parse.
- **Ai, Keeper of the Peak** (~62 logs, `resultUnknown`): needs the encounter split into
  Elemental / Dark / Both Phases and phase-based detection.
- **Xunlai Jade Junkyard**, **Harvest Temple**: have `resultByTeamChange` / skill-based
  detection now; verify against ground truth after re-parse and add custom logic if still off.

Confirmed kill counts from arcdps Logs Manager (ground truth — may be stale, re-audit after
the re-parse run):

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

**After the re-parse run:** re-audit every encounter against arcdps Logs Manager, update the
table above, and open follow-up items for any encounter still wrong.

### Agent: Fix `gorrik setup` Paste Doubling on Windows
When running `gorrik setup` in a Windows terminal (particularly `cmd.exe`), pasting values
causes each field to be doubled in the saved `gorrik.toml`. Root cause: some Windows terminals
deliver pasted text both as a bracketed paste event and as individual keystrokes, so the
`charmbracelet/huh` TUI receives the input twice.

- Investigate whether setting a specific Windows console mode in `bubbletea` fixes it
- Fallback option: replace the TUI wizard with plain line-input (`fmt.Scan` / `bufio.Scanner`)
  on Windows, or add a `--no-tui` flag
- Until fixed, workaround is to edit `%APPDATA%\gorrik\gorrik.toml` directly

### Agent + Web: Local Operations UI
The Windows terminal is a poor management surface, and the tool Gorrik replaces (arcdps Log
Manager) has a local UI. A `gorrik ui` command would serve a localhost web app that:
- browses the log history — reads the same Neon DB directly, or via the Vercel API
- exposes the operations as buttons: Sync / Import / Backfill / edit config, with streamed progress
- **Local Storage panel** — the feature that actually justifies replacing ALM: show which local
  logs are already in R2 and indexed, total reclaimable disk (~20 GB), and a prune action guarded
  so it never deletes a log not confirmed in both R2 and the DB. ALM cannot do this because ALM
  *is* the local store.

The frontend can reuse `web/` React components. Later polish: wrap it in a Wails shell for a
native window and native directory dialogs — a ~15 MB binary over the OS webview, not Electron.

A menu-driven TUI (`bubbletea`) was considered as a lighter alternative but is a strict subset of
this; skip it if the local UI is built.

Builds on `gorrik status` / `gorrik sync` (shipped) and the `POST /api/logs/missing` endpoint —
the prune panel extends that with an R2 HeadObject check per local log.

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

### Wingman Upload Integration
Gw2Wingman (gw2wingman.nl) aggregates instanced-PvE logs for build and rotation analytics; it
has its own local uploader that watches the log folder. Once Gorrik prunes local logs that
uploader can no longer reach the ones it missed, so Gorrik — the one process that sees every log
exactly once — is the right place to fan out to Wingman alongside R2 and dps.report.

- `upload_to_wingman = true` in `[behaviour]`, parallel to `upload_to_dps_report`, off by default
- filter the fan-out to instanced content using the parser's existing `category` / `subcategory`
- `gorrik backfill-wingman`: Wingman can ingest a dps.report permalink, and those URLs are
  already collected — backfill is a POST per URL, no file uploads, and works after local logs
  are gone
- confirm Wingman's API (ingest-by-dps.report-link endpoint, the already-known check) against
  their docs or by inspecting the installed uploader's traffic
- opt-in only: uploading shares squadmate account names with a public aggregator

Depends on the local-log/R2 reconciliation from the Local Operations UI work.

---

## Shipped

### `gorrik reparse` (2026-08-30, PR #3)
- Re-parses local files and updates matching DB records in place (result, mode, duration,
  encounter). Preserves `dps_report_url`, favourites, tags; skips logs not in the DB; nothing
  uploaded or deleted. `--dry-run` reports how many results would change.
- API: `PATCH /api/logs/reparse` (Bearer-authed), with a `dry_run` flag.

### `gorrik status` and `gorrik sync` (2026-08-29, PR #1)
- `gorrik status` (also bare `gorrik`): config path, resolved log dir + file count + size,
  Log Manager cache location, DB totals + newest `logged_at`, "behind by N" local logs not yet
  indexed, and a suggested next command.
- `gorrik sync`: `import` → `import-dps-urls` → `backfill-dps` as one verb, with `--dry-run` and
  `--skip-dps`.
- `[arcdps_log_manager] cache_path` config, auto-detected at
  `%LOCALAPPDATA%\ArcdpsLogManager\LogDataCache.json`.
- API: `GET /api/logs/stats`, `POST /api/logs/missing` (both Bearer-authed).
