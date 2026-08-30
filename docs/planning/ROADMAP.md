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
   machine:
   ```
   GOOS=windows GOARCH=amd64 ~/go/bin/go1.24.3 build -o gorrik.exe .
   ```
2. Connect to the Neon DB and delete the stale records:
   ```sql
   DELETE FROM logs
   WHERE encounter_name IN ('Deimos', 'Xera', 'Soulless Horror', 'Statues of Grenth');
   ```
3. On the Windows machine, run `gorrik import` against the full log directory.
   - R2 uploads are skipped (HeadObject check; files already uploaded).
   - Parser re-runs on every file; only the deleted encounters get re-inserted.
   - This is safe to interrupt and re-run.

### Agent: Fix `gorrik setup` Paste Doubling on Windows
When running `gorrik setup` in a Windows terminal (particularly `cmd.exe`), pasting values
causes each field to be doubled in the saved `gorrik.toml`. Root cause: some Windows terminals
deliver pasted text both as a bracketed paste event and as individual keystrokes, so the
`charmbracelet/huh` TUI receives the input twice.

- Investigate whether setting a specific Windows console mode in `bubbletea` fixes it
- Fallback option: replace the TUI wizard with plain line-input (`fmt.Scan` / `bufio.Scanner`)
  on Windows, or add a `--no-tui` flag
- Until fixed, workaround is to edit `%APPDATA%\gorrik\gorrik.toml` directly

### Agent: `gorrik status` and `gorrik sync`
The CLI has one verb per operation (`import`, `import-dps-urls`, `backfill-dps`), each with its
own flags. These run infrequently enough that the flags never become second nature, and until
`cobra.NoArgs` was added a bare positional path was silently ignored in favour of the configured
directory. Two additions collapse the common cases:

**`gorrik status`** — also the bare `gorrik` output, replacing the help dump:
- config path and whether it loaded
- resolved log dir: exists, file count, size on disk
- DB: total indexed logs, newest `logged_at`
- "behind by N": local files not yet indexed — needs a new `POST /api/logs/missing` taking a
  filename list and returning the unindexed subset
- dps.report: count of indexed logs still missing a URL
- a suggested next command

**`gorrik sync`** — the catch-up flow as one verb:
- `import` (configured dir) → `import-dps-urls` (if a cache path is set or found at the default
  location) → `backfill-dps`
- flags: `--dry-run`, `--skip-dps`
- config addition: `[arcdps_log_manager] cache_path`, auto-detected from
  `%APPDATA%\arcdps Log Manager\LogDataCache.json`, so `sync` needs no arguments

This is the first step; the Local Operations UI below builds on it.

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
