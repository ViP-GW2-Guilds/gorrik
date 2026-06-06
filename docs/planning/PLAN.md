# Gorrik — Project Plan

## Context

The user is a full-stack Node/TypeScript/React developer who plays Guild Wars 2 and uses arcdps to
automatically record combat logs. They have 16,621 logs dating back to 12/6/2022, totalling ~20GB
of local storage. The existing desktop tool (arcdps Log Manager) only works with locally-stored
files, and the user cannot maintain a C# codebase.

The goal is to build **Gorrik**: a cloud-native replacement that automatically detects new logs,
parses them for metadata, archives the raw files to cloud storage, and provides a modern web UI
for browsing, filtering, and searching the log history. The user also wants to use this project as
an opportunity to learn Go.

---

## Project Name

**Gorrik** — named after the meticulous asuran scientist from Guild Wars 2, known for obsessively
cataloguing and documenting data.

---

## Tech Stack

| Layer | Technology | Rationale |
|---|---|---|
| EVTC Parser + Local Agent | **Go** | Native int64/uint64 (no BigInt), clean struct-based binary parsing via `encoding/binary`, goroutines for bulk import, single deployable binary, learning opportunity |
| Web App (API + Frontend) | **Next.js (TypeScript)** | API routes + React in one deployment, user's home stack, easy Vercel deployment |
| Database | **Neon (PostgreSQL)** | Free tier with no auto-pausing, standard SQL, sufficient for metadata at this scale |
| File Storage | **Cloudflare R2** | No egress fees, S3-compatible API, ~$0.15/month for 20GB |
| Hosting | **Vercel** | Free Hobby tier, native Next.js support, zero-config deployment |

---

## What We Are NOT Building

- A full EVTC combat parser (DPS numbers, buff uptimes, rotations) — that is dps.report's job
- GW2 API integration for guild data — deferred to a future phase
- Weekly clears tracking — deferred to a future phase
- Multi-user / shared access — personal tool only for now

---

## Architecture

```
arcdps saves .evtc to local folder
        │
        ▼
Gorrik Agent (Go binary, runs on user's PC)
  ├── reads arcdps config to find log directory
  ├── watches directory with fsnotify
  ├── on new file: parses metadata → uploads to R2 → POSTs to API
  └── bulk import mode for existing 16,621 logs
        │
        ├──────────────────────────┐
        ▼                          ▼
Cloudflare R2               Neon (PostgreSQL)
(raw .evtc files)           (parsed metadata)
                                   │
                                   ▼
                         Next.js API (Vercel)
                                   │
                                   ▼
                         React Frontend (Vercel)
                         (browse, filter, search)
```

---

## Component 1: Gorrik Agent (Go)

A single Go binary that runs locally on the user's Windows PC as a background process.

### Responsibilities
- Auto-detect the arcdps log directory by reading `arcdps.ini`
- Watch the directory for new `.evtc`, `.evtc.zip`, and `.zevtc` files using `fsnotify`
- Parse each file for metadata (see Parser section below)
- Upload the raw file to Cloudflare R2
- POST the parsed metadata JSON to the Next.js API
- Optionally delete the local file after confirmed upload (user-configurable, default: keep)

### Three Commands
- `gorrik setup` — interactive first-run wizard; auto-detects arcdps log directory, prompts for
  API URL/key and R2 credentials, writes `gorrik.toml`. Safe to re-run to update config.
- `gorrik watch` — normal operation; runs continuously and processes new logs as they appear
- `gorrik import` — one-time bulk processing of all existing logs in a directory, uses goroutines
  to parallelise uploads (parsing is fast; upload is the bottleneck)

The `setup` command uses `charmbracelet/huh` for the interactive prompt flow — a lightweight TUI
form library that produces clean terminal UI without requiring a GUI framework.

### Config File (`gorrik.toml`)
```toml
[arcdps]
auto_detect = true
# log_dir = "C:\\path\\to\\logs"  # override if auto-detect fails

[api]
url = "https://gorrik.vercel.app/api"
key = "your-api-key"

[storage]
r2_account_id = "..."
r2_access_key_id = "..."
r2_secret_access_key = "..."
r2_bucket = "gorrik-logs"

[behaviour]
delete_after_upload = false
```

