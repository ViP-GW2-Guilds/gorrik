# Gorrik — Roadmap

## Active Backlog

### Agent + Web: Local Operations UI
The Windows terminal is a poor management surface, and the tool Gorrik replaces (arcdps Log
Manager) has a local UI. The committed end-state is a **native desktop app**, built with
**Wails v3** (its tray icon and native menu support are first-class in v3, bolt-on in v2).
Target behaviour:

- launches from an icon, no terminal window
- configuration set through native menus, not by hand-editing `gorrik.toml`
- can be set to run on Windows startup (a registry `Run` key toggled from Settings)
- runs minimised to the system tray; the tray icon opens the full window

The window renders the same React frontend as `web/`. What it shows:

- browses the log history — reads the same Neon DB directly, or via the Vercel API
- exposes the operations as buttons: Sync / Import / Backfill / edit config, with streamed progress
- **Local Storage panel** — the feature that actually justifies replacing ALM: show which local
  logs are already in R2 and indexed, total reclaimable disk (~20 GB), and a prune action guarded
  so it never deletes a log not confirmed in both R2 and the DB. ALM cannot do this because ALM
  *is* the local store.

**Build order.** The valuable, shell-agnostic core comes first and is worth having on its own:
the localhost server, the reused `web/` components, and especially the Local Storage
reconciliation and prune logic. That phase can render in a plain browser tab during development
(`gorrik ui` → open `localhost:PORT`). The Wails v3 shell — window, tray, native menus,
startup toggle, native directory dialogs — is a firm second phase, not optional polish. This
order defers the v3-vs-v2 commitment until v3's release status (beta, hoping for ~Q3 2026, no
official date) is no longer a guess; a single-user tool can tolerate running on a v3 beta, so
pin it to a specific tag and upgrade deliberately. The only throwaway piece is "open the
browser," ~10 lines.

Builds on `gorrik status` / `gorrik sync` (shipped) and the `POST /api/logs/missing` endpoint —
the prune panel extends that with an R2 HeadObject check per local log.

### Agent: Windows Defender false positives — won't sign
Unsigned `gorrik.exe` builds get false-positived by Windows Defender (heuristics flag the
Service Control Manager calls behind `gorrik service` plus the static-Go + `net/http` pattern),
non-deterministically build-to-build.

**Decision (2026-09-02): not worth an annual code-signing cert for a single-user tool.** The
working process, cost-free:

- Distribute as a `.zip` (the download is not a PE, so it is not blocked mid-transfer).
- Publish the SHA-256 with each build; verify with `Get-FileHash` after transfer.
- Keep a Defender **folder exclusion** for wherever `gorrik.exe` lives (`C:\gorrik`) — once
  excluded, future builds dropped there are never scanned.

If distribution ever widens beyond one user, revisit: Azure Trusted Signing (~$10/mo, best CI
integration) or Certum's individual OSS cert (~$120/yr).

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

### `gorrik setup`: plain-prompt wizard (2026-09-02)
Replaced the `charmbracelet/huh` TUI with plain `bufio` line prompts (`golang.org/x/term`
for the no-echo secret fields). Fixes the Windows paste-doubling bug — pasted text now goes
through native terminal line editing, not a bracketed-paste event stream — works over SSH and
in any terminal, and drops the whole charmbracelet dependency tree (~1.4 MB off the binary).
Non-interactive invocation errors cleanly pointing at manual `gorrik.toml` editing.

### Parser: encounter accuracy pass (2026-09-01, PR #8 + #9)
After a full pass against arcdps Log Manager's parser (EVTCAnalytics), the DB is at 17,907 logs
/ 9,308 success / **3 unknown** (all WvW — no win/loss) / 1,562 challenge-mode.

- **All previously-unknown logs identified**: Spirit Race (trigger 47188 = Ethereal Barrier
  gadget, reward 404), Freezie (species 21333, Determined 762), World vs World (trigger 1), Map
  (trigger 2), Lonely Tower alternate trigger 26257, River of Souls (reward 771).
- **Ai, Keeper of the Peak** split into three — Elemental / Dark / Dark and Light — classified by
  skill 61356 presence and Determined (895) event ordering.
- **`resultByReward`** now matches any reward event, not just the last (a second reward follows
  the completion reward for Spirit Race / River of Souls).
- **Health-based challenge-mode detection** was entirely broken: `scMaxHealth` stores the value
  in `dstAgent`, not `value`. Fixed for Aetherblade Hideout, Mursaat Overseer, Samarog, Xunlai
  Jade Junkyard (all now correct). Deimos needed a `MainSpecies` for the check; Kaineng Overlook's
  CM boss is a distinct species (24266) — adding it also fixed CM result detection. Skorvald
  shows 0 CM, believed correct (no 99CM logs in the archive).
- **Harvest Temple** result detection (skill 63896 presence) confirmed against a real wipe.

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
