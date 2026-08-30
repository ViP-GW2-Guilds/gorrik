import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { sql } from "drizzle-orm";

// GET /api/logs/stats — summary counts for the gorrik agent's `status` command.
export async function GET(req: NextRequest) {
  const apiKey = req.headers.get("authorization")?.replace("Bearer ", "");
  if (apiKey !== process.env.API_KEY) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const [row] = (
    await db.execute(sql`
      SELECT
        COUNT(*)::int                                       AS total,
        MAX(logged_at)                                      AS newest_logged_at,
        COUNT(*) FILTER (WHERE dps_report_url IS NULL)::int AS missing_dps_report
      FROM logs
    `)
  ).rows;

  const newest = row?.newest_logged_at as string | null;

  return NextResponse.json({
    total: (row?.total as number) ?? 0,
    // Normalise Postgres' "2026-08-26 02:00:21+00" to ISO 8601 so the Go agent
    // can unmarshal it into a time.Time.
    newestLoggedAt: newest ? new Date(newest).toISOString() : null,
    missingDpsReport: (row?.missing_dps_report as number) ?? 0,
  });
}
