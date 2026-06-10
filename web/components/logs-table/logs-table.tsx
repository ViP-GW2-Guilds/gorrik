"use client";

import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
  type SortingState,
} from "@tanstack/react-table";
import { ChevronUp, ChevronDown, ChevronsUpDown } from "lucide-react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useRef, useState, useCallback, useEffect, Fragment } from "react";
import { columns } from "./columns";
import { PlayerRoster, type Player } from "./player-roster";
import { Log } from "@/lib/schema";

// Estimated heights — roster is 5 entries × ~36px + 16px padding
const BASE_ROW_HEIGHT = 44;
const EXPANDED_ROW_HEIGHT = 44 + 196;

export function LogsTable({ logs }: { logs: Log[] }) {
  const parentRef = useRef<HTMLDivElement>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [playerCache, setPlayerCache] = useState<Record<string, Player[]>>({});
  const [sorting, setSorting] = useState<SortingState>([]);

  const table = useReactTable({
    data: logs,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    state: { sorting },
    onSortingChange: setSorting,
  });

  const { rows } = table.getRowModel();

  const estimateSize = useCallback(
    (index: number) =>
      expanded.has(rows[index]?.original.id)
        ? EXPANDED_ROW_HEIGHT
        : BASE_ROW_HEIGHT,
    [rows, expanded]
  );

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize,
    overscan: 20,
  });

  // Flush virtualizer size cache whenever expansion state changes
  useEffect(() => {
    virtualizer.measure();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expanded]);

  const toggleExpand = useCallback(
    async (logId: string) => {
      const wasExpanded = expanded.has(logId);

      setExpanded((prev) => {
        const next = new Set(prev);
        if (next.has(logId)) next.delete(logId);
        else next.add(logId);
        return next;
      });

      if (!wasExpanded && !playerCache[logId]) {
        try {
          const res = await fetch(`/api/logs/${logId}/players`);
          const data = await res.json();
          setPlayerCache((prev) => ({ ...prev, [logId]: data.players }));
        } catch {
          // roster stays in "Loading…" state — user can collapse and retry
        }
      }
    },
    [expanded, playerCache]
  );

  const virtualRows = virtualizer.getVirtualItems();
  const totalHeight = virtualizer.getTotalSize();
  const paddingTop = virtualRows.length > 0 ? virtualRows[0].start : 0;
  const paddingBottom =
    virtualRows.length > 0
      ? totalHeight - virtualRows[virtualRows.length - 1].end
      : 0;

  if (logs.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-2">
        <p className="text-lg font-medium">No logs found</p>
        <p className="text-sm">Run gorrik import to index your existing logs</p>
      </div>
    );
  }

  return (
    <div ref={parentRef} className="overflow-auto h-full">
      <table className="w-full text-sm border-collapse">
        <thead className="sticky top-0 bg-background border-b border-border z-10">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => {
                const sorted = header.column.getIsSorted();
                const canSort = header.column.getCanSort();
                return (
                  <th
                    key={header.id}
                    onClick={canSort ? header.column.getToggleSortingHandler() : undefined}
                    className={`text-left px-3 py-2.5 font-medium text-xs uppercase tracking-wider whitespace-nowrap ${
                      canSort
                        ? "cursor-pointer select-none text-primary/80 hover:text-primary"
                        : "text-primary/80"
                    }`}
                  >
                    <div className="flex items-center gap-1">
                      {flexRender(header.column.columnDef.header, header.getContext())}
                      {canSort && (
                        sorted === "asc" ? (
                          <ChevronUp className="w-3 h-3" />
                        ) : sorted === "desc" ? (
                          <ChevronDown className="w-3 h-3" />
                        ) : (
                          <ChevronsUpDown className="w-3 h-3 opacity-40" />
                        )
                      )}
                    </div>
                  </th>
                );
              })}
            </tr>
          ))}
        </thead>
        <tbody>
          {paddingTop > 0 && (
            <tr><td style={{ height: paddingTop }} /></tr>
          )}
          {virtualRows.map((virtualRow) => {
            const row = rows[virtualRow.index];
            const logId = row.original.id;
            const isExpanded = expanded.has(logId);

            return (
              <Fragment key={row.id}>
                <tr
                  onClick={() => toggleExpand(logId)}
                  className={`border-b border-border/50 cursor-pointer transition-colors ${
                    isExpanded ? "bg-muted/60" : "hover:bg-muted/40"
                  }`}
                >
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="px-3 py-2.5 whitespace-nowrap">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
                {isExpanded && (
                  <tr>
                    <td colSpan={columns.length} className="p-0">
                      <PlayerRoster players={playerCache[logId]} />
                    </td>
                  </tr>
                )}
              </Fragment>
            );
          })}
          {paddingBottom > 0 && (
            <tr><td style={{ height: paddingBottom }} /></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