### arcdps Config Auto-Detection
arcdps stores its config at:
`%USERPROFILE%\Documents\Guild Wars 2\addons\arcdps\arcdps.ini`

The agent reads this file to extract the configured log path, falling back to the user-supplied
path in `gorrik.toml` if auto-detection fails.

### Go Dependencies
- `github.com/fsnotify/fsnotify` — cross-platform file watching
- `github.com/aws/aws-sdk-go-v2` — S3-compatible R2 uploads
- `github.com/BurntSushi/toml` — config file parsing
- `github.com/charmbracelet/huh` — interactive terminal forms for the setup wizard

---

## Component 2: EVTC Metadata Parser (Go, part of Agent)

A package within the agent binary. Reads the binary EVTC format and extracts metadata only.
Not a full combat parser — no DPS, no buff uptimes, no event replay.

### EVTC Format Overview
Binary format with three sections:
1. **Header** — magic bytes `EVTC`, build date string, encounter ID byte, revision byte
2. **Agent list** — fixed-width structs (agent address, profession, elite spec, name)
3. **Skill list** — skipped for metadata purposes
4. **Combat events** — scanned only for first/last timestamps and result detection

### What the Parser Extracts

```go
type LogMetadata struct {
    EncounterID   uint16
    EncounterName string
    Category      string // "raid", "strike", "fractal"
    Subcategory   string // "Spirit Vale (W1)", "End of Dragons", "Nightmare", etc.
    Result        string // "success", "failure", "unknown"
    Mode          string // "normal", "emboldened", "challenge", "legendary", "quickplay"
    DurationMs    int64
    LoggedAt      time.Time
    Players       []PlayerEntry
}

type PlayerEntry struct {
    AccountName   string // e.g. "username.1234"
    CharacterName string
    Profession    string
    EliteSpec     string
}
```

### Encounter Identification
A lookup table maps encounter IDs to names, categories, and subcategories. Scope is limited to:
- **Raids**: All 8 wings (Spirit Vale W1 through Mount Balrior W8)
- **Strike Missions**: Icebrood Saga, End of Dragons, Secrets of the Obscure, Visions of Eternity
- **Fractals of the Mists**: Nightmare, Shattered Observatory, Sunqua Peak, Silent Surf,
  Lonely Tower, Kinfall

### Success/Failure Detection
Scans combat events for:
- Specific NPC death events (per-encounter lookup)
- Reward chest events (reliable cross-encounter success signal)
- Special cases handled individually (e.g. Deimos becomes untargetable at 10% health)

### Zip Handling
`.evtc.zip` and `.zevtc` files are decompressed in memory using Go's `archive/zip` before parsing.
No temp files written to disk.

### C# Reference
The EVTCAnalytics library at `evtc/EVTCAnalytics/` serves as the specification for the binary
format and encounter detection logic. When in doubt, read the C# source.

---

## Component 3: Next.js Web App

A single Next.js application deployed to Vercel. API routes handle data persistence; React handles
the UI.

