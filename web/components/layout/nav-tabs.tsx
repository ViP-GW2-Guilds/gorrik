"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const TABS = [
  { href: "/", label: "Logs", exact: true },
  { href: "/encounters", label: "Encounters", exact: false },
  { href: "/players", label: "Players", exact: false },
];

export function NavTabs() {
  const pathname = usePathname();
  return (
    <nav className="flex items-center gap-1 ml-4">
      {TABS.map(({ href, label, exact }) => {
        const active = exact ? pathname === href : pathname.startsWith(href);
        return (
          <Link
            key={href}
            href={href}
            className={`text-sm px-3 py-1 rounded transition-colors ${
              active
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {label}
          </Link>
        );
      })}
    </nav>
  );
}
