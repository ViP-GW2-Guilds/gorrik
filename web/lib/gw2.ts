export function iconPath(profession: string, eliteSpec: string | null | undefined): string {
  const slug = (s: string) => s.toLowerCase().replace(/\s+/g, "-");
  if (eliteSpec) return `/icons/specializations/${slug(eliteSpec)}.png`;
  return `/icons/professions/${slug(profession)}.png`;
}

export function formatDuration(ms: number): string {
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

export function slugify(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}
