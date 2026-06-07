import { Suspense } from "react";
import { NuqsAdapter } from "nuqs/adapters/next/app";
import { db } from "@/lib/db";
import { logs } from "@/lib/schema";
import { desc, eq, and } from "drizzle-orm";
import { Filters } from "@/components/sidebar/filters";
import { LogsTable } from "@/components/logs-table/logs-table";

async function fetchLogs(filters: {
  category?: string;
  result?: string;
  mode?: string;
}) {
  const where = and(
    filters.category ? eq(logs.category, filters.category) : undefined,
    filters.result ? eq(logs.result, filters.result) : undefined,
    filters.mode ? eq(logs.mode, filters.mode) : undefined
  );

  return db.select().from(logs).where(where).orderBy(desc(logs.loggedAt));
}

export default async function Page({
  searchParams,
}: {
  searchParams: Promise<{ category?: string; result?: string; mode?: string }>;
}) {
  const params = await searchParams;
  const allLogs = await fetchLogs(params);

  return (
    <NuqsAdapter>
      <div className="flex h-screen overflow-hidden">
        {/* Header */}
        <div className="absolute top-0 left-0 right-0 h-12 border-b border-border flex items-center px-4 gap-2 bg-background z-20">
          <span className="font-semibold tracking-tight">Gorrik</span>
          <span className="text-muted-foreground text-sm">
            — {allLogs.length.toLocaleString()} logs
          </span>
        </div>

        {/* Body below header */}
        <div className="flex w-full pt-12 h-screen overflow-hidden">
          <Suspense>
            <Filters />
          </Suspense>

          <main className="flex-1 overflow-hidden">
            <LogsTable logs={allLogs} />
          </main>
        </div>
      </div>
    </NuqsAdapter>
  );
}
