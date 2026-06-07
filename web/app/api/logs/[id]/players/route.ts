import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { logPlayers, characters, accounts } from "@/lib/schema";
import { eq } from "drizzle-orm";

export async function GET(
  _req: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;

  const rows = await db
    .select({
      accountName: accounts.accountName,
      characterName: characters.characterName,
      profession: logPlayers.profession,
      eliteSpec: logPlayers.eliteSpec,
    })
    .from(logPlayers)
    .innerJoin(characters, eq(logPlayers.characterId, characters.id))
    .innerJoin(accounts, eq(characters.accountId, accounts.id))
    .where(eq(logPlayers.logId, id));

  return NextResponse.json({ players: rows });
}
