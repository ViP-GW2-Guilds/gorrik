import Image from "next/image";
import { iconPath, formatDuration } from "@/lib/gw2";
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
  fastestMs: number | null;
  meanMs: number | null;
  medianMs: number | null;
  topSpecs: SpecEntry[];
};

function SuccessRate({ total, kills }: { total: number; kills: number }) {
  const pct = total > 0 ? Math.round((kills / total) * 100) : 0;
  return (
    <span>
      {kills}/{total}{" "}
      <span className="text-muted-foreground">({pct}%)</span>
    </span>
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

export function EncounterTable({ stats }: { stats: EncounterStat[] }) {
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

  // Sort categories, subcategories, and encounters by canonical game order
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
          <div key={category}>
            <h3 className="text-sm font-semibold text-primary mb-3">
              {CATEGORY_LABELS[category] ?? category}
            </h3>
            <table className="w-full text-sm border-collapse">
              <thead>
                <tr className="border-b border-border">
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
                    <tr key={`sub-${subcategory}`} className="bg-muted/20">
                      <td
                        colSpan={6}
                        className="px-3 py-1 text-xs text-muted-foreground font-medium uppercase tracking-wider"
                      >
                        {subcategory}
                      </td>
                    </tr>,
                    ...sortedEncounters.map((stat) => (
                      <tr
                        key={stat.encounterName}
                        className="border-b border-border/40 hover:bg-muted/30 transition-colors"
                      >
                        <td className="px-3 py-2.5 pl-5">{stat.encounterName}</td>
                        <td className="px-3 py-2.5">
                          <SuccessRate total={stat.totalLogs} kills={stat.kills} />
                        </td>
                        <td className="px-3 py-2.5 tabular-nums">
                          {stat.fastestMs != null ? formatDuration(stat.fastestMs) : "—"}
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
                      </tr>
                    )),
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
