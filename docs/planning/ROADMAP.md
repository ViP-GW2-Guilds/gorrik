# Gorrik — Roadmap

## Active Backlog

### Agent: Encounter Result Accuracy — re-audit after PR #8
Two `gorrik reparse` passes (2026-08-30/31) plus the parser fixes brought DB success counts to
an exact match with arcdps Logs Manager across every audited encounter. PR #8 then resolved the
remaining gaps — the 88 `Unknown` logs (Spirit Race, Freezie, WvW, Map, Lonely Tower alt
trigger), River of Souls (reward 771), the Ai three-way split (Elemental / Dark / Dark and
Light), and Harvest Temple — and fixed two bugs: reward-event matching and health-based CM
detection (8 encounters had 0 challenge-mode logs among 2,252).

After PR #8 deploys and `gorrik reparse` runs again:
- confirm the 88 previously-unknown logs land in the right encounters
- confirm challenge-mode counts appear for Deimos, Samarog, Mursaat Overseer, Aetherblade
  Hideout, Xunlai Jade Junkyard, Kaineng Overlook, Skorvald, Eparch
- confirm Twisted Castle failure count drops (reward fix)
- spot-check the three Ai variants against ALM

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

### Agent: Sign the Windows Binary
Unsigned `gorrik.exe` builds get false-positived by Windows Defender (heuristics flag the
Service Control Manager calls behind `gorrik service`, plus the usual static-Go + `net/http`
pattern). It is non-deterministic build-to-build and has cost real time — on 2026-08-30 a build
was quarantined repeatedly, downloads blocked, and a folder exclusion was the only way through.

- Get an Authenticode code-signing certificate. An OV cert still gets a SmartScreen reputation
  ramp; an EV cert clears SmartScreen immediately but is more expensive and needs a hardware
  token / cloud HSM.
- Sign in CI (`signtool` / `osslsigncode`) as part of the release build, not by hand.
- Interim mitigations if signing is deferred: ship a `.zip` (the download is not a PE, so it is
  not blocked mid-transfer), publish the SHA-256 with each build, and document the folder
  exclusion. `-trimpath -ldflags="-s -w"` changes the fingerprint but does not reliably help.
- Also submit false positives to Microsoft (`microsoft.com/wdsi/filesubmission`) — slow, and
  only fixes one hash at a time.

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

### Parser: arcdps BuffApply statechange (2026-08-30, PR #5)
arcdps builds from ~2026-05-01 emit buff applications as a dedicated statechange
(`BuffApply = 69`) instead of a normal (statechange 0) event with `buff != 0`. `resultByBuff895`
/ `resultByBuff762` only looked at statechange-0 events, so every Determined-on-boss success
signal was missed on logs from newer arcdps builds — affecting Aetherblade Hideout, Soulless
Horror, Eparch (895) and Guardian's Glade, Kanaxai (762). Root-caused with the
`20260604-231716.zevtc` sample (Mai Trin's kill signal was a lone `BuffApply` at fight end).

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