### API Routes

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/logs` | Receive metadata from agent, write to DB |
| GET | `/api/logs` | Fetch log list with filters |
| GET | `/api/logs/[id]` | Fetch single log detail |
| GET | `/api/players` | Fetch player list with log counts |
| GET | `/api/statistics` | Fetch per-encounter aggregate stats |
| GET | `/api/categories` | Fetch category tree with counts |

### Authentication
A static API key in the agent config is sent as a bearer token on inbound POST requests.
This prevents unauthorized writes. Read endpoints are unauthenticated for simplicity.

---

## Component 4: React Frontend

### Layout
```
┌─────────────────────────────────────────────────────┐
│ Gorrik                                    [Settings] │
├──────────────┬──────────────────────────────────────┤
│ FILTERS      │ [Logs] [Players] [Statistics]         │
│              │                                       │
│ Result       │  (tab content)                        │
│ ☑ Success    │                                       │
│ ☑ Failure    │                                       │
│ ☑ Unknown    │                                       │
│              │                                       │
│ Mode         │                                       │
│ ☑ Normal     │                                       │
│ ☑ Challenge  │                                       │
│ ☑ Emboldened │                                       │
│ ☑ Legendary  │                                       │
│              │                                       │
│ Date         │                                       │
│ [All time ▼] │                                       │
│              │                                       │
│ CATEGORIES   │                                       │
│ ▼ All (16621)│                                       │
│  ▼ Raids     │                                       │
│    W1 (1457) │                                       │
│    W2 (1056) │                                       │
│    ...       │                                       │
│  ▶ Strikes   │                                       │
│  ▶ Fractals  │                                       │
└──────────────┴──────────────────────────────────────┘
```

### Tabs

**Logs tab**
- Sortable table: encounter name, result badge, mode, date, duration, player profession icons
- Click a row → drawer or detail page with link to dps.report (if uploaded there)
- Virtualised list (react-virtual or TanStack Virtual) to handle 16k+ rows

**Players tab**
- Search by account or character name
- Table: account name, character count, log count
- Click a player → filtered log view showing only logs containing that player

**Statistics tab**
- Per-encounter aggregate table: logs, first clear, success count, avg success time,
  failure count, avg failure time, success rate
- Filterable by category

### Component Library
**shadcn/ui** — accessible, unstyled-by-default components built on Radix UI primitives.
Styled with Tailwind CSS. Gives a modern look without fighting a design system.

### Key Libraries
- `@tanstack/react-query` — server state, caching, background refetching
- `@tanstack/react-virtual` — virtualised log list (required for 16k+ rows)
- `@tanstack/react-table` — sortable, filterable table headings
- `nuqs` — URL-based filter state (shareable/bookmarkable filter combinations)

---

## Database Schema (PostgreSQL / Neon)

```sql
-- Encounter reference data (seeded, not user-generated)
CREATE TABLE encounters (
    id          TEXT PRIMARY KEY,  -- EVTC encounter ID as string
    name        TEXT NOT NULL,
    category    TEXT NOT NULL,     -- 'raid', 'strike', 'fractal'
    subcategory TEXT NOT NULL      -- wing name, expansion, fractal name
);

-- Individual log records
CREATE TABLE logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename        TEXT NOT NULL UNIQUE,
    encounter_id    TEXT REFERENCES encounters(id),
    encounter_name  TEXT NOT NULL,  -- denormalised for convenience
    category        TEXT NOT NULL,
    subcategory     TEXT NOT NULL,
    result          TEXT NOT NULL,  -- 'success', 'failure', 'unknown'
    mode            TEXT NOT NULL,  -- 'normal', 'emboldened', 'challenge', 'legendary', 'quickplay'
    duration_ms     BIGINT NOT NULL,
    logged_at       TIMESTAMPTZ NOT NULL,
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    file_url        TEXT NOT NULL,
    is_favorite     BOOLEAN NOT NULL DEFAULT FALSE,
    tags            TEXT[] DEFAULT '{}'
);

-- GW2 accounts (one per player)
CREATE TABLE accounts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_name TEXT NOT NULL UNIQUE  -- e.g. "username.1234"
);

-- Characters (an account can have many)
CREATE TABLE characters (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   UUID NOT NULL REFERENCES accounts(id),
    character_name TEXT NOT NULL,
    UNIQUE (account_id, character_name)
);

-- Join: which characters appeared in which logs
CREATE TABLE log_players (
    log_id       UUID NOT NULL REFERENCES logs(id) ON DELETE CASCADE,
    character_id UUID NOT NULL REFERENCES characters(id),
    profession   TEXT NOT NULL,
    elite_spec   TEXT,
    PRIMARY KEY (log_id, character_id)
);

