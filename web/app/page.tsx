import { Suspense } from "react";
import { NuqsAdapter } from "nuqs/adapters/next/app";
import { db } from "@/lib/db";
import { logs } from "@/lib/schema";
import { desc, eq, and, like } from "drizzle-orm";
import { EncounterTree } from "@/components/sidebar/encounter-tree";
import { FilterBar } from "@/components/logs-table/filter-bar";
import { LogsTable } from "@/components/logs-table/logs-table";
import { NavTabs } from "@/components/layout/nav-tabs";

async function fetchLogs(filters: {
  category?: string;
  subcategory?: string;
  encounter?: string;
  result?: string;
  mode?: string;
}) {
  const where = and(
    filters.category ? eq(logs.category, filters.category) : undefined,
    filters.subcategory ? eq(logs.subcategory, filters.subcategory) : undefined,
    filters.encounter ? eq(logs.encounterName, filters.encounter) : undefined,
    filters.result ? eq(logs.result, filters.result) : undefined,
    filters.mode
      ? filters.mode === "emboldened"
        ? like(logs.mode, "emboldened%")
        : eq(logs.mode, filters.mode)
      : undefined
  );

  return db.select().from(logs).where(where).orderBy(desc(logs.loggedAt));
}

export default async function Page({
  searchParams,
}: {
  searchParams: Promise<{
    category?: string;
    subcategory?: string;
    encounter?: string;
    result?: string;
    mode?: string;
  }>;
}) {
  const params = await searchParams;
  const allLogs = await fetchLogs(params);

  return (
    <NuqsAdapter>
      <div className="flex h-screen overflow-hidden">
        {/* Header */}
        <div className="absolute top-0 left-0 right-0 h-12 border-b border-border flex items-center px-4 gap-2 bg-background z-20">
          <span className="font-semibold tracking-tight">Gorrik</span>
          <NavTabs />
          <span className="ml-auto text-muted-foreground text-sm">
            {allLogs.length.toLocaleString()} logs
          </span>
        </div>

        {/* Body below header */}
        <div className="flex w-full pt-12 h-screen overflow-hidden">
          <Suspense>
            <EncounterTree mode="filter" />
          </Suspense>

          <main className="flex-1 overflow-hidden flex flex-col">
            <Suspense>
              <FilterBar />
            </Suspense>
            <div className="flex-1 overflow-hidden">
              <LogsTable logs={allLogs} />
            </div>
          </main>
        </div>
      </div>
    </NuqsAdapter>
  );
}
