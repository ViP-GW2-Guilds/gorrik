package parser

import (
	"archive/zip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// LogMetadata holds the extracted metadata from a single EVTC log file.
type LogMetadata struct {
	EncounterID   uint16
	EncounterName string
	Category      string // "raid", "strike", "fractal", "other"
	Subcategory   string
	Result        string // "success", "failure", "unknown"
	Mode          string // "normal", "challenge", "emboldened1"–"emboldened5", "quickplay"
	DurationMs    int64
	LoggedAt      time.Time
	Players       []PlayerEntry
}

// PlayerEntry holds per-player metadata extracted from the agent list.
type PlayerEntry struct {
	AccountName   string
	CharacterName string
	Profession    string
	EliteSpec     string
}

// State change values from the arcdps cbtstatechange enum.
const (
	scNormal       uint8 = 0
	scEnterCombat  uint8 = 1
	scExitCombat   uint8 = 2
	scChangeDead   uint8 = 4
	scSpawn        uint8 = 6
	scHealthUpdate uint8 = 8 // dstAgent = health fraction * 10000 (10000 = 100%)
	scLogStart     uint8 = 9
	scLogEnd       uint8 = 10
	scMaxHealth    uint8 = 12
	scReward       uint8 = 17
	scBuffInitial  uint8 = 18
	scTeamChange   uint8 = 22 // agent switches affiliation (e.g. boss surrenders); dstAgent = new team ID
	scAttackTarget uint8 = 23 // links attack target to parent gadget
	scTargetable   uint8 = 24 // agent becomes targetable/untargetable (dstAgent != 0 = targetable)
	// arcdps builds from ~2026-05-01 emit buff applications as this dedicated
	// statechange instead of a normal (statechange 0) event with buff != 0.
	scBuffApply uint8 = 69
)

// result field values for scNormal (damage) events.
const resultKillingBlow uint8 = 8

// Skill IDs used for mode detection.
const (
	skillEmboldened     uint32 = 68087
	skillQuickplayBoost uint32 = 77676 // fractals
	skillQuickplayMoral uint32 = 79492 // raids
	skillDetermined762  uint32 = 762
	skillDetermined895  uint32 = 895
)

// Profession names indexed by (prof field value − 1).
var professionNames = []string{
	"Guardian", "Warrior", "Engineer", "Ranger", "Thief",
	"Elementalist", "Mesmer", "Necromancer", "Revenant",
}

// Elite specialisation names keyed by IsElite field value.
var eliteSpecNames = map[uint32]string{
	5: "Druid", 7: "Daredevil", 18: "Berserker", 27: "Dragonhunter",
	34: "Reaper", 40: "Chronomancer", 43: "Scrapper", 48: "Tempest",
	52: "Herald", 55: "Soulbeast", 56: "Weaver", 57: "Holosmith",
	58: "Deadeye", 59: "Mirage", 60: "Scourge", 61: "Spellbreaker",
	62: "Firebrand", 63: "Renegade", 64: "Harbinger", 65: "Willbender",
	66: "Virtuoso", 67: "Catalyst", 68: "Bladesworn", 69: "Vindicator",
	70: "Mechanist", 71: "Specter", 72: "Untamed", 73: "Troubadour",
	74: "Paragon", 75: "Amalgam", 76: "Ritualist", 77: "Antiquary",
	78: "Galeshot", 79: "Conduit",
}

// Heart of Thorns elite spec (IsElite == 1), keyed by base profession name.
var hotSpecByProfession = map[string]string{
	"Guardian": "Dragonhunter", "Warrior": "Berserker", "Engineer": "Scrapper",
	"Ranger": "Druid", "Thief": "Daredevil", "Elementalist": "Tempest",
	"Mesmer": "Chronomancer", "Necromancer": "Reaper", "Revenant": "Herald",
}

// rawAgent is the parsed form of an evtc_agent struct entry.
type rawAgent struct {
	address uint64
	prof    uint32
	isElite uint32
	name    string // raw 68-byte field; null-separated for players: charName\0accountName\0subgroup
}

// combatItem holds only the fields of a cbtevent struct that the parser needs.
type combatItem struct {
	time         int64
	srcAgent     uint64
	dstAgent     uint64
	value        int32
	skillID      uint32
	result       uint8 // for scNormal events: 8 = killing blow
	buff         uint8
	buffRemove   uint8
	isActivation uint8
	stateChange  uint8
}

// Parse parses a .evtc, .evtc.zip, or .zevtc file and returns its metadata.
func Parse(filename string) (meta *LogMetadata, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("malformed file: %v", r)
		}
	}()
	data, err := readRaw(filename)
	if err != nil {
		return nil, err
	}
	return parseBytes(data)
}

