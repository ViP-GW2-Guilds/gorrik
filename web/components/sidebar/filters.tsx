"use client";

import { useQueryState } from "nuqs";

const CATEGORIES = [
  { value: "raid", label: "Raids" },
  { value: "strike", label: "Raid Encounters" },
  { value: "fractal", label: "Fractals" },
  { value: "other", label: "Other" },
];

export function Filters() {
  const [category, setCategory] = useQueryState("category");

  return (
    <aside className="w-48 shrink-0 flex flex-col gap-4 p-4 border-r border-border overflow-y-auto">
      <span className="font-semibold text-sm">Category</span>
      <div className="space-y-1">
        {CATEGORIES.map((opt) => (
          <button
            key={opt.value}
            onClick={() => setCategory(category === opt.value ? null : opt.value)}
            className={`w-full text-left text-sm px-2 py-1 rounded transition-colors ${
              category === opt.value
                ? "bg-primary text-primary-foreground"
                : "hover:bg-muted"
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>
    </aside>
  );
}
