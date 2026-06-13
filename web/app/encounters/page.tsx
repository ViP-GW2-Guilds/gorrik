import { Suspense } from "react";
import { NuqsAdapter } from "nuqs/adapters/next/app";
import { db } from "@/lib/db";
import { sql } from "drizzle-orm";
import { NavTabs } from "@/components/layout/nav-tabs";
import { EncounterTree } from "@/components/sidebar/encounter-tree";
import { EncounterTable, type EncounterStat, type SpecEntry } from "@/components/stats/encounter-table";

async function fetchEncounterStats(category?: string): Promise<EncounterStat[]> {
  // Two separate fragments: CTE uses unaliased columns, outer query uses the 'l' alias
  const ctaFilter = category ? sql`AND category = ${category}` : sql``;
  const outerFilter = category ? sql`AND l.category = ${category}` : sql``;

  const [baseResult, specResult] = await Promise.all([
    db.execute(sql`
      WITH success AS (
        SELECT encounter_name, duration_ms, logged_at
        FROM logs
        WHERE result = 'success' ${ctaFilter}
      ),
      fastest AS (
        SELECT DISTINCT ON (encounter_name) encounter_name, duration_ms, logged_at
        FROM success
        ORDER BY encounter_name, duration_ms ASC
      ),
      medians AS (
        SELECT
          encounter_name,
          PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms) AS median_ms
        FROM success
        GROUP BY encounter_name
      )
      SELECT
        l.encounter_name,
        l.category,
        l.subcategory,
        COUNT(*)::int                                                    AS total_logs,
        COUNT(*) FILTER (WHERE l.result = 'success')::int               AS kills,
        f.duration_ms                                                    AS fastest_ms,
        f.logged_at                                                      AS fastest_logged_at,
        ROUND(AVG(l.duration_ms) FILTER (WHERE l.result = 'success'))::bigint AS mean_ms,
        m.median_ms
      FROM logs l
      LEFT JOIN fastest f ON l.encounter_name = f.encounter_name
      LEFT JOIN medians m ON l.encounter_name = m.encounter_name
      WHERE 1=1 ${outerFilter}
      GROUP BY l.encounter_name, l.category, l.subcategory, f.duration_ms, f.logged_at, m.median_ms
      ORDER BY l.category, l.subcategory, l.encounter_name
    `),
    db.execute(sql`
      SELECT
        l.encounter_name,
        lp.profession,
        lp.elite_spec,
        COUNT(*)::int AS cnt
      FROM log_players lp
      JOIN logs l ON lp.log_id = l.id
      WHERE l.result = 'success' ${outerFilter}
      GROUP BY l.encounter_name, lp.profession, lp.elite_spec
      ORDER BY l.encounter_name, cnt DESC
    `),
  ]);

  const specsByEncounter = new Map<string, SpecEntry[]>();
  for (const row of specResult.rows) {
    const name = row.encounter_name as string;
    if (!specsByEncounter.has(name)) specsByEncounter.set(name, []);
    const list = specsByEncounter.get(name)!;
    if (list.length < 5) {
      list.push({
        profession: row.profession as string,
        eliteSpec: (row.elite_spec as string | null) || null,
      });
    }
  }

  return baseResult.rows.map((row) => ({
    encounterName: row.encounter_name as string,
    category: row.category as string,
    subcategory: row.subcategory as string,
    totalLogs: row.total_logs as number,
    kills: row.kills as number,
    fastestMs: (row.fastest_ms as number | null) ?? null,
    fastestLoggedAt: row.fastest_logged_at
      ? new Date(row.fastest_logged_at as string).toISOString().slice(0, 10)
      : null,
    meanMs: (row.mean_ms as number | null) ?? null,
    medianMs: (row.median_ms as number | null) ?? null,
    topSpecs: specsByEncounter.get(row.encounter_name as string) ?? [],
  }));
}

export default async function EncountersPage({
  searchParams,
}: {
  searchParams: Promise<{ category?: string }>;
}) {
  const { category } = await searchParams;
  const stats = await fetchEncounterStats(category);

  return (
    <NuqsAdapter>
      <div className="flex h-screen overflow-hidden">
        {/* Header */}
        <div className="absolute top-0 left-0 right-0 h-12 border-b border-border flex items-center px-4 gap-2 bg-background z-20">
          <span className="font-semibold tracking-tight">Gorrik</span>
          <NavTabs />
        </div>

        {/* Body */}
        <div className="flex w-full pt-12 h-screen overflow-hidden">
          <Suspense>
            <EncounterTree mode="scroll" />
          </Suspense>
          <main className="flex-1 overflow-y-auto">
            <div className="px-6 py-6">
              <EncounterTable stats={stats} />
            </div>
          </main>
        </div>
      </div>
    </NuqsAdapter>
  );
}
