# Gorrik

A personal GW2 combat log archiver. The agent runs on your Windows gaming machine, watches your arcdps log directory, parses new logs, uploads the raw files to Cloudflare R2, and indexes metadata in a Neon PostgreSQL database. The web UI (Next.js, deployed on Vercel) displays the indexed logs with filtering, encounter stats, and player breakdowns.

## Components

| Component | Location | Runs on |
|-----------|----------|---------|
| Agent | `agent/` | Windows (your gaming PC) |
| Web UI | `web/` | Vercel |
| Database | Neon PostgreSQL | Cloud |
| File storage | Cloudflare R2 | Cloud |

---

## Agent

### First-time setup

Run the interactive setup wizard to create `gorrik.toml`:

```
gorrik setup
```

The wizard asks for your arcdps log directory, Cloudflare R2 credentials, and the web API URL and key. On Windows, the config file is saved to `%APPDATA%\gorrik\gorrik.toml`. You can also edit it directly.

### Config file reference

```toml
[arcdps]
auto_detect = true          # read log dir from arcdps.ini automatically
log_dir     = ""            # override: explicit path to log directory

[arcdps_log_manager]
cache_path = ""             # override: path to LogDataCache.json
                            # (auto-detected at %LOCALAPPDATA%\ArcdpsLogManager\LogDataCache.json)

[api]
url = "https://your-app.vercel.app/api"
key = "your-api-key"

[storage]
r2_account_id        = "..."
r2_access_key_id     = "..."
r2_secret_access_key = "..."
r2_bucket            = "..."

[behaviour]
delete_after_upload    = false  # delete local log file after successful upload
upload_to_dps_report   = false  # upload new logs to dps.report during watch
dps_report_user_token  = ""     # optional: associate dps.report uploads with your account
```

### Commands

#### `gorrik status`

Prints a summary of the current setup: the config file in use, the resolved
arcdps log directory and its size, how many logs are indexed in the database,
and how many local logs are not yet indexed. Running `gorrik` with no arguments
does the same thing.

```
gorrik status
```

#### `gorrik sync`

Runs the full catch-up flow in order: `import` → `import-dps-urls` → `backfill-dps`.
The log directory and Log Manager cache path come from the config file, so no
arguments are needed. Use this after the agent has been offline for a while.

```
gorrik sync [--dry-run] [--skip-dps]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | false | Show what would happen without uploading or writing |
| `--skip-dps` | false | Skip the `import-dps-urls` and `backfill-dps` steps |

#### `gorrik watch`

Watches the log directory and processes each new `.evtc` / `.zevtc` file as arcdps creates it. For each file: parse → upload to R2 → (optionally) upload to dps.report → index in the database.

```
gorrik watch [--dry-run]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | false | Parse and log what would happen without uploading anything |

To run without a terminal window, install it as a Windows service instead (see below).

**dps.report uploads:** set `upload_to_dps_report = true` in `[behaviour]`. Each new log is uploaded to dps.report immediately after the R2 upload. A link to the dps.report page appears in the web UI.

#### `gorrik import`

Bulk-imports all existing log files from the log directory. Skips files already in the database (safe to re-run).

```
gorrik import [--dir <path>] [--workers <n>] [--dry-run]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | arcdps log dir from config | Directory to import from |
| `--workers` | 4 | Number of parallel upload workers |
| `--dry-run` | false | Parse and log without uploading |

The directory must be given with the `--dir` flag. A bare positional path
(`gorrik import C:\logs`) is rejected — it is not treated as the directory.

Note: `gorrik import` does not upload to dps.report. Use `gorrik backfill-dps` for that after importing.

#### `gorrik backfill-dps`

Uploads to dps.report any log that is in the database but has no dps.report URL. Rate-limited and safe to interrupt and re-run — already-uploaded logs are skipped on the next run.

```
gorrik backfill-dps [--dir <path>] [--rate <n>] [--dry-run]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | arcdps log dir from config | Directory containing the local log files |
| `--rate` | 1.0 | Max uploads per second (e.g. `0.5` = one every two seconds) |
| `--dry-run` | false | Show what would be uploaded without uploading |

As with `gorrik import`, the directory must be passed via `--dir`; a bare
positional path is rejected.

HTTP 429 responses are handled automatically: the command waits for the duration in the `Retry-After` header (defaulting to 60 s) and retries up to 3 times before giving up on that file.

**Recommended workflow for historical logs:**
1. Run `gorrik import-dps-urls` first — this copies URLs already recorded by arcdps Log Manager without re-uploading anything.
2. Run `gorrik backfill-dps` for anything still missing.

#### `gorrik import-dps-urls`

Reads the arcdps Log Manager cache file and patches any matching database record with its stored dps.report URL. No files are uploaded — this is a zero-cost way to populate URLs for logs that were previously uploaded through Log Manager.

```
gorrik import-dps-urls --cache <path>
```

| Flag | Required | Description |
|------|----------|-------------|
| `--cache` | yes | Path to `LogDataCache.json` (usual location: `%LOCALAPPDATA%\ArcdpsLogManager\LogDataCache.json`) |

Logs are matched by base filename. Logs that already have a URL in the database are skipped.

`gorrik sync` runs this step automatically using the cache path from
`[arcdps_log_manager]` in the config, or the auto-detected default.

#### `gorrik service`

Manages Gorrik as a Windows service so `gorrik watch` runs automatically at login without a terminal window. Requires administrator privileges.

```
gorrik service install    # register the service
gorrik service start      # start it
gorrik service stop       # stop it
gorrik service uninstall  # remove it
```

The service runs `gorrik watch` using the same config file as the CLI (`%APPDATA%\gorrik\gorrik.toml`).

#### Global flag

```
gorrik --config <path>   # use a custom config file path
```

---

## Web UI

Deployed on Vercel. Displays all indexed logs with:

- **Logs page** — filterable by result, mode, and encounter category; sortable columns; expandable rows showing the player roster
- **Encounters page** — per-encounter kill rates, fastest kill times, and top specs
- **Players page** — per-account stats with per-character breakdowns

Logs that have a dps.report URL show an external-link icon in the rightmost column.

---

## Development

### Prerequisites

- Go 1.24+ (agent)
- Node.js 20+ with pnpm (web)
- A Neon PostgreSQL database
- A Cloudflare R2 bucket

### Agent

```bash
cd agent
go build -o gorrik .
```

Cross-compile for Windows from Linux/macOS:

```bash
cd agent
GOOS=windows GOARCH=amd64 go build -o gorrik.exe .
```

### Web

```bash
cd web
pnpm install
pnpm dev
```

Copy `web/.env.local.example` to `web/.env.local` and fill in `DATABASE_URL` and `API_KEY`.

Push schema changes to the database:

```bash
pnpm db:push
```
