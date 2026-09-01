package parser

import (
	"path/filepath"
	"testing"
)

// fixture describes one test log and the values the parser is expected to produce.
// expectedResult may differ from actualResult for encounters where the parser's
// result detection is incomplete (e.g. Harvest Temple, which uses a gadget-based
// phase system the parser does not yet model).
type fixture struct {
	file           string
	encounter      string
	category       string
	subcategory    string
	actualResult   string // what actually happened in the log
	expectedResult string // what Parse() is expected to return
	mode           string
	minPlayers     int // minimum number of player entries expected
}

var fixtures = []fixture{
	{
		file:           "20250926-202028.zevtc",
		encounter:      "Ensolyss of the Infinite Torment",
		category:       "fractal",
		subcategory:    "Nightmare",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "challenge",
		minPlayers:     1,
	},
	{
		file:           "20250927-231732.zevtc",
		encounter:      "Whispering Shadow",
		category:       "fractal",
		subcategory:    "Kinfall",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "quickplay",
		minPlayers:     1,
	},
	{
		file:           "20260526-200101.zevtc",
		encounter:      "Harvest Temple",
		category:       "strike",
		subcategory:    "End of Dragons",
		actualResult:   "failure",
		expectedResult: "unknown", // gadget-based encounter; result detection not yet implemented
		mode:           "normal",
		minPlayers:     1,
	},
	{
		file:           "20260509-151251.zevtc",
		encounter:      "Spirit Race",
		category:       "raid",
		subcategory:    "Spirit Vale (Wing 1)",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "normal",
		minPlayers:     1,
	},
	{
		file:           "20250301-230226.zevtc",
		encounter:      "Eparch",
		category:       "fractal",
		subcategory:    "Lonely Tower",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "normal",
		minPlayers:     1,
	},
	{
		// Alternate trigger id 26257; an aborted pull that arcdps Log Manager omits.
		file:           "20240615-231736.zevtc",
		encounter:      "Eparch",
		category:       "fractal",
		subcategory:    "Lonely Tower",
		actualResult:   "failure",
		expectedResult: "failure",
		mode:           "normal",
		minPlayers:     1,
	},
	{
		file:           "20250301-224352.zevtc",
		encounter:      "Eparch",
		category:       "fractal",
		subcategory:    "Lonely Tower",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "challenge",
		minPlayers:     1,
	},
	{
		file:           "20260329-130149.zevtc",
		encounter:      "River of Souls",
		category:       "raid",
		subcategory:    "Hall of Chains (Wing 5)",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "normal",
		minPlayers:     1,
	},
	{
		file:           "20260329-125826.zevtc",
		encounter:      "River of Souls",
		category:       "raid",
		subcategory:    "Hall of Chains (Wing 5)",
		actualResult:   "failure",
		expectedResult: "failure",
		mode:           "normal",
		minPlayers:     1,
	},
	{
		file:           "20260602-205937.zevtc",
		encounter:      "Voice of the Fallen and Claw of the Fallen",
		category:       "strike",
		subcategory:    "Icebrood Saga",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "normal",
		minPlayers:     1,
	},
	{
		file:           "20260604-231716.zevtc",
		encounter:      "Aetherblade Hideout",
		category:       "strike",
		subcategory:    "End of Dragons",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "quickplay",
		minPlayers:     1,
	},
	{
		file:           "20260605-200837.zevtc",
		encounter:      "Qadim the Peerless",
		category:       "raid",
		subcategory:    "The Key of Ahdashim (Wing 7)",
		actualResult:   "failure",
		expectedResult: "failure",
		mode:           "normal",
		minPlayers:     1,
	},
	{
		file:           "20260605-211126.zevtc",
		encounter:      "Gorseval the Multifarious",
		category:       "raid",
		subcategory:    "Spirit Vale (Wing 1)",
		actualResult:   "success",
		expectedResult: "success",
		mode:           "normal",
		minPlayers:     1,
	},
}

func TestParse(t *testing.T) {
	for _, f := range fixtures {
		f := f
		t.Run(f.file, func(t *testing.T) {
			path := filepath.Join("testdata", f.file)
			meta, err := Parse(path)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if meta.EncounterName != f.encounter {
				t.Errorf("encounter name: got %q, want %q", meta.EncounterName, f.encounter)
			}
			if meta.Category != f.category {
				t.Errorf("category: got %q, want %q", meta.Category, f.category)
			}
			if meta.Subcategory != f.subcategory {
				t.Errorf("subcategory: got %q, want %q", meta.Subcategory, f.subcategory)
			}
			if meta.Result != f.expectedResult {
				t.Errorf("result: got %q, want %q (actual log result: %q)", meta.Result, f.expectedResult, f.actualResult)
			}
			if meta.Mode != f.mode {
				t.Errorf("mode: got %q, want %q", meta.Mode, f.mode)
			}
			if len(meta.Players) < f.minPlayers {
				t.Errorf("players: got %d, want at least %d", len(meta.Players), f.minPlayers)
			}
			if meta.DurationMs <= 0 {
				t.Errorf("duration: got %d, want > 0", meta.DurationMs)
			}
			if meta.LoggedAt.IsZero() {
				t.Error("logged_at: got zero time")
			}
		})
	}
}
