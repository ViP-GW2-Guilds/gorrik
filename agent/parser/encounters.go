package parser

type resultKind int

const (
	resultByDeath                    resultKind = iota // all mainSpecies addresses must receive a ChangeDead or killing-blow event
	resultByAnyDeath                                   // any mainSpecies address receives a ChangeDead or killing-blow event
	resultByReward                                     // a Reward event with a matching rewardID must appear
	resultByBuff895                                    // skill 895 (SoullessHorrorDetermined) applied to the boss
	resultByBuff762                                    // skill 762 (Determined) applied to the boss
	resultByTeamChange                                 // main boss changes team after health drops below 50%
	resultByAttackTargetUntargetable                   // attack target of AttackTargetGadgetSpecies goes targetable→untargetable
	resultByExitCombatAfterSpawn                       // boss exits combat; if ExitCombatMinDelay > 0, must be ≥ that many ms after spawning
	resultByNPCSpawn                                   // a specific NPC species emits a spawn event during combat
	resultBySkillPresent                               // a specific skill ID appears in the log's skill list
	resultUnknown                                      // parser cannot detect result for this encounter
)

type cmKind int

const (
	cmNone      cmKind = iota // only universal emboldened/quickplay detection applies
	cmAlways                  // always challenge mode (e.g. fractal CM-only encounters)
	cmByHealth                // challenge if main boss max health > cmValue
	cmBySkill                 // challenge if skill with ID == cmValue appears in skill list
	cmBySpecies               // challenge if NPC with species ID == cmValue is in agent list
)

type encounterDef struct {
	Name        string
	Category    string // "raid", "strike", "fractal", "other"
	Subcategory string

	// Species IDs of NPCs that constitute the main target(s). All must receive a
	// ChangeDead event for resultByDeath to return "success".
	MainSpecies []int

	ResultKind resultKind
	RewardID   uint64 // for resultByReward

	// For resultByAttackTargetUntargetable: the gadget species whose attack target
	// going targetable→untargetable signals a successful kill (e.g. Deimos = 24660).
	AttackTargetGadgetSpecies int

	// For resultByNPCSpawn: success when an NPC of this species emits a spawn event.
	ResultNPCSpecies int

	// For resultBySkillPresent: success when this skill ID appears in the log's skill list.
	ResultSkillID uint32

	// For resultByExitCombatAfterSpawn: minimum ms between the boss's spawn event
	// and the exit-combat event. Use 0 for no minimum (boss may be pre-placed in
	// the instance and never emit a spawn event).
	ExitCombatMinDelay int64

	CMKind  cmKind
	CMValue int64 // health threshold (cmByHealth), skill ID (cmBySkill), or species ID (cmBySpecies)
}

type resolvedEncounter struct {
	Name        string
	Category    string
	Subcategory string

	// mainBossAddrs maps each MainSpecies ID to the agent addresses found in the
	// log. A species may appear at multiple addresses (NPC respawn / merge artifacts).
	// Success requires at least one address per species to receive a ChangeDead event.
	mainBossAddrs    map[int][]uint64
	resultKind       resultKind
	rewardID         uint64
	cmKind           cmKind
	cmValue          int64
	cmSpeciesPresent bool // set for cmBySpecies encounters when the CM NPC was found

	// attackTargetGadgetAddrs holds addresses of gadget agents whose attack target
	// going targetable→untargetable signals success (resultByAttackTargetUntargetable).
	attackTargetGadgetAddrs []uint64

	// resultNPCAddrs holds agent addresses of ResultNPCSpecies NPCs found in the agent list.
	// Used by resultByNPCSpawn to match spawn events.
	resultNPCAddrs []uint64

	resultSkillID      uint32
	exitCombatMinDelay int64
}

var encountersByTriggerID = buildEncounterTable()