func readRaw(filename string) ([]byte, error) {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".zevtc") {
		return readZipped(filename)
	}
	return os.ReadFile(filename)
}

func readZipped(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return nil, err
	}
	if len(zr.File) == 0 {
		return nil, fmt.Errorf("empty zip archive")
	}
	rc, err := zr.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func parseBytes(data []byte) (*LogMetadata, error) {
	r := &byteReader{b: data}

	// ── Header ────────────────────────────────────────────────────────────────
	// 12 bytes: arcdps build version string
	r.skip(12)
	// 1 byte: revision (0 or 1)
	revision := r.readU8()
	if revision > 1 {
		return nil, fmt.Errorf("unsupported EVTC revision: %d", revision)
	}
	// 2 bytes: trigger/boss species ID
	triggerID := r.readU16()
	// 1 byte: padding
	r.skip(1)

	// ── Agents ────────────────────────────────────────────────────────────────
	agentCount := int(r.readI32())
	agents := make([]rawAgent, agentCount)
	for i := range agents {
		agents[i] = r.readAgent()
	}

	// ── Skills ────────────────────────────────────────────────────────────────
	skillCount := int(r.readI32())
	skillIDs := make(map[uint32]bool, skillCount)
	for i := 0; i < skillCount; i++ {
		id := uint32(r.readI32())
		r.skip(64) // skill name, not needed
		skillIDs[id] = true
	}

	// ── Encounter identification ───────────────────────────────────────────────
	enc := lookupEncounter(triggerID, agents)

	// Flat set of all main boss addresses — used to filter events to the boss only.
	allBossAddrs := make(map[uint64]bool)
	for _, addrs := range enc.mainBossAddrs {
		for _, addr := range addrs {
			allBossAddrs[addr] = true
		}
	}

	// Set of result-NPC addresses for resultByNPCSpawn (e.g. Zommoros in CA).
	resultNPCAddrSet := make(map[uint64]bool, len(enc.resultNPCAddrs))
	for _, addr := range enc.resultNPCAddrs {
		resultNPCAddrSet[addr] = true
	}

	// Pre-compute whether the result skill is present in the skill list.
	resultSkillPresent := enc.resultSkillID != 0 && skillIDs[enc.resultSkillID]

	// ── Combat event scan ─────────────────────────────────────────────────────
	var (
		logStartTime        int64
		logEndTime          int64
		logStartUnix        int32
		bossDeaths          = make(map[uint64]bool) // includes killing blows
		rewardIDs           = make(map[uint64]bool) // all reward IDs seen (a log may emit several)
		emboldenedCount     = make(map[uint64]int)  // agent address → stack count
		maxHealthByAddr     = make(map[uint64]int64)
		determinedOnBoss    bool // Determined (895) applied to main boss (not any agent)
		determinedOnBoss762 bool // Determined (762) applied to main boss

		// For resultByTeamChange: boss's health must drop below 50% before a non-zero team change.
		bossHealthBelow50           = make(map[uint64]bool) // boss addr → true once health ≤ 50%
		bossTeamChangedAfterBelow50 = make(map[uint64]bool) // boss addr → true once team changed after below-50%

		// For resultByAttackTargetUntargetable (e.g. Deimos):
		// srcAgent of scAttackTarget = attack target; dstAgent = parent gadget.
		gadgetAttackTargets             = make(map[uint64][]uint64) // gadget addr → attack target addrs
		seenTargetable                  = make(map[uint64]bool)     // attack targets seen as targetable
		seenUntargetableAfterTargetable = make(map[uint64]bool)     // attack targets that completed true→false

		// For resultByExitCombatAfterSpawn: boss exits combat after spawning.
		// If enc.exitCombatMinDelay == 0, any exit combat counts (boss may be pre-placed).
		bossSpawnTime            = make(map[uint64]int64) // boss addr → spawn event time (ms)
		bossExitCombatAfterSpawn = make(map[uint64]bool)  // boss addrs that satisfied the condition

		// For resultByNPCSpawn: success when the designated NPC emits a spawn event.
		npcSpawnedForResult bool
	)

	hasQuickplay := skillIDs[skillQuickplayBoost] || skillIDs[skillQuickplayMoral]
	hasEmboldened := skillIDs[skillEmboldened]

	// Determined applied to a main boss address signals success for some encounters.
	// Checking dstAgent (the recipient) avoids false positives from Determined applied
	// to players or other agents during mechanics.
	checkDeterminedOnBoss := func(it combatItem) {
		if it.isActivation != 0 || !allBossAddrs[it.dstAgent] {
			return
		}
		if it.skillID == skillDetermined895 {
			determinedOnBoss = true
		}
		if it.skillID == skillDetermined762 {
			determinedOnBoss762 = true
		}
	}

	for r.remaining() >= 64 {
		var item combatItem
		if revision == 0 {
			item = r.readCombatItemRev0()
		} else {
			item = r.readCombatItemRev1()
		}

		switch item.stateChange {
		case scLogStart:
			logStartTime = item.time
			logStartUnix = item.value
		case scLogEnd:
			logEndTime = item.time
		case scReward:
			rewardIDs[item.dstAgent] = true
		case scChangeDead:
			bossDeaths[item.srcAgent] = true
		case scHealthUpdate:
			// dstAgent = health fraction * 10000; 5000 = 50%.
			if allBossAddrs[item.srcAgent] && item.dstAgent <= 5000 {
				bossHealthBelow50[item.srcAgent] = true
			}
		case scTeamChange:
			// Success = boss health dropped below 50% first, then non-zero team change.
			if allBossAddrs[item.srcAgent] && bossHealthBelow50[item.srcAgent] && item.dstAgent != 0 {
				bossTeamChangedAfterBelow50[item.srcAgent] = true
			}
		case scMaxHealth:
			if item.value > 0 {
				maxHealthByAddr[item.srcAgent] = int64(item.value)
			}
		case scSpawn:
			if allBossAddrs[item.srcAgent] {
				bossSpawnTime[item.srcAgent] = item.time
			}
			if resultNPCAddrSet[item.srcAgent] {
				npcSpawnedForResult = true
			}
		case scExitCombat:
			if allBossAddrs[item.srcAgent] {
				minDelay := enc.exitCombatMinDelay
				if minDelay == 0 {
					bossExitCombatAfterSpawn[item.srcAgent] = true
				} else if spawnAt, ok := bossSpawnTime[item.srcAgent]; ok && item.time-spawnAt >= minDelay {
					bossExitCombatAfterSpawn[item.srcAgent] = true
				}
			}
		case scAttackTarget:
			// srcAgent = attack target, dstAgent = parent gadget
			gadgetAttackTargets[item.dstAgent] = append(gadgetAttackTargets[item.dstAgent], item.srcAgent)
		case scTargetable:
			// dstAgent != 0 = became targetable; dstAgent == 0 = became untargetable
			if item.dstAgent != 0 {
				seenTargetable[item.srcAgent] = true
			} else if seenTargetable[item.srcAgent] {
				seenUntargetableAfterTargetable[item.srcAgent] = true
			}
		case scBuffInitial:
			if hasEmboldened && item.skillID == skillEmboldened {
				emboldenedCount[item.srcAgent]++
			}
			// Intentionally not checking Determined here: evtc ignores initial buffs
			// (AgentBuffGainedDeterminer uses ignoreInitial=true by default).
		case scNormal:
			// Killing blow (result=8): treat the target as dead, same as a ChangeDead event.
			// Some bosses (e.g. Bandit Trio) emit a killing blow without a subsequent ChangeDead.
			if item.result == resultKillingBlow && allBossAddrs[item.dstAgent] {
				bossDeaths[item.dstAgent] = true
			}
			// Legacy buff application (pre-2026-05-01 arcdps): statechange 0 with buff != 0.
			if item.buff != 0 && item.buffRemove == 0 {
				checkDeterminedOnBoss(item)
			}
		case scBuffApply:
			checkDeterminedOnBoss(item)
		}
	}

	maxEmbStacks := 0
	for _, n := range emboldenedCount {
		if n > maxEmbStacks {
			maxEmbStacks = n
		}
	}

	// ── Derived fields ────────────────────────────────────────────────────────
	mode := detectMode(enc, hasQuickplay, hasEmboldened, maxEmbStacks, maxHealthByAddr, skillIDs)
	result := detectResult(enc, bossDeaths, bossTeamChangedAfterBelow50, rewardIDs, determinedOnBoss, determinedOnBoss762, gadgetAttackTargets, seenUntargetableAfterTargetable, bossExitCombatAfterSpawn, npcSpawnedForResult, resultSkillPresent)

	var duration int64
	if logEndTime > logStartTime {
		duration = logEndTime - logStartTime
	}

	var loggedAt time.Time
	if logStartUnix > 0 {
		loggedAt = time.Unix(int64(logStartUnix), 0).UTC()
	}

	return &LogMetadata{
		EncounterID:   triggerID,
		EncounterName: enc.Name,
		Category:      enc.Category,
		Subcategory:   enc.Subcategory,
		Result:        result,
		Mode:          mode,
		DurationMs:    duration,
		LoggedAt:      loggedAt,
		Players:       extractPlayers(agents),
	}, nil
}