-- Indexes for common query patterns
CREATE INDEX idx_logs_logged_at      ON logs (logged_at DESC);
CREATE INDEX idx_logs_encounter_id   ON logs (encounter_id);
CREATE INDEX idx_logs_result         ON logs (result);
CREATE INDEX idx_logs_category       ON logs (category);
CREATE INDEX idx_log_players_char    ON log_players (character_id);
```

---

## Hosting & Infrastructure

| Service | Tier | Monthly Cost |
|---|---|---|
| Vercel (Next.js) | Hobby (free) | $0 |
| Neon (PostgreSQL) | Free (0.5GB, no pause) | $0 |
| Cloudflare R2 | Pay-as-you-go | ~$0.15 (current 20GB) |
| Domain (optional) | — | ~$1.00 |
| **Total** | | **~$0.15 – $1.15/month** |

R2 storage grows at roughly 1–2GB/month at current play volume. Even at 100GB total,
cost remains under $1.50/month.

---

## Repository Structure

```
gorrik/
├── agent/               # Go binary (watcher + parser + uploader)
│   ├── main.go
│   ├── config/
│   ├── parser/          # EVTC binary parser package
│   │   ├── parser.go
│   │   ├── encounters.go  # lookup table
│   │   └── result.go      # success/failure detection
│   ├── watcher/
│   ├── uploader/        # R2 upload
│   └── api/             # HTTP client for Next.js API
├── web/                 # Next.js application
│   ├── app/
│   │   ├── page.tsx     # main UI
│   │   └── api/         # API routes
│   ├── components/
│   │   ├── sidebar/
│   │   ├── logs-table/
│   │   ├── players-table/
│   │   └── statistics-table/
│   └── lib/
│       └── db.ts        # Neon connection + query helpers
├── schema.sql           # Database schema
├── PLAN.md              # This document (copy)
└── README.md
```

---

## Implementation Phases

### Phase 1: Go Agent — Parser
- Set up Go module in `agent/`
- Implement EVTC binary parser for metadata extraction
- Build encounter lookup table (raids + strikes + fractals)
- Implement success/failure detection for common encounters
- Write tests using real log files from the existing collection
- Reference: `evtc/EVTCAnalytics/` C# source as spec

### Phase 2: Go Agent — Watcher & Uploader
- Implement R2 upload using aws-sdk-go-v2
- Implement arcdps config auto-detection
- Implement fsnotify directory watcher
- Wire parser → uploader → API client
- Implement bulk import mode with goroutine-based parallelism

### Phase 3: Database & API
- Provision Neon database, apply schema
- Build Next.js API routes (POST /api/logs, GET /api/logs, etc.)
- Implement API key authentication for write endpoints
- Deploy to Vercel

### Phase 4: Frontend — Logs Tab
- Set up Next.js project with shadcn/ui + Tailwind
- Implement filter sidebar (result, mode, date, category tree)
- Implement virtualised log list with sorting
- Wire filters to URL state with nuqs

### Phase 5: Frontend — Players & Statistics Tabs
- Players tab with search and log count
- Click-through to filtered log view
- Statistics tab with per-encounter aggregates

### Phase 6: Bulk Import & Migration
- Run agent in import mode against existing 20GB / 16,621 logs
- Verify all logs indexed in database
- Verify all files uploaded to R2
- Local copies can be deleted at user's discretion after confirmation

---

## Self-Hosting

Gorrik is designed so that any person can run their own independent instance. No infrastructure
is shared between users. Each instance has its own Vercel deployment, Neon database, and R2 bucket.

To support this cleanly:
- No hardcoded URLs or credentials anywhere in the codebase
- `.env.example` file documents all required environment variables for the Next.js app
- `schema.sql` is the single source of truth for database setup
- README includes a full self-hosting setup guide covering Vercel, Neon, and R2 provisioning

A guildmate who wants their own instance clones the repo, follows the setup guide, and configures
their own `gorrik.toml`. Their data never touches anyone else's infrastructure.

---

## Deferred (Future Phases)

- **GW2 API integration** — guild tags, guild names, member lookups
- **Guilds tab** — requires GW2 API
- **Weekly clears tab** — requires reset timer logic and encounter scheduling data
- **dps.report upload** — trigger upload from within Gorrik UI, store permalink
- **Multi-user / sharing** — open the app to guildmates
