"use client";

import { useEffect, useState } from "react";
import { useQueryState } from "nuqs";
import { ChevronRight, ChevronDown } from "lucide-react";
import {
  CATEGORY_LABELS,
  CATEGORY_ORDER,
  ENCOUNTER_DATA,
  subcategoryIndex,
  encounterIndex,
} from "@/lib/gw2-encounters";
import { slugify } from "@/lib/gw2";

interface SidebarCount {
  category: string;
  subcategory: string;
  encounterName: string;
  count: number;
}

interface EncNode {
  name: string;
  count: number;
}

interface SubNode {
  name: string;
  count: number;
  encounters: EncNode[];
}

interface CatNode {
  category: string;
  count: number;
  subcategories: SubNode[];
}

function buildTree(apiCounts: SidebarCount[]): CatNode[] {
  const countMap = new Map<string, number>();
  for (const c of apiCounts) countMap.set(c.encounterName, c.count);

  // Unknown encounters (in DB but not in static list) — preserve their category/subcategory
  const knownNames = new Set(ENCOUNTER_DATA.map((e) => e.name));
  const unknownEntries = apiCounts
    .filter((c) => !knownNames.has(c.encounterName))
    .map((c) => ({ category: c.category, subcategory: c.subcategory, name: c.encounterName }));

  const allEntries = [...ENCOUNTER_DATA, ...unknownEntries];

  const catMap = new Map<string, Map<string, Map<string, number>>>();
  for (const e of allEntries) {
    if (!catMap.has(e.category)) catMap.set(e.category, new Map());
    const subMap = catMap.get(e.category)!;
    if (!subMap.has(e.subcategory)) subMap.set(e.subcategory, new Map());
    subMap.get(e.subcategory)!.set(e.name, countMap.get(e.name) ?? 0);
  }

  return CATEGORY_ORDER.filter((cat) => catMap.has(cat)).map((cat) => {
    const subMap = catMap.get(cat)!;
    const subs: SubNode[] = [...subMap.entries()]
      .sort(([a], [b]) => subcategoryIndex(a) - subcategoryIndex(b))
      .map(([sub, encMap]) => {
        const encs: EncNode[] = [...encMap.entries()]
          .sort(([a], [b]) => encounterIndex(a) - encounterIndex(b))
          .map(([name, count]) => ({ name, count }));
        return { name: sub, count: encs.reduce((s, e) => s + e.count, 0), encounters: encs };
      });
    return {
      category: cat,
      count: subs.reduce((s, sub) => s + sub.count, 0),
      subcategories: subs,
    };
  });
}

const STORAGE_KEY = "gorrik-sidebar-expanded";

function loadExpanded(): Set<string> {
  if (typeof window === "undefined") return new Set();
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    return saved ? new Set(JSON.parse(saved)) : new Set();
  } catch {
    return new Set();
  }
}

function saveExpanded(next: Set<string>) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify([...next]));
}

