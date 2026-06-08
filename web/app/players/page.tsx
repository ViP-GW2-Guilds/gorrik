import { db } from "@/lib/db";
import { sql } from "drizzle-orm";
import { NavTabs } from "@/components/layout/nav-tabs";
import { PlayerTable, type PlayerStat } from "@/components/stats/player-table";
import type { SpecEntry } from "@/components/stats/encounter-table";

async function fetchPlayerStats(): Promise<PlayerStat[]> {
  const [baseResult, specResult] = await Promise.all([
    db.execute(sql`
      SELECT
        a.id                                                              AS account_id,
        a.account_name,
        COUNT(DISTINCT lp.log_id)::int                                   AS total_logs,
        COUNT(DISTINCT CASE WHEN l.result = 'success' THEN lp.log_id END)::int AS kills
      FROM log_players lp
      JOIN characters c ON lp.character_id = c.id
      JOIN accounts a ON c.account_id = a.id
      JOIN logs l ON lp.log_id = l.id
      GROUP BY a.id, a.account_name
      ORDER BY total_logs DESC
    `),
    db.execute(sql`
      SELECT
        a.id AS account_id,
        lp.profession,
        lp.elite_spec,
        COUNT(*)::int AS cnt
      FROM log_players lp
      JOIN characters c ON lp.character_id = c.id
      JOIN accounts a ON c.account_id = a.id
      GROUP BY a.id, lp.profession, lp.elite_spec
      ORDER BY a.id, cnt DESC
    `),
  ]);

  const specsByAccount = new Map<string, SpecEntry[]>();
  for (const row of specResult.rows) {
    const id = row.account_id as string;
    if (!specsByAccount.has(id)) specsByAccount.set(id, []);
    const list = specsByAccount.get(id)!;
    if (list.length < 5) {
      list.push({
        profession: row.profession as string,
        eliteSpec: (row.elite_spec as string | null) || null,
      });
    }
  }

  return baseResult.rows.map((row) => ({
    accountId: row.account_id as string,
    accountName: row.account_name as string,
    totalLogs: row.total_logs as number,
    kills: row.kills as number,
    topSpecs: specsByAccount.get(row.account_id as string) ?? [],
  }));
}

export default async function PlayersPage() {
  const players = await fetchPlayerStats();

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Header */}
      <div className="absolute top-0 left-0 right-0 h-12 border-b border-border flex items-center px-4 gap-2 bg-background z-20">
        <span className="font-semibold tracking-tight">Gorrik</span>
        <NavTabs />
      </div>

      {/* Content */}
      <div className="w-full pt-12 h-screen overflow-y-auto">
        <div className="max-w-4xl mx-auto px-6 py-6">
          <PlayerTable players={players} />
        </div>
      </div>
    </div>
  );
}
