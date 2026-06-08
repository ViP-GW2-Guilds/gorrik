"use client";

import { useState, useCallback, Fragment } from "react";
import Image from "next/image";
import { iconPath } from "@/lib/gw2";
import type { SpecEntry } from "./encounter-table";

export type PlayerStat = {
  accountId: string;
  accountName: string;
  totalLogs: number;
  kills: number;
  topSpecs: SpecEntry[];
};

export type CharacterStat = {
  characterId: string;
  characterName: string;
  totalLogs: number;
  kills: number;
  topSpec: SpecEntry | null;
};

function SuccessRate({ total, kills }: { total: number; kills: number }) {
  const pct = total > 0 ? Math.round((kills / total) * 100) : 0;
  return (
    <span>
      {kills}/{total}{" "}
      <span className="text-muted-foreground">({pct}%)</span>
    </span>
  );
}

function SpecIcons({ specs }: { specs: SpecEntry[] }) {
  return (
    <div className="flex items-center gap-1">
      {specs.map((s, i) => (
        <Image
          key={i}
          src={iconPath(s.profession, s.eliteSpec)}
          alt={s.eliteSpec ?? s.profession}
          width={20}
          height={20}
          unoptimized
          title={s.eliteSpec ?? s.profession}
        />
      ))}
    </div>
  );
}

function CharacterBreakdown({
  characters,
}: {
  characters: CharacterStat[] | undefined;
}) {
  if (!characters) {
    return (
      <tr>
        <td colSpan={5} className="px-8 py-3 text-sm text-muted-foreground italic border-b border-border/30 bg-muted/20">
          Loading characters…
        </td>
      </tr>
    );
  }

  if (characters.length === 0) {
    return (
      <tr>
        <td colSpan={5} className="px-8 py-3 text-sm text-muted-foreground italic border-b border-border/30 bg-muted/20">
          No character data found.
        </td>
      </tr>
    );
  }

  return (
    <>
      {characters.map((char) => (
        <tr key={char.characterId} className="border-b border-border/30 bg-muted/20">
          <td className="px-8 py-2 text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              {char.topSpec && (
                <Image
                  src={iconPath(char.topSpec.profession, char.topSpec.eliteSpec)}
                  alt={char.topSpec.eliteSpec ?? char.topSpec.profession}
                  width={16}
                  height={16}
                  unoptimized
                  title={char.topSpec.eliteSpec ?? char.topSpec.profession}
                />
              )}
              {char.characterName}
            </div>
          </td>
          <td className="px-3 py-2 text-sm" colSpan={2}>
            <SuccessRate total={char.totalLogs} kills={char.kills} />
          </td>
          <td colSpan={2} />
        </tr>
      ))}
    </>
  );
}

export function PlayerTable({ players }: { players: PlayerStat[] }) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [charCache, setCharCache] = useState<Record<string, CharacterStat[]>>({});

  const toggleExpand = useCallback(
    async (accountId: string) => {
      const wasExpanded = expanded.has(accountId);

      setExpanded((prev) => {
        const next = new Set(prev);
        if (next.has(accountId)) next.delete(accountId);
        else next.add(accountId);
        return next;
      });

      if (!wasExpanded && !charCache[accountId]) {
        try {
          const res = await fetch(`/api/stats/players/${accountId}/characters`);
          const data = await res.json();
          setCharCache((prev) => ({ ...prev, [accountId]: data.characters }));
        } catch {
          // stays in loading state
        }
      }
    },
    [expanded, charCache]
  );

  if (players.length === 0) {
    return <p className="text-muted-foreground text-sm">No player data yet.</p>;
  }

  return (
    <table className="w-full text-sm border-collapse">
      <thead>
        <tr className="border-b border-border">
          <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider w-56">Account</th>
          <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider w-28">Logs</th>
          <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider w-32">Kills</th>
          <th className="text-left px-3 py-2 text-xs font-medium text-primary/80 uppercase tracking-wider">Top Specs</th>
          <th className="w-8" />
        </tr>
      </thead>
      <tbody>
        {players.map((player) => {
          const isExpanded = expanded.has(player.accountId);
          return (
            <Fragment key={player.accountId}>
              <tr
                onClick={() => toggleExpand(player.accountId)}
                className={`border-b border-border/50 cursor-pointer transition-colors ${
                  isExpanded ? "bg-muted/60" : "hover:bg-muted/40"
                }`}
              >
                <td className="px-3 py-2.5 font-medium">{player.accountName.replace(/^:/, "")}</td>
                <td className="px-3 py-2.5 tabular-nums">{player.totalLogs}</td>
                <td className="px-3 py-2.5">
                  <SuccessRate total={player.totalLogs} kills={player.kills} />
                </td>
                <td className="px-3 py-2.5">
                  <SpecIcons specs={player.topSpecs} />
                </td>
                <td className="px-3 py-2.5 text-muted-foreground text-xs">
                  {isExpanded ? "▲" : "▼"}
                </td>
              </tr>
              {isExpanded && (
                <CharacterBreakdown
                  characters={charCache[player.accountId]}
                />
              )}
            </Fragment>
          );
        })}
      </tbody>
    </table>
  );
}
