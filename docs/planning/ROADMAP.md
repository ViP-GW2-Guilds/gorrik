# Gorrik — Roadmap

## Active Backlog

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
