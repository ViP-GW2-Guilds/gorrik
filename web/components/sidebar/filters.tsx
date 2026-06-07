"use client";

import { useQueryState } from "nuqs";

const CATEGORIES = [
  { value: "raid", label: "Raids" },
  { value: "strike", label: "Raid Encounters" },
  { value: "fractal", label: "Fractals" },
  { value: "other", label: "Other" },
];

const RESULTS = [
  { value: "success", label: "Success" },
  { value: "failure", label: "Failure" },
  { value: "unknown", label: "Unknown" },
];

const MODES = [
  { value: "normal", label: "Normal" },
  { value: "challenge", label: "Challenge" },
  { value: "emboldened1", label: "Emboldened 1" },
  { value: "emboldened2", label: "Emboldened 2" },
  { value: "emboldened3", label: "Emboldened 3" },
  { value: "emboldened4", label: "Emboldened 4" },
  { value: "emboldened5", label: "Emboldened 5" },
  { value: "quickplay", label: "Quickplay" },
];

function FilterGroup({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: { value: string; label: string }[];
  value: string | null;
  onChange: (v: string | null) => void;
}) {
  return (
    <div className="space-y-1">
      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground px-1">
        {label}
      </p>
      {options.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(value === opt.value ? null : opt.value)}
          className={`w-full text-left text-sm px-2 py-1 rounded transition-colors ${
            value === opt.value
              ? "bg-primary text-primary-foreground"
              : "hover:bg-muted"
          }`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}

export function Filters() {
  const [category, setCategory] = useQueryState("category");
  const [result, setResult] = useQueryState("result");
  const [mode, setMode] = useQueryState("mode");

  const hasFilters = category || result || mode;

  return (
    <aside className="w-48 shrink-0 flex flex-col gap-6 p-4 border-r border-border overflow-y-auto">
      <div className="flex items-center justify-between">
        <span className="font-semibold text-sm">Filters</span>
        {hasFilters && (
          <button
            onClick={() => {
              setCategory(null);
              setResult(null);
              setMode(null);
            }}
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            Clear
          </button>
        )}
      </div>

      <FilterGroup
        label="Category"
        options={CATEGORIES}
        value={category}
        onChange={setCategory}
      />
      <FilterGroup
        label="Result"
        options={RESULTS}
        value={result}
        onChange={setResult}
      />
      <FilterGroup
        label="Mode"
        options={MODES}
        value={mode}
        onChange={setMode}
      />
    </aside>
  );
}