func detectMode(
	enc resolvedEncounter,
	hasQuickplay, hasEmboldened bool,
	maxEmbStacks int,
	maxHealthByAddr map[uint64]int64,
	skillIDs map[uint32]bool,
) string {
	// Universal signals take priority over per-encounter CM detection.
	if hasQuickplay {
		return "quickplay"
	}
	if hasEmboldened && maxEmbStacks > 0 {
		n := maxEmbStacks
		if n > 5 {
			n = 5
		}
		return fmt.Sprintf("emboldened%d", n)
	}

	switch enc.cmKind {
	case cmAlways:
		return "challenge"
	case cmByHealth:
		for _, addrs := range enc.mainBossAddrs {
			for _, addr := range addrs {
				if hp, ok := maxHealthByAddr[addr]; ok && hp > enc.cmValue {
					return "challenge"
				}
			}
		}
	case cmBySkill:
		if skillIDs[uint32(enc.cmValue)] {
			return "challenge"
		}
	case cmBySpecies:
		if enc.cmSpeciesPresent {
			return "challenge"
		}
	}

	return "normal"
}

func detectResult(
	enc resolvedEncounter,
	bossDeaths, bossTeamChangedAfterBelow50 map[uint64]bool,
	rewardIDs map[uint64]bool,
	determinedOnBoss bool,
	determinedOnBoss762 bool,
	gadgetAttackTargets map[uint64][]uint64,
	seenUntargetableAfterTargetable map[uint64]bool,
	bossExitCombatAfterSpawn map[uint64]bool,
	npcSpawnedForResult bool,
	resultSkillPresent bool,
) string {
	switch enc.resultKind {
	case resultByAnyDeath:
		// Success when any species' address receives a death or killing-blow event.
		for _, addrs := range enc.mainBossAddrs {
			for _, addr := range addrs {
				if bossDeaths[addr] {
					return "success"
				}
			}
		}
		return "failure"

	case resultByReward:
		if rewardIDs[enc.rewardID] {
			return "success"
		}
		return "failure"

	case resultByBuff895:
		if determinedOnBoss {
			return "success"
		}
		return "failure"

	case resultByBuff762:
		if determinedOnBoss762 {
			return "success"
		}
		return "failure"

	case resultByTeamChange:
		if len(enc.mainBossAddrs) == 0 {
			return "unknown"
		}
		// Success when the boss's health dropped below 50% and then the boss changed teams
		// to a non-zero affiliation. Filtering by health guards against bosses that change
		// teams multiple times during a fight (e.g. Ankka in XJJ).
		for _, addrs := range enc.mainBossAddrs {
			for _, addr := range addrs {
				if bossTeamChangedAfterBelow50[addr] {
					return "success"
				}
			}
		}
		return "failure"

	case resultByNPCSpawn:
		if npcSpawnedForResult {
			return "success"
		}
		return "failure"

	case resultBySkillPresent:
		if resultSkillPresent {
			return "success"
		}
		return "failure"

	case resultByExitCombatAfterSpawn:
		// Success when the boss exits combat ≥10 s after spawning (Xera phase 2: filters
		// out the brief combat dropout that occurs at the very start of the phase).
		for _, addrs := range enc.mainBossAddrs {
			for _, addr := range addrs {
				if bossExitCombatAfterSpawn[addr] {
					return "success"
				}
			}
		}
		return "failure"

	case resultByAttackTargetUntargetable:
		// Success when the attack target of the designated gadget goes targetable then
		// untargetable (Deimos: gadget phase starts → attack target targetable; kill → untargetable).
		for _, gadgetAddr := range enc.attackTargetGadgetAddrs {
			for _, atAddr := range gadgetAttackTargets[gadgetAddr] {
				if seenUntargetableAfterTargetable[atAddr] {
					return "success"
				}
			}
		}
		return "failure"

	case resultByDeath:
		if len(enc.mainBossAddrs) == 0 {
			return "unknown"
		}
		// Each species must have at least one address that received a ChangeDead event.
		// A species may appear at multiple addresses (NPC respawn artifacts); any one dying counts.
		for _, addrs := range enc.mainBossAddrs {
			died := false
			for _, addr := range addrs {
				if bossDeaths[addr] {
					died = true
					break
				}
			}
			if !died {
				return "failure"
			}
		}
		return "success"
	}

	return "unknown"
}

