# Stats Roadmap

## Built (Phase 1)

### Encounter Stats (`/encounters`)
- ✅ Success rate (kills / total logs)
- ✅ Fastest kill time
- ✅ Mean kill time
- ✅ Median kill time
- ✅ Most popular specs per encounter (top 5, shown as icons)
- ✅ Grouped by category → wing/area in canonical game order
- ✅ Category sidebar filter (Raids / Raid Encounters / Fractals / Other)

### Player Stats (`/players`)
- ✅ Total logs attended per account
- ✅ Success rate across attended logs
- ✅ Most played specs (top 5, shown as icons)
- ✅ Expandable character drill-down: per-character log count, success rate, and most played spec

---

## Planned

### Character Drill-Down Detail
Currently the expanded character row shows log count, success rate, and top spec icon.
The original intent was richer detail:
- Which specific encounters that character attended
- Kill times they were part of (per-encounter breakdown)

This may be better suited to a dedicated character detail page/modal rather than an
inline expansion, given the potential data density.

### Hierarchical Sidebar with Log Counts ✅

Replaced the flat category sidebar on both Logs and Encounters pages with a collapsible
encounter tree. Shows counts at every level (category → wing/area → encounter).

**Behaviour:**
- Default: all nodes collapsed; all logs/encounters shown in main viewport
- Zero-log encounters appear with `(0)` and are fully clickable
- Expansion state persisted in `localStorage`
- **Logs page** (`mode="filter"`): clicking any node sets URL params to filter the logs table
  - Category → `?category=…`
  - Subcategory → `?category=…&subcategory=…`
  - Encounter → `?encounter=…` (clears category/subcategory since encounter is unambiguous)
  - "Clear" button appears when any filter is active
- **Encounters page** (`mode="scroll"`): clicking any node scrolls to that section of the table
  - IDs on category divs, subcategory header rows, and encounter rows enable smooth scroll

**Implementation:**
- `web/components/sidebar/encounter-tree.tsx` — new component (replaces `filters.tsx`)
- `web/app/api/sidebar-counts/route.ts` — aggregates log counts from DB
- `web/lib/gw2-encounters.ts` — added `ENCOUNTER_DATA` (full static tree for zero-log entries)
- `web/lib/gw2.ts` — added `slugify()` shared utility
- Both page server components updated to handle `subcategory` and `encounter` URL params

### Tags
Schema already has `tags text[]` on the logs table. UI for viewing/filtering/editing tags
is deferred. No design decisions needed before building the sidebar.
