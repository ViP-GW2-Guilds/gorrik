import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { logs } from "@/lib/schema";
import { eq } from "drizzle-orm";

// PATCH /api/logs/[id] — called by the gorrik agent to write back a dps.report permalink
export async function PATCH(
  req: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const apiKey = req.headers.get("authorization")?.replace("Bearer ", "");
  if (apiKey !== process.env.API_KEY) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await params;
  const body = await req.json();
  const { dps_report_url } = body;

  if (typeof dps_report_url !== "string") {
    return NextResponse.json({ error: "dps_report_url required" }, { status: 400 });
  }

  const updated = await db
    .update(logs)
    .set({ dpsReportUrl: dps_report_url })
    .where(eq(logs.id, id))
    .returning({ id: logs.id });

  if (updated.length === 0) {
    return NextResponse.json({ error: "Not found" }, { status: 404 });
  }

  return NextResponse.json({ status: "ok", id: updated[0].id });
}
