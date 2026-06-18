"use client";

import { ColumnDef } from "@tanstack/react-table";
import { ExternalLink } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Log } from "@/lib/schema";

function ResultBadge({ result }: { result: string }) {
  const variants: Record<string, string> = {
    success: "bg-emerald-900/50 text-emerald-300 border-emerald-700",
    failure: "bg-red-900/50 text-red-300 border-red-700",
    unknown: "bg-zinc-800 text-zinc-400 border-zinc-600",
  };
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border ${variants[result] ?? variants.unknown}`}
    >
      {result}
    </span>
  );
}

function ModeBadge({ mode }: { mode: string }) {
  if (mode === "normal") return null;
  const labels: Record<string, string> = {
    challenge: "CM",
    quickplay: "QP",
    emboldened1: "EM1",
    emboldened2: "EM2",
    emboldened3: "EM3",
    emboldened4: "EM4",
    emboldened5: "EM5",
  };
  return (
    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-amber-900/50 text-amber-300 border border-amber-700 ml-1.5">
      {labels[mode] ?? mode}
    </span>
  );
}

function formatDuration(ms: number): string {
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  const remainS = s % 60;
  return `${m}:${remainS.toString().padStart(2, "0")}`;
}

function formatDate(date: Date): string {
  return new Date(date).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export const columns: ColumnDef<Log>[] = [
  {
    accessorKey: "encounterName",
    header: "Encounter",
    cell: ({ row }) => (
      <div className="flex items-center gap-0">
        <span className="font-medium">{row.original.encounterName}</span>
        <ModeBadge mode={row.original.mode} />
      </div>
    ),
  },
  {
    accessorKey: "subcategory",
    header: "Wing / Area",
    cell: ({ row }) => (
      <span className="text-muted-foreground text-sm">{row.original.subcategory}</span>
    ),
  },
  {
    accessorKey: "result",
    header: "Result",
    cell: ({ row }) => <ResultBadge result={row.original.result} />,
  },
  {
    accessorKey: "durationMs",
    header: "Duration",
    cell: ({ row }) => (
      <span className="tabular-nums text-sm">
        {formatDuration(row.original.durationMs)}
      </span>
    ),
  },
  {
    accessorKey: "loggedAt",
    header: "Date",
    cell: ({ row }) => (
      <span className="text-muted-foreground text-sm tabular-nums">
        {formatDate(row.original.loggedAt)}
      </span>
    ),
  },
  {
    id: "view",
    header: "",
    cell: ({ row }) => {
      const url = row.original.dpsReportUrl;
      if (!url) return null;
      return (
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="text-muted-foreground hover:text-foreground transition-colors"
          title="View on dps.report"
        >
          <ExternalLink className="h-4 w-4" />
        </a>
      );
    },
  },
];