func extractPlayers(agents []rawAgent) []PlayerEntry {
	seen := make(map[uint64]bool)
	var players []PlayerEntry

	for _, a := range agents {
		// isElite != 0xFFFFFFFF identifies players.
		if a.isElite == 0xFFFFFFFF || seen[a.address] {
			continue
		}
		seen[a.address] = true

		// Player name field: "CharacterName\0AccountName\0Subgroup\0..."
		parts := strings.SplitN(strings.TrimRight(a.name, "\x00"), "\x00", 3)
		charName := parts[0]
		accountName := ""
		if len(parts) > 1 {
			accountName = parts[1]
		}

		prof := ""
		if a.prof > 0 && int(a.prof-1) < len(professionNames) {
			prof = professionNames[a.prof-1]
		}

		spec := ""
		switch a.isElite {
		case 0:
			// no elite spec — core profession
		case 1:
			spec = hotSpecByProfession[prof]
		default:
			spec = eliteSpecNames[a.isElite]
		}

		players = append(players, PlayerEntry{
			AccountName:   accountName,
			CharacterName: charName,
			Profession:    prof,
			EliteSpec:     spec,
		})
	}

	return players
}

// byteReader is a minimal little-endian binary reader over a byte slice.
type byteReader struct {
	b   []byte
	pos int
}

