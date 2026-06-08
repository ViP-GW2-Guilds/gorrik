import { NextResponse } from "next/server";
import { db } from "@/lib/db";
import { sql } from "drizzle-orm";

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;

  const result = await db.execute(sql`
    SELECT
      ch.id                                                      AS character_id,
      ch.character_name,
      COUNT(DISTINCT lp.log_id)::int                            AS total_logs,
      COUNT(DISTINCT CASE WHEN l.result = 'success' THEN lp.log_id END)::int AS kills,
      (
        SELECT lp2.profession
        FROM log_players lp2
        WHERE lp2.character_id = ch.id
        GROUP BY lp2.profession, lp2.elite_spec
        ORDER BY COUNT(*) DESC
        LIMIT 1
      ) AS top_profession,
      (
        SELECT lp2.elite_spec
        FROM log_players lp2
        WHERE lp2.character_id = ch.id
        GROUP BY lp2.profession, lp2.elite_spec
        ORDER BY COUNT(*) DESC
        LIMIT 1
      ) AS top_elite_spec
    FROM log_players lp
    JOIN characters ch ON lp.character_id = ch.id
    JOIN accounts a ON ch.account_id = a.id
    JOIN logs l ON lp.log_id = l.id
    WHERE a.id = ${id}
    GROUP BY ch.id, ch.character_name
    ORDER BY total_logs DESC
  `);

  const characters = result.rows.map((row) => ({
    characterId: row.character_id as string,
    characterName: row.character_name as string,
    totalLogs: row.total_logs as number,
    kills: row.kills as number,
    topSpec:
      row.top_profession != null
        ? {
            profession: row.top_profession as string,
            eliteSpec: (row.top_elite_spec as string | null) || null,
          }
        : null,
  }));

  return NextResponse.json({ characters });
}
