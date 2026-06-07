"use client";

import { useQueryState } from "nuqs";

const RESULTS = [
  { value: "success", label: "Success" },
  { value: "failure", label: "Failure" },
  { value: "unknown", label: "Unknown" },
];

const MODES = [
  { value: "normal", label: "Normal" },
  { value: "challenge", label: "CM" },
  { value: "quickplay", label: "Quickplay" },
  { value: "emboldened", label: "Emboldened" },
];

function ChipGroup({
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
    <div className="flex items-center gap-2">
      <span className="text-xs font-semibold uppercase tracking-wider text-[#D4A853] shrink-0">
        {label}
      </span>
      <div className="flex flex-wrap gap-1">
        {options.map((opt) => (
          <button
            key={opt.value}
            onClick={() => onChange(value === opt.value ? null : opt.value)}
            className={`px-2 py-0.5 rounded text-xs font-medium border transition-colors ${
              value === opt.value
                ? "bg-[#D4A853] text-[#1a1400] border-[#D4A853]"
                : "border-border hover:border-muted-foreground text-muted-foreground hover:text-foreground"
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>
    </div>
  );
}

export function FilterBar() {
  const [result, setResult] = useQueryState("result", { shallow: false });
  const [mode, setMode] = useQueryState("mode", { shallow: false });

  return (
    <div className="flex items-center gap-6 px-4 py-2 border-b border-border flex-wrap shrink-0 bg-accent">
      <ChipGroup label="Result" options={RESULTS} value={result} onChange={setResult} />
      <ChipGroup label="Mode" options={MODES} value={mode} onChange={setMode} />
    </div>
  );
}
