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
	scNormal      uint8 = 0
	scChangeDead  uint8 = 4
	scLogStart    uint8 = 9
	scLogEnd      uint8 = 10
	scMaxHealth   uint8 = 12
	scReward      uint8 = 17
	scBuffInitial uint8 = 18
)

// Skill IDs used for mode detection.
const (
	skillEmboldened     uint32 = 68087
	skillQuickplayBoost uint32 = 77676 // fractals
	skillQuickplayMoral uint32 = 79492 // raids
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

	// ── Combat event scan ─────────────────────────────────────────────────────
	var (
		logStartTime    int64
		logEndTime      int64
		logStartUnix    int32
		bossDeaths      = make(map[uint64]bool)
		rewardID        uint64
		emboldenedCount = make(map[uint64]int) // agent address → stack count
		maxHealthByAddr = make(map[uint64]int64)
		determinedSeen  bool
	)

	hasQuickplay := skillIDs[skillQuickplayBoost] || skillIDs[skillQuickplayMoral]
	hasEmboldened := skillIDs[skillEmboldened]

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
			rewardID = item.dstAgent
		case scChangeDead:
			bossDeaths[item.srcAgent] = true
		case scMaxHealth:
			if item.value > 0 {
				maxHealthByAddr[item.srcAgent] = int64(item.value)
			}
		case scBuffInitial:
			if hasEmboldened && item.skillID == skillEmboldened {
				emboldenedCount[item.srcAgent]++
			}
			// Some encounters signal success by applying Determined (895) to the boss
			// as a BuffInitial state change rather than a normal buff apply event.
			if item.skillID == skillDetermined895 {
				determinedSeen = true
			}
		case scNormal:
			// Detect Determined (895) buff apply — signals success for some encounters.
			if item.buff != 0 && item.buffRemove == 0 && item.isActivation == 0 &&
				item.skillID == skillDetermined895 {
				determinedSeen = true
			}
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
	result := detectResult(enc, bossDeaths, rewardID, determinedSeen)

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

func detectResult(enc resolvedEncounter, bossDeaths map[uint64]bool, rewardID uint64, determinedSeen bool) string {
	switch enc.resultKind {
	case resultByReward:
		if rewardID == enc.rewardID {
			return "success"
		}
		return "failure"

	case resultByBuff895:
		if determinedSeen {
			return "success"
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
	r.skip(1) // result
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
	r.skip(4)              // buff_dmg
	r.skip(4)              // overstack_value (uint32)
	item.skillID = r.readU32()
	r.skip(2)              // src_instid
	r.skip(2)              // dst_instid
	r.skip(2)              // src_master_instid
	r.skip(2)              // dst_master_instid
	r.skip(1)              // iff
	item.buff = r.readU8()
	r.skip(1) // result
	item.isActivation = r.readU8()
	item.buffRemove = r.readU8()
	r.skip(3) // is_ninety, is_fifty, is_moving
	item.stateChange = r.readU8()
	r.skip(7) // is_flanking, is_shields, is_offcycle, padding(4)
	return item
}
