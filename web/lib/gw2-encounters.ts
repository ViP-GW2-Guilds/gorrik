// Canonical subcategory display order (mirrors encounters.go addition order).
// Unknown subcategories sort to the end.
export const SUBCATEGORY_ORDER: readonly string[] = [
  // Raids — Wings 1–8
  "Spirit Vale (Wing 1)",
  "Salvation Pass (Wing 2)",
  "Stronghold of the Faithful (Wing 3)",
  "Bastion of the Penitent (Wing 4)",
  "Hall of Chains (Wing 5)",
  "Mythwright Gambit (Wing 6)",
  "The Key of Ahdashim (Wing 7)",
  "Mount Balrior (Wing 8)",
  // Raid Encounters
  "Icebrood Saga",
  "End of Dragons",
  "Secrets of the Obscure",
  "Visions of Eternity",
  // Fractals
  "Nightmare",
  "Shattered Observatory",
  "Sunqua Peak",
  "Silent Surf",
  "Lonely Tower",
  "Kinfall",
  // Other
  "Special Forces Training Area",
];

// Canonical encounter display order within each subcategory.
// Maps encounter name → absolute sort index (unique globally).
// Encounters not listed here sort alphabetically after known ones.
export const ENCOUNTER_ORDER: Readonly<Record<string, number>> = {
  // Spirit Vale (Wing 1)
  "Vale Guardian": 0,
  "Gorseval the Multifarious": 1,
  "Sabetha the Saboteur": 2,
  // Salvation Pass (Wing 2)
  Slothasor: 10,
  "Bandit Trio": 11,
  "Matthias Gabrel": 12,
  // Stronghold of the Faithful (Wing 3)
  Escort: 20,
  "Keep Construct": 21,
  "Twisted Castle": 22,
  Xera: 23,
  // Bastion of the Penitent (Wing 4)
  "Cairn the Indomitable": 30,
  "Mursaat Overseer": 31,
  Samarog: 32,
  Deimos: 33,
  // Hall of Chains (Wing 5)
  "Soulless Horror": 40,
  "River of Souls": 41,
  "Statues of Grenth": 42,
  Dhuum: 43,
  // Mythwright Gambit (Wing 6)
  "Conjured Amalgamate": 50,
  "Twin Largos": 51,
  Qadim: 52,
  // The Key of Ahdashim (Wing 7)
  "Cardinal Adina": 60,
  "Cardinal Sabir": 61,
  "Qadim the Peerless": 62,
  // Mount Balrior (Wing 8)
  "Greer, the Blind Furor": 70,
  "Decima, the Stormsinger": 71,
  "Ura, the Steamshrieker": 72,
  // Icebrood Saga
  "Shiverpeaks Pass": 80,
  "Voice of the Fallen and Claw of the Fallen": 81,
  "Fraenir of Jormag": 82,
  Boneskinner: 83,
  "Whisper of Jormag": 84,
  "Varinia Stormsounder": 85,
  // End of Dragons
  "Aetherblade Hideout": 90,
  "Xunlai Jade Junkyard": 91,
  "Kaineng Overlook": 92,
  "Harvest Temple": 93,
  "Old Lion's Court": 94,
  // Secrets of the Obscure
  "Cosmic Observatory": 100,
  "Temple of Febe": 101,
  // Visions of Eternity
  "Guardian's Glade": 110,
  // Nightmare
  MAMA: 120,
  "Siax the Corrupted": 121,
  "Ensolyss of the Infinite Torment": 122,
  // Shattered Observatory
  "Skorvald the Shattered": 130,
  Artsariiv: 131,
  Arkk: 132,
  // Sunqua Peak
  "Ai, Keeper of the Peak": 140,
  // Silent Surf
  Kanaxai: 150,
  // Lonely Tower
  Eparch: 160,
  // Kinfall
  "Whispering Shadow": 170,
  // Special Forces Training Area
  "Standard Kitty Golem": 180,
  "Medium Kitty Golem": 181,
  "Large Kitty Golem": 182,
  "Massive Kitty Golem": 183,
};

export const CATEGORY_LABELS: Readonly<Record<string, string>> = {
  raid: "Raids",
  strike: "Raid Encounters",
  fractal: "Fractals",
  other: "Other",
};

export const CATEGORY_ORDER: readonly string[] = ["raid", "strike", "fractal", "other"];

export function subcategoryIndex(sub: string): number {
  const i = SUBCATEGORY_ORDER.indexOf(sub);
  return i === -1 ? SUBCATEGORY_ORDER.length : i;
}

export function encounterIndex(name: string): number {
  return ENCOUNTER_ORDER[name] ?? 9999;
}