func buildEncounterTable() map[uint16]*encounterDef {
	m := make(map[uint16]*encounterDef, 160)

	add := func(def *encounterDef, ids ...uint16) {
		for _, id := range ids {
			m[id] = def
		}
	}

	// ── Raids ──────────────────────────────────────────────────────────────────

	// Wing 1 — Spirit Vale
	add(&encounterDef{Name: "Vale Guardian", Category: "raid", Subcategory: "Spirit Vale (Wing 1)", MainSpecies: []int{15438}}, 15438)
	// Spirit Race: trigger is the Ethereal Barrier gadget (47188); success is a Reward event (404).
	add(&encounterDef{
		Name: "Spirit Race", Category: "raid", Subcategory: "Spirit Vale (Wing 1)",
		ResultKind: resultByReward, RewardID: 404,
	}, 47188)
	add(&encounterDef{Name: "Gorseval the Multifarious", Category: "raid", Subcategory: "Spirit Vale (Wing 1)", MainSpecies: []int{15429}}, 15429)
	add(&encounterDef{Name: "Sabetha the Saboteur", Category: "raid", Subcategory: "Spirit Vale (Wing 1)", MainSpecies: []int{15375}}, 15375)

	// Wing 2 — Salvation Pass
	add(&encounterDef{Name: "Slothasor", Category: "raid", Subcategory: "Salvation Pass (Wing 2)", MainSpecies: []int{16123}}, 16123)
	add(&encounterDef{
		Name: "Bandit Trio", Category: "raid", Subcategory: "Salvation Pass (Wing 2)",
		// All three must die for success.
		MainSpecies: []int{16088, 16137, 16125},
	}, 16088, 16137, 16125)
	add(&encounterDef{Name: "Matthias Gabrel", Category: "raid", Subcategory: "Salvation Pass (Wing 2)", MainSpecies: []int{16115}}, 16115)

	// Wing 3 — Stronghold of the Faithful
	add(&encounterDef{Name: "Escort", Category: "raid", Subcategory: "Stronghold of the Faithful (Wing 3)", MainSpecies: []int{16253}}, 16253)
	add(&encounterDef{
		Name: "Keep Construct", Category: "raid", Subcategory: "Stronghold of the Faithful (Wing 3)",
		MainSpecies: []int{16235},
		CMKind:      cmBySkill, CMValue: 34958,
	}, 16235)
	add(&encounterDef{
		Name: "Twisted Castle", Category: "raid", Subcategory: "Stronghold of the Faithful (Wing 3)",
		ResultKind: resultByReward, RewardID: 496,
	}, 16247)
	// Phase 1 Xera (16246) teleports away at 50% and is replaced by phase 2 (16286).
	// Phase 2 Xera briefly drops out of combat at the very start of the phase; the kill
	// is detected by her exiting combat ≥10 s after spawning (filters out that false positive).
	add(&encounterDef{Name: "Xera", Category: "raid", Subcategory: "Stronghold of the Faithful (Wing 3)", MainSpecies: []int{16286}, ResultKind: resultByExitCombatAfterSpawn, ExitCombatMinDelay: 10000}, 16246, 16286)

	// Wing 4 — Bastion of the Penitent
	add(&encounterDef{
		Name: "Cairn the Indomitable", Category: "raid", Subcategory: "Bastion of the Penitent (Wing 4)",
		MainSpecies: []int{17194},
		CMKind:      cmBySkill, CMValue: 38098,
	}, 17194)
	add(&encounterDef{
		Name: "Mursaat Overseer", Category: "raid", Subcategory: "Bastion of the Penitent (Wing 4)",
		MainSpecies: []int{17172},
		CMKind:      cmByHealth, CMValue: 29_000_000,
	}, 17172)
	add(&encounterDef{
		Name: "Samarog", Category: "raid", Subcategory: "Bastion of the Penitent (Wing 4)",
		MainSpecies: []int{17188},
		CMKind:      cmByHealth, CMValue: 39_000_000,
	}, 17188)
	// Deimos (NPC 17154) transforms into a gadget (species 24660) at 10% HP and never emits
	// ChangeDead. Success is detected by the gadget's attack target going targetable (phase
	// begins) then untargetable (kill).
	add(&encounterDef{
		Name: "Deimos", Category: "raid", Subcategory: "Bastion of the Penitent (Wing 4)",
		ResultKind:                resultByAttackTargetUntargetable,
		AttackTargetGadgetSpecies: 24660,
		CMKind:                    cmByHealth, CMValue: 42_000_000,
	}, 17154)

	// Wing 5 — Hall of Chains
	// Soulless Horror signals success by applying Determined (895) to itself, not via ChangeDead.
	add(&encounterDef{Name: "Soulless Horror", Category: "raid", Subcategory: "Hall of Chains (Wing 5)", MainSpecies: []int{19767}, ResultKind: resultByBuff895}, 19767)
	add(&encounterDef{
		Name: "River of Souls", Category: "raid", Subcategory: "Hall of Chains (Wing 5)",
		ResultKind: resultUnknown, // Desmina does not die; reward-based but no fixed reward ID
	}, 19828)
	// "Statues of Grenth" is actually three separate encounters in arcdps logs.
	// Eyes: either eye (Judgment=19651, Fate=19844) dying counts as success (AnyCombinedResultDeterminer).
	add(&encounterDef{Name: "Eyes of Fate", Category: "raid", Subcategory: "Hall of Chains (Wing 5)", MainSpecies: []int{19651, 19844}, ResultKind: resultByAnyDeath}, 19651, 19844)
	add(&encounterDef{Name: "Broken King", Category: "raid", Subcategory: "Hall of Chains (Wing 5)", MainSpecies: []int{19691}}, 19691)
	add(&encounterDef{Name: "Eater of Souls", Category: "raid", Subcategory: "Hall of Chains (Wing 5)", MainSpecies: []int{19536}}, 19536)
	add(&encounterDef{Name: "Dhuum", Category: "raid", Subcategory: "Hall of Chains (Wing 5)", MainSpecies: []int{19450}}, 19450)

	// Wing 6 — Mythwright Gambit
	// Trigger is the ConjuredAmalgamate gadget (43974). Success = RoleplayZommoros (21118) spawns.
	// Zommoros is present in the agent list in all logs; we need to detect his spawn event.
	add(&encounterDef{
		Name: "Conjured Amalgamate", Category: "raid", Subcategory: "Mythwright Gambit (Wing 6)",
		ResultKind:       resultByNPCSpawn,
		ResultNPCSpecies: 21118,
	}, 43974)
	add(&encounterDef{
		Name: "Twin Largos", Category: "raid", Subcategory: "Mythwright Gambit (Wing 6)",
		MainSpecies: []int{21105, 21089},
	}, 21105, 21089)
	// Qadim exits combat when killed (no ChangeDead). ExitCombatMinDelay=0 because he is
	// pre-placed and may not emit a spawn event, so any exit combat = success.
	add(&encounterDef{Name: "Qadim", Category: "raid", Subcategory: "Mythwright Gambit (Wing 6)", MainSpecies: []int{20934}, ResultKind: resultByExitCombatAfterSpawn, ExitCombatMinDelay: 0}, 20934)

	// Wing 7 — The Key of Ahdashim
	add(&encounterDef{
		Name: "Cardinal Adina", Category: "raid", Subcategory: "The Key of Ahdashim (Wing 7)",
		MainSpecies: []int{22006},
	}, 22006)
	add(&encounterDef{
		Name: "Cardinal Sabir", Category: "raid", Subcategory: "The Key of Ahdashim (Wing 7)",
		MainSpecies: []int{21964},
	}, 21964)
	add(&encounterDef{
		Name: "Qadim the Peerless", Category: "raid", Subcategory: "The Key of Ahdashim (Wing 7)",
		MainSpecies: []int{22000},
		CMKind:      cmByHealth, CMValue: 50_000_000,
	}, 22000)

	// Wing 8 — Mount Balrior
	add(&encounterDef{Name: "Greer, the Blind Furor", Category: "raid", Subcategory: "Mount Balrior (Wing 8)", MainSpecies: []int{26725}}, 26725, 26859)
	add(&encounterDef{
		Name: "Decima, the Stormsinger", Category: "raid", Subcategory: "Mount Balrior (Wing 8)",
		MainSpecies: []int{26774},
		CMKind:      cmBySpecies, CMValue: 26867,
	}, 26774, 26867)
	add(&encounterDef{Name: "Ura, the Steamshrieker", Category: "raid", Subcategory: "Mount Balrior (Wing 8)", MainSpecies: []int{26712}}, 26712)

	// ── Strikes (Raid Encounters) ───────────────────────────────────────────

	// Icebrood Saga
	add(&encounterDef{Name: "Shiverpeaks Pass", Category: "strike", Subcategory: "Icebrood Saga", MainSpecies: []int{22154}}, 22154)
	add(&encounterDef{
		Name: "Voice of the Fallen and Claw of the Fallen", Category: "strike", Subcategory: "Icebrood Saga",
		// Both must die for success.
		MainSpecies: []int{22343, 22481},
	}, 22343, 22481)
	add(&encounterDef{Name: "Fraenir of Jormag", Category: "strike", Subcategory: "Icebrood Saga", MainSpecies: []int{22492}}, 22492)
	add(&encounterDef{Name: "Boneskinner", Category: "strike", Subcategory: "Icebrood Saga", MainSpecies: []int{22521}}, 22521)
	add(&encounterDef{Name: "Whisper of Jormag", Category: "strike", Subcategory: "Icebrood Saga", MainSpecies: []int{22711}}, 22711)
	add(&encounterDef{Name: "Varinia Stormsounder", Category: "strike", Subcategory: "Icebrood Saga", MainSpecies: []int{22836}}, 22836)

	// End of Dragons
	add(&encounterDef{
		Name: "Aetherblade Hideout", Category: "strike", Subcategory: "End of Dragons",
		MainSpecies: []int{24033},
		ResultKind:  resultByBuff895,
		CMKind:      cmByHealth, CMValue: 8_000_000,
	}, 24033)
	// Xunlai Jade Junkyard and Kaineng Overlook: boss surrenders at ~50% HP via a team-change
	// event rather than dying (ChangeDead never fires).
	add(&encounterDef{
		Name: "Xunlai Jade Junkyard", Category: "strike", Subcategory: "End of Dragons",
		MainSpecies: []int{23957},
		ResultKind:  resultByTeamChange,
		CMKind:      cmByHealth, CMValue: 45_000_000,
	}, 23957)
	add(&encounterDef{
		Name: "Kaineng Overlook", Category: "strike", Subcategory: "End of Dragons",
		MainSpecies: []int{24485},
		ResultKind:  resultByTeamChange,
		CMKind:      cmByHealth, CMValue: 30_000_000,
		// MinisterLiChallengeMode (24266) can also be the trigger ID
	}, 24485, 24266)
	add(&encounterDef{
		Name: "Harvest Temple", Category: "strike", Subcategory: "End of Dragons",
		// Success detected via skill 63896 (HarvestTempleLiftOff) appearing in the skill
		// list — this animation only triggers in the final victory sequence.
		ResultKind:    resultBySkillPresent,
		ResultSkillID: 63896,
	}, 24375, 24223, 43488, 1378)
	add(&encounterDef{
		Name: "Old Lion's Court", Category: "strike", Subcategory: "End of Dragons",
		MainSpecies: []int{25413, 25415, 25419},
		CMKind:      cmBySpecies, CMValue: 25414,
	}, 25413, 25415, 25419, 25414, 25416, 25423)

	// Secrets of the Obscure
	add(&encounterDef{
		Name: "Cosmic Observatory", Category: "strike", Subcategory: "Secrets of the Obscure",
		MainSpecies: []int{25705},
	}, 25705)
	add(&encounterDef{
		Name: "Temple of Febe", Category: "strike", Subcategory: "Secrets of the Obscure",
		MainSpecies: []int{25989},
	}, 25989)

	// Visions of Eternity
	// Guardian's Glade signals success via Determined (762), not ChangeDead.
	add(&encounterDef{
		Name: "Guardian's Glade", Category: "strike", Subcategory: "Visions of Eternity",
		MainSpecies: []int{27124}, ResultKind: resultByBuff762,
	}, 27124)

	// ── Fractals ───────────────────────────────────────────────────────────────

	// Nightmare CM (CM-only encounters — always challenge mode)
	add(&encounterDef{Name: "MAMA", Category: "fractal", Subcategory: "Nightmare", MainSpecies: []int{17021}, CMKind: cmAlways}, 17021)
	add(&encounterDef{Name: "Siax the Corrupted", Category: "fractal", Subcategory: "Nightmare", MainSpecies: []int{17028}, CMKind: cmAlways}, 17028)
	add(&encounterDef{Name: "Ensolyss of the Infinite Torment", Category: "fractal", Subcategory: "Nightmare", MainSpecies: []int{16948}, CMKind: cmAlways}, 16948)

	// Shattered Observatory CM (CM-only encounters)
	add(&encounterDef{
		Name: "Skorvald the Shattered", Category: "fractal", Subcategory: "Shattered Observatory",
		MainSpecies: []int{17632},
		CMKind:      cmByHealth, CMValue: 5_550_000,
	}, 17632)
	add(&encounterDef{Name: "Artsariiv", Category: "fractal", Subcategory: "Shattered Observatory", MainSpecies: []int{17949}, CMKind: cmAlways}, 17949)
	add(&encounterDef{Name: "Arkk", Category: "fractal", Subcategory: "Shattered Observatory", MainSpecies: []int{17759}, CMKind: cmAlways}, 17759)

	// Sunqua Peak
	add(&encounterDef{
		Name: "Ai, Keeper of the Peak", Category: "fractal", Subcategory: "Sunqua Peak",
		ResultKind: resultUnknown, // phase-based; complex detection
		CMKind:     cmAlways,
	}, 23254)

	// Silent Surf
	// Kanaxai signals success via Determined (762), not ChangeDead.
	add(&encounterDef{Name: "Kanaxai", Category: "fractal", Subcategory: "Silent Surf", MainSpecies: []int{25572}, ResultKind: resultByBuff762, CMKind: cmNone}, 25572)
	add(&encounterDef{Name: "Kanaxai", Category: "fractal", Subcategory: "Silent Surf", MainSpecies: []int{25577}, ResultKind: resultByBuff762, CMKind: cmAlways}, 25577)

	// Lonely Tower
	// Eparch applies Determined (895) on successful defeat rather than emitting ChangeDead.
	add(&encounterDef{
		Name: "Eparch", Category: "fractal", Subcategory: "Lonely Tower",
		MainSpecies: []int{26231},
		ResultKind:  resultByBuff895,
		CMKind:      cmByHealth, CMValue: 21_000_000,
	}, 26231)

	// Kinfall
	add(&encounterDef{
		Name: "Whispering Shadow", Category: "fractal", Subcategory: "Kinfall",
		MainSpecies: []int{27010},
		CMKind:      cmBySkill, CMValue: 76339,
	}, 27010)

	// ── Festival ───────────────────────────────────────────────────────────────

	// Freezie signals success via Determined (762), like the fractal Determined bosses.
	add(&encounterDef{Name: "Freezie", Category: "other", Subcategory: "Festival", MainSpecies: []int{21333}, ResultKind: resultByBuff762}, 21333)

	// ── Non-instanced logs ─────────────────────────────────────────────────────
	// arcdps writes a trigger/boss species ID of 1 for World vs World logs and 2 for
	// generic map logs. Neither has a win/loss result.
	add(&encounterDef{Name: "World vs World", Category: "other", Subcategory: "World vs World", ResultKind: resultUnknown}, 1)
	add(&encounterDef{Name: "Map", Category: "other", Subcategory: "Map", ResultKind: resultUnknown}, 2)

	// ── Training Area ──────────────────────────────────────────────────────────

	add(&encounterDef{Name: "Standard Kitty Golem", Category: "other", Subcategory: "Special Forces Training Area", MainSpecies: []int{16199}}, 16199)
	add(&encounterDef{Name: "Medium Kitty Golem", Category: "other", Subcategory: "Special Forces Training Area", MainSpecies: []int{19645}}, 19645)
	add(&encounterDef{Name: "Large Kitty Golem", Category: "other", Subcategory: "Special Forces Training Area", MainSpecies: []int{19676}}, 19676)
	add(&encounterDef{Name: "Massive Kitty Golem", Category: "other", Subcategory: "Special Forces Training Area", MainSpecies: []int{16202}}, 16202)

	return m
}

