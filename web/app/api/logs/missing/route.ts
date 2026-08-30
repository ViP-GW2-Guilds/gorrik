import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { logs } from "@/lib/schema";
import { inArray } from "drizzle-orm";

// POST /api/logs/missing — given a list of local log filenames, return the
// subset that is not yet indexed. Used by the gorrik agent's `status` command
// to report how far behind the database is.
export async function POST(req: NextRequest) {
  const apiKey = req.headers.get("authorization")?.replace("Bearer ", "");
  if (apiKey !== process.env.API_KEY) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const body = await req.json().catch(() => null);
  const filenames: unknown = body?.filenames;
  if (!Array.isArray(filenames) || filenames.some((f) => typeof f !== "string")) {
    return NextResponse.json(
      { error: "filenames must be an array of strings" },
      { status: 400 }
    );
  }
  if (filenames.length === 0) {
    return NextResponse.json({ missing: [] });
  }

  const known = new Set(
    (
      await db
        .select({ filename: logs.filename })
        .from(logs)
        .where(inArray(logs.filename, filenames as string[]))
    ).map((r) => r.filename)
  );

  const missing = (filenames as string[]).filter((f) => !known.has(f));
  return NextResponse.json({ missing });
}
