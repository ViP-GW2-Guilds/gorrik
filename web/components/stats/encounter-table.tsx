"use client";

import { useState } from "react";
import Image from "next/image";
import { ChevronRight } from "lucide-react";
import { iconPath, formatDuration, slugify } from "@/lib/gw2";
import {
  CATEGORY_LABELS,
  CATEGORY_ORDER,
  subcategoryIndex,
  encounterIndex,
} from "@/lib/gw2-encounters";

export type SpecEntry = { profession: string; eliteSpec: string | null };

export type EncounterStat = {
  encounterName: string;
  category: string;
  subcategory: string;
  totalLogs: number;
  kills: number;
  firstKillAt: string | null;
  latestKillAt: string | null;
  fastestMs: number | null;
  fastestLoggedAt: string | null;
  meanMs: number | null;
  medianMs: number | null;
  topSpecs: SpecEntry[];
};

function SuccessRate({ total, kills }: { total: number; kills: number }) {
  const pct = total > 0 ? Math.round((kills / total) * 100) : 0;
  return (
    <div>
      <div>{kills}/{total}</div>
      <div className="text-muted-foreground text-xs">({pct}%)</div>
    </div>
  );
}

function SpecIcons({ specs }: { specs: SpecEntry[] }) {
  return (
    <div className="flex items-center gap-1">
      {specs.map((s, i) => (
        <Image
          key={i}
          src={iconPath(s.profession, s.eliteSpec)}
          alt={s.eliteSpec ?? s.profession}
          width={20}
          height={20}
          unoptimized
          title={s.eliteSpec ?? s.profession}
        />
      ))}
    </div>
  );
}

function EncounterDetail({ stat }: { stat: EncounterStat }) {
  return (
    <tr>
      <td colSpan={6} className="px-3 pb-3 pt-0">
        <div className="ml-4 pl-4 border-l border-border/60 flex gap-8 text-sm py-2">
          <div>
            <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">First kill</div>
            <div className="tabular-nums">{stat.firstKillAt ?? "—"}</div>
          </div>
          <div>
            <div className="text-xs text-muted-foreground uppercase tracking-wider mb-1">Latest kill</div>
            <div className="tabular-nums">{stat.latestKillAt ?? "—"}</div>
          </div>
        </div>
      </td>
    </tr>
  );
}

export function EncounterTable({ stats }: { stats: EncounterStat[] }) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const toggle = (name: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });

  if (stats.length === 0) {
    return <p className="text-muted-foreground text-sm">No encounter data yet.</p>;
  }

  // Group by category → subcategory
  const grouped = new Map<string, Map<string, EncounterStat[]>>();
  for (const stat of stats) {
    if (!grouped.has(stat.category)) grouped.set(stat.category, new Map());
    const bySub = grouped.get(stat.category)!;
    if (!bySub.has(stat.subcategory)) bySub.set(stat.subcategory, []);
    bySub.get(stat.subcategory)!.push(stat);
  }

  const sortedCategories = [...grouped.keys()].sort(
    (a, b) => CATEGORY_ORDER.indexOf(a) - CATEGORY_ORDER.indexOf(b)
  );

  return (
    <div className="space-y-8">
      {sortedCategories.map((category) => {
        const bySub = grouped.get(category)!;

        const sortedSubs = [...bySub.entries()].sort(
          ([a], [b]) => subcategoryIndex(a) - subcategoryIndex(b)
        );

        return (
          <div key={category} id={`cat-${slugify(category)}`}>
            <h3 className="text-sm font-semibold text-primary mb-3">
              {CATEGORY_LABELS[category] ?? category}
            </h3>
            <table className="w-full text-sm border-collapse">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider w-6" />
                  <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider">Encounter</th>
                  <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider w-28">Kills</th>
                  <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider w-20">Fastest</th>
                  <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider w-20">Mean</th>
                  <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider w-20">Median</th>
                  <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider">Top Specs</th>
                </tr>
              </thead>
              <tbody>
                {sortedSubs.flatMap(([subcategory, encounters]) => {
                  const sortedEncounters = [...encounters].sort(
                    (a, b) => encounterIndex(a.encounterName) - encounterIndex(b.encounterName)
                  );
                  return [
                    <tr key={`sub-${subcategory}`} id={`sub-${slugify(subcategory)}`} className="bg-muted/20">
                      <td
                        colSpan={7}
                        className="px-3 py-1 text-xs text-muted-foreground font-medium uppercase tracking-wider"
                      >
                        {subcategory}
                      </td>
                    </tr>,
                    ...sortedEncounters.flatMap((stat) => {
                      const isExpanded = expanded.has(stat.encounterName);
                      const canExpand = stat.kills > 0;
                      return [
                        <tr
                          key={stat.encounterName}
                          id={`enc-${slugify(stat.encounterName)}`}
                          onClick={canExpand ? () => toggle(stat.encounterName) : undefined}
                          className={`border-b border-border/40 transition-colors ${
                            canExpand ? "cursor-pointer" : ""
                          } ${isExpanded ? "bg-muted/60" : canExpand ? "hover:bg-muted/30" : ""}`}
                        >
                          <td className="px-3 py-2.5 w-6">
                            {canExpand && (
                              <ChevronRight
                                className={`w-3.5 h-3.5 text-muted-foreground transition-transform ${
                                  isExpanded ? "rotate-90" : ""
                                }`}
                              />
                            )}
                          </td>
                          <td className="px-3 py-2.5 pl-2">{stat.encounterName}</td>
                          <td className="px-3 py-2.5">
                            <SuccessRate total={stat.totalLogs} kills={stat.kills} />
                          </td>
                          <td className="px-3 py-2.5 tabular-nums">
                            {stat.fastestMs != null ? (
                              <div>
                                <div>{formatDuration(stat.fastestMs)}</div>
                                {stat.fastestLoggedAt && (
                                  <div className="text-muted-foreground text-xs">{stat.fastestLoggedAt}</div>
                                )}
                              </div>
                            ) : "—"}
                          </td>
                          <td className="px-3 py-2.5 tabular-nums">
                            {stat.meanMs != null ? formatDuration(stat.meanMs) : "—"}
                          </td>
                          <td className="px-3 py-2.5 tabular-nums">
                            {stat.medianMs != null ? formatDuration(stat.medianMs) : "—"}
                          </td>
                          <td className="px-3 py-2.5">
                            <SpecIcons specs={stat.topSpecs} />
                          </td>
                        </tr>,
                        ...(isExpanded ? [<EncounterDetail key={`${stat.encounterName}-detail`} stat={stat} />] : []),
                      ];
                    }),
                  ];
                })}
              </tbody>
            </table>
          </div>
        );
      })}
    </div>
  );
}
