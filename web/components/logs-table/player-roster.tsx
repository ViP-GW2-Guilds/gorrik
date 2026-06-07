"use client";

import Image from "next/image";

export type Player = {
  accountName: string;
  characterName: string;
  profession: string;
  eliteSpec: string | null;
};

function iconPath(profession: string, eliteSpec: string | null): string {
  const slug = (s: string) => s.toLowerCase().replace(/\s+/g, "-");
  if (eliteSpec) return `/icons/specializations/${slug(eliteSpec)}.png`;
  return `/icons/professions/${slug(profession)}.png`;
}

function PlayerEntry({ player }: { player: Player }) {
  return (
    <div className="flex items-center gap-2 py-1">
      <Image
        src={iconPath(player.profession, player.eliteSpec)}
        alt={player.eliteSpec ?? player.profession}
        width={20}
        height={20}
        className="shrink-0"
        unoptimized
      />
      <div className="flex flex-col min-w-0">
        <span className="text-sm leading-tight truncate">{player.characterName}</span>
        <span className="text-xs text-muted-foreground leading-tight truncate">{player.accountName}</span>
      </div>
    </div>
  );
}

export function PlayerRoster({ players }: { players: Player[] | undefined }) {
  if (!players) {
    return (
      <div className="px-6 py-3 text-sm text-muted-foreground italic">
        Loading roster…
      </div>
    );
  }

  if (players.length === 0) {
    return (
      <div className="px-6 py-3 text-sm text-muted-foreground italic">
        No player data recorded for this log.
      </div>
    );
  }

  const col1 = players.slice(0, 5);
  const col2 = players.slice(5, 10);

  return (
    <div className="px-6 py-2 grid grid-cols-2 gap-x-12 border-t border-border/40 bg-card/60">
      <div>{col1.map((p, i) => <PlayerEntry key={i} player={p} />)}</div>
      {col2.length > 0 && (
        <div>{col2.map((p, i) => <PlayerEntry key={i} player={p} />)}</div>
      )}
    </div>
  );
}