export function EncounterTree({ mode }: { mode: "filter" | "scroll" }) {
  const [counts, setCounts] = useState<SidebarCount[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [loaded, setLoaded] = useState(false);

  const [category, setCategory] = useQueryState("category", { shallow: false });
  const [subcategory, setSubcategory] = useQueryState("subcategory", { shallow: false });
  const [encounter, setEncounter] = useQueryState("encounter", { shallow: false });

  useEffect(() => {
    setExpanded(loadExpanded());
    fetch("/api/sidebar-counts")
      .then((r) => r.json())
      .then((data) => {
        setCounts(data);
        setLoaded(true);
      });
  }, []);

  function toggleExpand(key: string) {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      saveExpanded(next);
      return next;
    });
  }

  function expandKey(key: string) {
    setExpanded((prev) => {
      if (prev.has(key)) return prev;
      const next = new Set(prev);
      next.add(key);
      saveExpanded(next);
      return next;
    });
  }

  function scrollTo(id: string) {
    document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function onCategoryClick(cat: string) {
    expandKey(`cat-${cat}`);
    if (mode === "filter") {
      setEncounter(null);
      setSubcategory(null);
      setCategory(category === cat ? null : cat);
    } else {
      scrollTo(`cat-${slugify(cat)}`);
    }
  }

  function onSubcategoryClick(cat: string, sub: string) {
    expandKey(`sub-${sub}`);
    if (mode === "filter") {
      setEncounter(null);
      setCategory(cat);
      setSubcategory(subcategory === sub ? null : sub);
    } else {
      scrollTo(`sub-${slugify(sub)}`);
    }
  }

  function onEncounterClick(cat: string, sub: string, enc: string) {
    if (mode === "filter") {
      if (encounter === enc) {
        setEncounter(null);
      } else {
        setCategory(null);
        setSubcategory(null);
        setEncounter(enc);
      }
    } else {
      scrollTo(`enc-${slugify(enc)}`);
    }
  }

  const tree = buildTree(counts);
  const hasFilter = mode === "filter" && !!(category || subcategory || encounter);

  return (
    <aside className="w-72 shrink-0 flex flex-col border-r border-border overflow-hidden bg-card">
      <div className="px-3 py-2 flex items-center justify-between border-b border-border shrink-0">
        <span className="font-semibold text-sm">Encounters</span>
        {hasFilter && (
          <button
            onClick={() => {
              setCategory(null);
              setSubcategory(null);
              setEncounter(null);
            }}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            Clear
          </button>
        )}
      </div>

      <div className="flex-1 overflow-y-auto py-1">
        {!loaded ? (
          <p className="text-xs text-muted-foreground px-4 py-3">Loading…</p>
        ) : (
          tree.map((catNode) => {
            const catKey = `cat-${catNode.category}`;
            const isCatExpanded = expanded.has(catKey);
            const isCatActive =
              mode === "filter" &&
              category === catNode.category &&
              !subcategory &&
              !encounter;

            return (
              <div key={catNode.category}>
                <div className="flex items-center gap-0.5 px-1">
                  <button
                    onClick={() => toggleExpand(catKey)}
                    className="p-1 rounded hover:bg-muted transition-colors text-muted-foreground shrink-0"
                    aria-label={isCatExpanded ? "Collapse" : "Expand"}
                  >
                    {isCatExpanded ? (
                      <ChevronDown className="w-3 h-3" />
                    ) : (
                      <ChevronRight className="w-3 h-3" />
                    )}
                  </button>
                  <button
                    onClick={() => onCategoryClick(catNode.category)}
                    className={`flex-1 flex items-center justify-between gap-1 text-left text-xs font-semibold uppercase tracking-wider px-1 py-1.5 rounded transition-colors ${
                      isCatActive
                        ? "text-primary bg-primary/10"
                        : "text-foreground hover:bg-muted"
                    }`}
                  >
                    <span className="truncate">
                      {CATEGORY_LABELS[catNode.category] ?? catNode.category}
                    </span>
                    <span className="text-muted-foreground font-normal shrink-0">
                      ({catNode.count})
                    </span>
                  </button>
                </div>

                {isCatExpanded &&
                  catNode.subcategories.map((subNode) => {
                    const subKey = `sub-${subNode.name}`;
                    const isSubExpanded = expanded.has(subKey);
                    const isSubActive =
                      mode === "filter" && subcategory === subNode.name && !encounter;

                    return (
                      <div key={subNode.name}>
                        <div className="flex items-center gap-0.5 pl-5 pr-1">
                          <button
                            onClick={() => toggleExpand(subKey)}
                            className="p-1 rounded hover:bg-muted transition-colors text-muted-foreground shrink-0"
                            aria-label={isSubExpanded ? "Collapse" : "Expand"}
                          >
                            {isSubExpanded ? (
                              <ChevronDown className="w-3 h-3" />
                            ) : (
                              <ChevronRight className="w-3 h-3" />
                            )}
                          </button>
                          <button
                            onClick={() => onSubcategoryClick(catNode.category, subNode.name)}
                            className={`flex-1 flex items-center justify-between gap-1 text-left text-xs px-1 py-1 rounded transition-colors ${
                              isSubActive
                                ? "text-primary bg-primary/10"
                                : "text-muted-foreground hover:text-foreground hover:bg-muted"
                            }`}
                          >
                            <span className="truncate">{subNode.name}</span>
                            <span className="shrink-0">({subNode.count})</span>
                          </button>
                        </div>

                        {isSubExpanded &&
                          subNode.encounters.map((encNode) => {
                            const isEncActive =
                              mode === "filter" && encounter === encNode.name;
                            return (
                              <div key={encNode.name} className="pl-12 pr-1">
                                <button
                                  onClick={() =>
                                    onEncounterClick(
                                      catNode.category,
                                      subNode.name,
                                      encNode.name
                                    )
                                  }
                                  className={`w-full flex items-center justify-between gap-1 text-left text-xs px-2 py-0.5 rounded transition-colors ${
                                    isEncActive
                                      ? "bg-primary text-primary-foreground"
                                      : "text-foreground hover:bg-muted"
                                  }`}
                                >
                                  <span className="truncate">{encNode.name}</span>
                                  <span
                                    className={`shrink-0 ${
                                      isEncActive
                                        ? "text-primary-foreground/70"
                                        : "text-muted-foreground"
                                    }`}
                                  >
                                    ({encNode.count})
                                  </span>
                                </button>
                              </div>
                            );
                          })}
                      </div>
                    );
                  })}
              </div>
            );
          })
        )}
      </div>
    </aside>
  );
}
