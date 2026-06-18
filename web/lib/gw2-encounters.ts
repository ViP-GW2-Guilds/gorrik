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
  "Eater of Souls": 42,
  "Eyes of Fate": 43,
  "Broken King": 44,
  Dhuum: 45,
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

// Full static encounter tree — used to show zero-log encounters in the sidebar.
// Keep in sync with agent/parser/encounters.go.
export const ENCOUNTER_DATA: readonly { category: string; subcategory: string; name: string }[] = [
  // Raids
  { category: "raid", subcategory: "Spirit Vale (Wing 1)", name: "Vale Guardian" },
  { category: "raid", subcategory: "Spirit Vale (Wing 1)", name: "Gorseval the Multifarious" },
  { category: "raid", subcategory: "Spirit Vale (Wing 1)", name: "Sabetha the Saboteur" },
  { category: "raid", subcategory: "Salvation Pass (Wing 2)", name: "Slothasor" },
  { category: "raid", subcategory: "Salvation Pass (Wing 2)", name: "Bandit Trio" },
  { category: "raid", subcategory: "Salvation Pass (Wing 2)", name: "Matthias Gabrel" },
  { category: "raid", subcategory: "Stronghold of the Faithful (Wing 3)", name: "Escort" },
  { category: "raid", subcategory: "Stronghold of the Faithful (Wing 3)", name: "Keep Construct" },
  { category: "raid", subcategory: "Stronghold of the Faithful (Wing 3)", name: "Twisted Castle" },
  { category: "raid", subcategory: "Stronghold of the Faithful (Wing 3)", name: "Xera" },
  { category: "raid", subcategory: "Bastion of the Penitent (Wing 4)", name: "Cairn the Indomitable" },
  { category: "raid", subcategory: "Bastion of the Penitent (Wing 4)", name: "Mursaat Overseer" },
  { category: "raid", subcategory: "Bastion of the Penitent (Wing 4)", name: "Samarog" },
  { category: "raid", subcategory: "Bastion of the Penitent (Wing 4)", name: "Deimos" },
  { category: "raid", subcategory: "Hall of Chains (Wing 5)", name: "Soulless Horror" },
  { category: "raid", subcategory: "Hall of Chains (Wing 5)", name: "River of Souls" },
  { category: "raid", subcategory: "Hall of Chains (Wing 5)", name: "Eyes of Fate" },
  { category: "raid", subcategory: "Hall of Chains (Wing 5)", name: "Broken King" },
  { category: "raid", subcategory: "Hall of Chains (Wing 5)", name: "Eater of Souls" },
  { category: "raid", subcategory: "Hall of Chains (Wing 5)", name: "Dhuum" },
  { category: "raid", subcategory: "Mythwright Gambit (Wing 6)", name: "Conjured Amalgamate" },
  { category: "raid", subcategory: "Mythwright Gambit (Wing 6)", name: "Twin Largos" },
  { category: "raid", subcategory: "Mythwright Gambit (Wing 6)", name: "Qadim" },
  { category: "raid", subcategory: "The Key of Ahdashim (Wing 7)", name: "Cardinal Adina" },
  { category: "raid", subcategory: "The Key of Ahdashim (Wing 7)", name: "Cardinal Sabir" },
  { category: "raid", subcategory: "The Key of Ahdashim (Wing 7)", name: "Qadim the Peerless" },
  { category: "raid", subcategory: "Mount Balrior (Wing 8)", name: "Greer, the Blind Furor" },
  { category: "raid", subcategory: "Mount Balrior (Wing 8)", name: "Decima, the Stormsinger" },
  { category: "raid", subcategory: "Mount Balrior (Wing 8)", name: "Ura, the Steamshrieker" },
  // Raid Encounters
  { category: "strike", subcategory: "Icebrood Saga", name: "Shiverpeaks Pass" },
  { category: "strike", subcategory: "Icebrood Saga", name: "Voice of the Fallen and Claw of the Fallen" },
  { category: "strike", subcategory: "Icebrood Saga", name: "Fraenir of Jormag" },
  { category: "strike", subcategory: "Icebrood Saga", name: "Boneskinner" },
  { category: "strike", subcategory: "Icebrood Saga", name: "Whisper of Jormag" },
  { category: "strike", subcategory: "Icebrood Saga", name: "Varinia Stormsounder" },
  { category: "strike", subcategory: "End of Dragons", name: "Aetherblade Hideout" },
  { category: "strike", subcategory: "End of Dragons", name: "Xunlai Jade Junkyard" },
  { category: "strike", subcategory: "End of Dragons", name: "Kaineng Overlook" },
  { category: "strike", subcategory: "End of Dragons", name: "Harvest Temple" },
  { category: "strike", subcategory: "End of Dragons", name: "Old Lion's Court" },
  { category: "strike", subcategory: "Secrets of the Obscure", name: "Cosmic Observatory" },
  { category: "strike", subcategory: "Secrets of the Obscure", name: "Temple of Febe" },
  { category: "strike", subcategory: "Visions of Eternity", name: "Guardian's Glade" },
  // Fractals
  { category: "fractal", subcategory: "Nightmare", name: "MAMA" },
  { category: "fractal", subcategory: "Nightmare", name: "Siax the Corrupted" },
  { category: "fractal", subcategory: "Nightmare", name: "Ensolyss of the Infinite Torment" },
  { category: "fractal", subcategory: "Shattered Observatory", name: "Skorvald the Shattered" },
  { category: "fractal", subcategory: "Shattered Observatory", name: "Artsariiv" },
  { category: "fractal", subcategory: "Shattered Observatory", name: "Arkk" },
  { category: "fractal", subcategory: "Sunqua Peak", name: "Ai, Keeper of the Peak" },
  { category: "fractal", subcategory: "Silent Surf", name: "Kanaxai" },
  { category: "fractal", subcategory: "Lonely Tower", name: "Eparch" },
  { category: "fractal", subcategory: "Kinfall", name: "Whispering Shadow" },
  // Other
  { category: "other", subcategory: "Special Forces Training Area", name: "Standard Kitty Golem" },
  { category: "other", subcategory: "Special Forces Training Area", name: "Medium Kitty Golem" },
  { category: "other", subcategory: "Special Forces Training Area", name: "Large Kitty Golem" },
  { category: "other", subcategory: "Special Forces Training Area", name: "Massive Kitty Golem" },
];
