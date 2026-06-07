import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { encounters, logs, accounts, characters, logPlayers } from "@/lib/schema";
import { and, eq, isNull, or, sql, desc } from "drizzle-orm";

// POST /api/logs — called by the gorrik agent after each upload
export async function POST(req: NextRequest) {
  const apiKey = req.headers.get("authorization")?.replace("Bearer ", "");
  if (apiKey !== process.env.API_KEY) {
    return NextResponse.json(
      { error: "Unauthorized", debug: process.env.API_KEY ? `key_mismatch (len=${process.env.API_KEY.length})` : "key_not_set" },
      { status: 401 }
    );
  }

  const body = await req.json();
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
    file_url,
    players = [],
  } = body;

  // Upsert encounter reference row so foreign key is satisfied.
  if (encounter_id && encounter_name) {
    await db
      .insert(encounters)
      .values({ id: encounter_id, name: encounter_name, category, subcategory })
      .onConflictDoNothing();
  }

  // Insert log — skip silently if filename already exists (re-upload protection).
  const inserted = await db
    .insert(logs)
    .values({
      filename,
      encounterId: encounter_id ?? null,
      encounterName: encounter_name,
      category,
      subcategory,
      result,
      mode,
      durationMs: duration_ms,
      loggedAt: new Date(logged_at),
      fileUrl: file_url,
    })
    .onConflictDoNothing()
    .returning({ id: logs.id });

  if (inserted.length === 0) {
    return NextResponse.json({ status: "duplicate" }, { status: 200 });
  }

  const logId = inserted[0].id;

  // Upsert accounts + characters and create log_players rows.
  for (const player of players) {
    const { account_name, character_name, profession, elite_spec } = player;
    if (!account_name) continue;

    const [account] = await db
      .insert(accounts)
      .values({ accountName: account_name })
      .onConflictDoUpdate({
        target: accounts.accountName,
        set: { accountName: account_name },
      })
      .returning({ id: accounts.id });

    const [character] = await db
      .insert(characters)
      .values({ accountId: account.id, characterName: character_name })
      .onConflictDoUpdate({
        target: [characters.accountId, characters.characterName],
        set: { characterName: character_name },
      })
      .returning({ id: characters.id });

    await db
      .insert(logPlayers)
      .values({
        logId,
        characterId: character.id,
        profession,
        eliteSpec: elite_spec ?? null,
      })
      .onConflictDoNothing();
  }

  return NextResponse.json({ status: "created", id: logId }, { status: 201 });
}

// GET /api/logs — fetch log list with optional filters
export async function GET(req: NextRequest) {
  const { searchParams } = req.nextUrl;
  const category = searchParams.get("category");
  const result = searchParams.get("result");
  const mode = searchParams.get("mode");
  const limit = Math.min(parseInt(searchParams.get("limit") ?? "100"), 500);
  const offset = parseInt(searchParams.get("offset") ?? "0");

  const where = and(
    category ? eq(logs.category, category) : undefined,
    result ? eq(logs.result, result) : undefined,
    mode ? eq(logs.mode, mode) : undefined
  );

  const [rows, [{ count }]] = await Promise.all([
    db
      .select({
        id: logs.id,
        filename: logs.filename,
        encounterName: logs.encounterName,
        category: logs.category,
        subcategory: logs.subcategory,
        result: logs.result,
        mode: logs.mode,
        durationMs: logs.durationMs,
        loggedAt: logs.loggedAt,
        fileUrl: logs.fileUrl,
        isFavorite: logs.isFavorite,
      })
      .from(logs)
      .where(where)
      .orderBy(desc(logs.loggedAt))
      .limit(limit)
      .offset(offset),

    db
      .select({ count: sql<number>`count(*)::int` })
      .from(logs)
      .where(where),
  ]);

  return NextResponse.json({ logs: rows, total: count });
}