func (r *byteReader) remaining() int { return len(r.b) - r.pos }
func (r *byteReader) skip(n int)     { r.pos += n }

func (r *byteReader) readU8() uint8 {
	v := r.b[r.pos]
	r.pos++
	return v
}

func (r *byteReader) readU16() uint16 {
	v := binary.LittleEndian.Uint16(r.b[r.pos:])
	r.pos += 2
	return v
}

func (r *byteReader) readU32() uint32 {
	v := binary.LittleEndian.Uint32(r.b[r.pos:])
	r.pos += 4
	return v
}

func (r *byteReader) readI32() int32 { return int32(r.readU32()) }

func (r *byteReader) readU64() uint64 {
	v := binary.LittleEndian.Uint64(r.b[r.pos:])
	r.pos += 8
	return v
}

func (r *byteReader) readI64() int64 { return int64(r.readU64()) }

// readAgent consumes the 96-byte evtc_agent struct.
func (r *byteReader) readAgent() rawAgent {
	if r.pos+96 > len(r.b) {
		panic(fmt.Sprintf("truncated agent record at offset %d (file length %d)", r.pos, len(r.b)))
	}
	address := r.readU64()
	prof := r.readU32()
	isElite := r.readU32()
	r.skip(12) // toughness, concentration, healing, hitbox_width, condition, hitbox_height (6×int16)
	name := string(r.b[r.pos : r.pos+68])
	r.pos += 68
	return rawAgent{address, prof, isElite, name}
}

