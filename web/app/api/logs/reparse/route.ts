import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { encounters, logs } from "@/lib/schema";
import { eq } from "drizzle-orm";

// PATCH /api/logs/reparse — re-apply parser output to an existing log, matched by
// filename. Only parser-derived columns are touched; dps_report_url, file_url,
// is_favorite and tags are preserved, and logs not in the database are left
// alone. Used by `gorrik reparse` to correct records after a parser fix.
export async function PATCH(req: NextRequest) {
  const apiKey = req.headers.get("authorization")?.replace("Bearer ", "");
  if (apiKey !== process.env.API_KEY) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const body = await req.json().catch(() => null);
  const {
    filename,
    encounter_id,
    encounter_name,
    category,
    subcategory,
    result,
    mode,
    duration_ms,
    logged_at,
    dry_run,
  } = body ?? {};

  if (typeof filename !== "string" || !filename) {
    return NextResponse.json({ error: "filename required" }, { status: 400 });
  }
  if (
    typeof encounter_name !== "string" ||
    typeof result !== "string" ||
    typeof mode !== "string" ||
    typeof duration_ms !== "number"
  ) {
    return NextResponse.json({ error: "malformed metadata" }, { status: 400 });
  }
  const loggedAt = new Date(logged_at);
  if (isNaN(loggedAt.getTime())) {
    return NextResponse.json({ error: "invalid logged_at" }, { status: 400 });
  }

  const [existing] = await db
    .select({ id: logs.id, result: logs.result })
    .from(logs)
    .where(eq(logs.filename, filename));

  if (!existing) {
    return NextResponse.json({ status: "not_found" });
  }

  const resultChanged = existing.result !== result;

  if (!dry_run) {
    // Keep the encounter reference row satisfied if the encounter changed.
    if (encounter_id && encounter_name) {
      await db
        .insert(encounters)
        .values({ id: encounter_id, name: encounter_name, category, subcategory })
        .onConflictDoNothing();
    }

    await db
      .update(logs)
      .set({
        encounterId: encounter_id ?? null,
        encounterName: encounter_name,
        category,
        subcategory,
        result,
        mode,
        durationMs: duration_ms,
        loggedAt,
      })
      .where(eq(logs.id, existing.id));
  }

  return NextResponse.json({
    status: "updated",
    resultChanged,
    oldResult: existing.result,
    newResult: result,
  });
}
