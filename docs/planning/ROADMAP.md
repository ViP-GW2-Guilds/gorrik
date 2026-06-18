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
