#!/usr/bin/env node
// Downloads GW2 profession and elite spec "small colorized" icons from the GW2 wiki
// into web/public/icons/professions/ and web/public/icons/specializations/
import https from "https";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const PROF_DIR = path.join(__dirname, "../web/public/icons/professions");
const SPEC_DIR = path.join(__dirname, "../web/public/icons/specializations");
const BASE = "https://wiki.guildwars2.com";

const PROFESSIONS = [
  ["guardian",     "/images/c/c7/Guardian_icon_small.png"],
  ["warrior",      "/images/4/45/Warrior_icon_small.png"],
  ["engineer",     "/images/0/07/Engineer_icon_small.png"],
  ["ranger",       "/images/1/1e/Ranger_icon_small.png"],
  ["thief",        "/images/a/a0/Thief_icon_small.png"],
  ["elementalist", "/images/4/4e/Elementalist_icon_small.png"],
  ["mesmer",       "/images/7/79/Mesmer_icon_small.png"],
  ["necromancer",  "/images/1/10/Necromancer_icon_small.png"],
  ["revenant",     "/images/4/4c/Revenant_icon_small.png"],
];

const ELITE_SPECS = [
  // Heart of Thorns
  ["dragonhunter",  "/images/5/5d/Dragonhunter_icon_small.png"],
  ["berserker",     "/images/a/a8/Berserker_icon_small.png"],
  ["scrapper",      "/images/7/7d/Scrapper_icon_small.png"],
  ["druid",         "/images/9/9b/Druid_icon_small.png"],
  ["daredevil",     "/images/f/f3/Daredevil_icon_small.png"],
  ["tempest",       "/images/5/58/Tempest_icon_small.png"],
  ["chronomancer",  "/images/e/e0/Chronomancer_icon_small.png"],
  ["reaper",        "/images/9/93/Reaper_icon_small.png"],
  ["herald",        "/images/3/39/Herald_icon_small.png"],
  // Path of Fire
  ["firebrand",     "/images/0/0e/Firebrand_icon_small.png"],
  ["spellbreaker",  "/images/0/08/Spellbreaker_icon_small.png"],
  ["holosmith",     "/images/a/aa/Holosmith_icon_small.png"],
  ["soulbeast",     "/images/6/6a/Soulbeast_icon_small.png"],
  ["deadeye",       "/images/7/70/Deadeye_icon_small.png"],
  ["weaver",        "/images/c/c3/Weaver_icon_small.png"],
  ["mirage",        "/images/c/c8/Mirage_icon_small.png"],
  ["scourge",       "/images/e/e8/Scourge_icon_small.png"],
  ["renegade",      "/images/b/be/Renegade_icon_small.png"],
  // End of Dragons
  ["willbender",    "/images/6/64/Willbender_icon_small.png"],
  ["bladesworn",    "/images/c/cf/Bladesworn_icon_small.png"],
  ["mechanist",     "/images/6/6d/Mechanist_icon_small.png"],
  ["untamed",       "/images/2/2d/Untamed_icon_small.png"],
  ["specter",       "/images/6/61/Specter_icon_small.png"],
  ["catalyst",      "/images/c/c5/Catalyst_icon_small.png"],
  ["virtuoso",      "/images/7/77/Virtuoso_icon_small.png"],
  ["harbinger",     "/images/1/1d/Harbinger_icon_small.png"],
  ["vindicator",    "/images/6/6d/Vindicator_icon_small.png"],
  // Secrets of the Obscure / Visions of Eternity
  ["evoker",        "/images/e/e3/Evoker_icon_small.png"],
  ["troubadour",    "/images/f/f4/Troubadour_icon_small.png"],
  ["ritualist",     "/images/f/f9/Ritualist_icon_small.png"],
  ["amalgam",       "/images/d/d0/Amalgam_icon_small.png"],
  ["galeshot",      "/images/a/a7/Galeshot_icon_small.png"],
  ["antiquary",     "/images/d/d1/Antiquary_icon_small.png"],
  ["luminary",      "/images/7/7b/Luminary_icon_small.png"],
  ["conduit",       "/images/a/a1/Conduit_icon_small.png"],
  ["paragon",       "/images/1/13/Paragon_icon_small.png"],
];

const HEADERS = { "User-Agent": "Mozilla/5.0 (gorrik icon downloader)" };

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https.get(url, { headers: HEADERS }, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        file.close();
        return downloadFile(res.headers.location, dest).then(resolve).catch(reject);
      }
      res.pipe(file);
      file.on("finish", () => file.close(resolve));
    }).on("error", (err) => {
      fs.unlink(dest, () => {});
      reject(err);
    });
  });
}

fs.mkdirSync(PROF_DIR, { recursive: true });
fs.mkdirSync(SPEC_DIR, { recursive: true });

console.log("Downloading profession icons…");
for (const [name, wikPath] of PROFESSIONS) {
  process.stdout.write(`  ${name}… `);
  await downloadFile(BASE + wikPath, path.join(PROF_DIR, `${name}.png`));
  console.log("✓");
}

console.log("Downloading elite spec icons…");
for (const [name, wikPath] of ELITE_SPECS) {
  process.stdout.write(`  ${name}… `);
  await downloadFile(BASE + wikPath, path.join(SPEC_DIR, `${name}.png`));
  console.log("✓");
}

console.log(`\nDone — ${PROFESSIONS.length} professions, ${ELITE_SPECS.length} elite specs.`);