// lookupEncounter resolves an encounter from the trigger ID and the full agent list.
// It also finds main boss addresses for use in result and mode detection.
func lookupEncounter(triggerID uint16, agents []rawAgent) resolvedEncounter {
	def := encountersByTriggerID[triggerID]

	// Special case: Xera (16246) sometimes appears as the trigger in Twisted Castle logs.
	if triggerID == 16246 {
		for _, a := range agents {
			if isNPC(a) && int(a.prof&0xFFFF) == 16247 { // HauntingStatue
				def = encountersByTriggerID[16247]
				break
			}
		}
	}

	if def == nil {
		return resolvedEncounter{
			Name:        "Unknown",
			Category:    "other",
			Subcategory: "",
			resultKind:  resultUnknown,
		}
	}

	// Collect main boss addresses grouped by species, and check for CM species.
	// A single species can appear at multiple addresses in the agent list due to
	// NPC respawning or arcdps tracking artifacts; we group them so result
	// detection can use "any one dies" semantics per species.
	mainBossAddrs := make(map[int][]uint64, len(def.MainSpecies))
	for _, s := range def.MainSpecies {
		mainBossAddrs[s] = nil // pre-populate so we know which species to expect
	}

	cmSpeciesPresent := false
	var attackTargetGadgetAddrs []uint64
	var resultNPCAddrs []uint64
	for _, a := range agents {
		// Players have isElite != 0xFFFFFFFF; skip them.
		// NPCs and gadgets both have isElite == 0xFFFFFFFF. Some boss kill targets
		// (e.g. Deimos ghost form, Statues of Grenth) are gadget-type agents
		// (prof>>16 == 0xFFFF) that isNPC() excludes. Collect both for mainBossAddrs.
		if a.isElite != 0xFFFFFFFF {
			continue
		}
		speciesID := int(a.prof & 0xFFFF)
		if _, wanted := mainBossAddrs[speciesID]; wanted {
			mainBossAddrs[speciesID] = append(mainBossAddrs[speciesID], a.address)
		}
		// CM species detection is NPC-only; gadgets are environmental objects.
		if isNPC(a) && def.CMKind == cmBySpecies && speciesID == int(def.CMValue) {
			cmSpeciesPresent = true
		}
		// Collect gadget addresses for resultByAttackTargetUntargetable.
		if def.AttackTargetGadgetSpecies != 0 && !isNPC(a) && speciesID == def.AttackTargetGadgetSpecies {
			attackTargetGadgetAddrs = append(attackTargetGadgetAddrs, a.address)
		}
		// Collect addresses of ResultNPCSpecies NPCs for resultByNPCSpawn.
		if def.ResultNPCSpecies != 0 && isNPC(a) && speciesID == def.ResultNPCSpecies {
			resultNPCAddrs = append(resultNPCAddrs, a.address)
		}
	}

	return resolvedEncounter{
		Name:                    def.Name,
		Category:                def.Category,
		Subcategory:             def.Subcategory,
		mainBossAddrs:           mainBossAddrs,
		resultKind:              def.ResultKind,
		rewardID:                def.RewardID,
		cmKind:                  def.CMKind,
		cmValue:                 def.CMValue,
		cmSpeciesPresent:        cmSpeciesPresent,
		attackTargetGadgetAddrs: attackTargetGadgetAddrs,
		resultNPCAddrs:          resultNPCAddrs,
		resultSkillID:           def.ResultSkillID,
		exitCombatMinDelay:      def.ExitCombatMinDelay,
	}
}

func isNPC(a rawAgent) bool {
	return a.isElite == 0xFFFFFFFF && a.prof>>16 != 0xFFFF
}