// readCombatItemRev0 consumes a 64-byte revision-0 cbtevent struct.
//
// Layout: time(8) srcAgent(8) dstAgent(8) value(4) buffDmg(4) overstackValue(2)
// skillId(2) srcInstId(2) dstInstId(2) srcMasterId(2) _pad(9) iff(1) buff(1)
// result(1) isActivation(1) isBuffRemove(1) isNinety(1) isFifty(1) isMoving(1)
// isStateChange(1) isFlanking(1) isShields(1) isOffCycle(1) _pad(1)
func (r *byteReader) readCombatItemRev0() combatItem {
	var item combatItem
	item.time = r.readI64()
	item.srcAgent = r.readU64()
	item.dstAgent = r.readU64()
	item.value = r.readI32()
	r.skip(4)                          // buff_dmg
	r.skip(2)                          // overstack_value (uint16)
	item.skillID = uint32(r.readU16()) // uint16 in rev0
	r.skip(2)                          // src_instid
	r.skip(2)                          // dst_instid
	r.skip(2)                          // src_master_instid
	r.skip(9)                          // padding
	r.skip(1)                          // iff
	item.buff = r.readU8()
	item.result = r.readU8()
	item.isActivation = r.readU8()
	item.buffRemove = r.readU8()
	r.skip(3) // is_ninety, is_fifty, is_moving
	item.stateChange = r.readU8()
	r.skip(4) // is_flanking, is_shields, is_offcycle, padding
	return item
}

// readCombatItemRev1 consumes a 64-byte revision-1 cbtevent struct.
//
// Layout: time(8) srcAgent(8) dstAgent(8) value(4) buffDmg(4) overstackValue(4)
// skillId(4) srcInstId(2) dstInstId(2) srcMasterId(2) dstMasterId(2) iff(1) buff(1)
// result(1) isActivation(1) isBuffRemove(1) isNinety(1) isFifty(1) isMoving(1)
// isStateChange(1) isFlanking(1) isShields(1) isOffCycle(1) padding(4)
func (r *byteReader) readCombatItemRev1() combatItem {
	var item combatItem
	item.time = r.readI64()
	item.srcAgent = r.readU64()
	item.dstAgent = r.readU64()
	item.value = r.readI32()
	r.skip(4) // buff_dmg
	r.skip(4) // overstack_value (uint32)
	item.skillID = r.readU32()
	r.skip(2) // src_instid
	r.skip(2) // dst_instid
	r.skip(2) // src_master_instid
	r.skip(2) // dst_master_instid
	r.skip(1) // iff
	item.buff = r.readU8()
	item.result = r.readU8()
	item.isActivation = r.readU8()
	item.buffRemove = r.readU8()
	r.skip(3) // is_ninety, is_fifty, is_moving
	item.stateChange = r.readU8()
	r.skip(7) // is_flanking, is_shields, is_offcycle, padding(4)
	return item
}
