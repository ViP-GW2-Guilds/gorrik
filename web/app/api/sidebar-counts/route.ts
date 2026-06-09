import { NextResponse } from "next/server";
import { db } from "@/lib/db";
import { sql } from "drizzle-orm";

export async function GET() {
  const result = await db.execute(sql`
    SELECT category, subcategory, encounter_name, COUNT(*)::int AS count
    FROM logs
    GROUP BY category, subcategory, encounter_name
  `);

  return NextResponse.json(
    result.rows.map((r) => ({
      category: r.category as string,
      subcategory: r.subcategory as string,
      encounterName: r.encounter_name as string,
      count: (r.count as number) ?? 0,
    }))
  );
}
