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

### Hierarchical Sidebar with Log Counts

Replace (or evolve) the current flat category sidebar with a tree that surfaces log counts
at every level. Inspired by the arcdps Logs Manager layout:

```
Category                                (count)
    Raids                               (count)
        Spirit Vale (Wing 1)            (count)
            Vale Guardian               (count)
            Gorseval the Multifarious   (count)
            Sabetha the Saboteur        (count)
        Salvation Pass (Wing 2)         (count)
            Slothasor                   (count)
            Bandit Trio                 (count)
            Matthias Gabrel             (count)
        ...
    Raid Encounters                     (count)
        Icebrood Saga                   (count)
            ...
        End of Dragons                  (count)
            ...
    Fractals                            (count)
        ...
```

Benefits:
- Gives the app much stronger navigation structure
- Surfaces "how many logs do I have for X" at a glance without a separate stats page
- Clicking a wing/encounter node filters the logs list to that scope (deeper than just category)

Design questions to resolve:
- Does this live on the Logs page only, or also on Encounters?
- Collapsible tree nodes?
- How to handle encounters the user has zero logs for (show greyed out, or hide entirely)?
- Should clicking an encounter name deep-link to the Encounters page filtered to that encounter?

### Tags
Schema already has `tags text[]` on the logs table. UI for viewing/filtering/editing tags
is deferred. No design decisions needed before building the sidebar.
