package workbench

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ExampleCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Browser state is session presentation, never schematic content or physics.
type ExampleBrowser struct {
	Visible     bool              `json:"visible"`
	Category    string            `json:"category"`
	Query       string            `json:"query"`
	SelectedID  string            `json:"selected_id"`
	MatchingIDs []string          `json:"matching_ids"`
	Categories  []ExampleCategory `json:"categories"`
}

// ExamplePreview is the exact immutable template drawing used for thumbnails.
// SVG is generated from the shipped graph/settings, without opening a document,
// solving, reading a file or using a network/image service.
type ExamplePreview struct {
	Info         ExampleInfo `json:"info"`
	Category     string      `json:"category"`
	SourceSHA256 string      `json:"source_sha256"`
	SVG          string      `json:"svg"`
}

func PreviewExample(id string) (ExamplePreview, error) {
	d, ok := findExample(id)
	if !ok {
		return ExamplePreview{}, &actionError{"not-found", "Choose an example ID from examples.list"}
	}
	hash := contentHash(d.graph, d.settings)
	display := CircuitDisplay{Preferences: DefaultDisplayPreferences()}
	data := schematicSVGDrawing(d.graph, 0, 0, display, configuredSourceAnnotations(d.graph, d.settings), d.settings, false)
	return ExamplePreview{Info: d.info, Category: exampleCategory(id), SourceSHA256: hex.EncodeToString(hash[:]), SVG: string(data)}, nil
}

func exampleCategory(id string) string {
	switch id {
	case "self-heating", "rc-block", "capacitor-loss", "winding-loss", "led", "split-supply", "rc", "rc-high-pass", "led-fade", "rl", "rlc", "flyback":
		return "basics"
	case "npn", "pnp", "emitter-follower", "bipolar-push-pull", "switch-npn", "blink", "flasher", "nmos", "pmos", "cmos":
		return "transistors"
	case "555-flasher", "555-oscillator", "555-one-shot":
		return "timers"
	case "shift-register", "display-counter", "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter", "logic-inverter", "logic-switches", "logic-half-adder", "button-counter", "button-debounce", "comparator-counter":
		return "logic"
	case "photodiode-load", "photodiode-tia", "night-light", "thermostat", "comparator-threshold":
		return "sensors"
	case "discrete-hbridge", "scr-latch", "triac-phase", "buck", "rectifier", "zener", "schottky", "coupled-windings", "transformer-supply", "opto-switch", "battery-led", "battery-regulator", "regulator-supply", "regulator-dropout", "relay-switch", "relay-flasher":
		return "power"
	case "hbridge-motor", "motor-dc", "motor-pwm":
		return "motors"
	case "555-tone", "light-alarm":
		return "sound"
	case "noise-rc", "noise-amplifier", "class-ab-temperature", "feedback-loop", "step-response", "jfet-preamp", "jfet-current", "pjfet-current", "current-led", "load-step", "constant-power-load", "class-ab-speaker", "speaker-amplifier", "rc-tolerance", "transistor-profile", "amplifier-distortion", "discrete-class-ab", "opamp-slew", "opamp-bandwidth", "opamp-follower", "opamp-inverting", "ac-rc", "ac-rlc", "ac-opamp":
		return "analysis"
	}
	return ""
}

func startExample(id string) bool {
	return slices.Contains([]string{"led", "rc", "npn", "555-flasher", "display-counter", "night-light"}, id)
}

func ExampleCategories() []ExampleCategory {
	groups := []ExampleCategory{{ID: "start", Name: "Start here"}, {ID: "all", Name: "All examples"}, {ID: "basics", Name: "Circuit basics"}, {ID: "transistors", Name: "Transistors"}, {ID: "timers", Name: "555 timers"}, {ID: "logic", Name: "Logic & counters"}, {ID: "sensors", Name: "Sensors & thresholds"}, {ID: "power", Name: "Power & relays"}, {ID: "motors", Name: "Motors"}, {ID: "sound", Name: "Sound"}, {ID: "analysis", Name: "Amplifiers & filters"}}
	for _, info := range ExampleCatalog() {
		for i := range groups {
			if groups[i].ID == "all" || groups[i].ID == exampleCategory(info.ID) || groups[i].ID == "start" && startExample(info.ID) {
				groups[i].Count++
			}
		}
	}
	return groups
}

func exampleMatches(category, query string) []string {
	words := strings.Fields(strings.ToLower(query))
	ids := []string{}
	for _, info := range ExampleCatalog() {
		if len(words) == 0 && category != "all" && category != exampleCategory(info.ID) && !(category == "start" && startExample(info.ID)) {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{info.ID, info.Title, info.Question, info.Expected, info.Try, exampleCategory(info.ID)}, " "))
		tokens := strings.FieldsFunc(haystack, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
		match := true
		for _, word := range words {
			if utf8.RuneCountInString(word) <= 2 {
				// RC/AC/DC are topics, not arbitrary fragments of "circuit".
				match = match && slices.Contains(tokens, word)
			} else {
				match = match && strings.Contains(strings.ReplaceAll(haystack, "-", ""), strings.ReplaceAll(word, "-", ""))
			}
		}
		if match {
			ids = append(ids, info.ID)
		}
	}
	return ids
}

func (s *Service) exampleBrowserLocked() ExampleBrowser {
	return cloneExampleBrowser(s.exampleBrowser)
}

func cloneExampleBrowser(b ExampleBrowser) ExampleBrowser {
	b.MatchingIDs = append([]string{}, b.MatchingIDs...)
	b.Categories = append([]ExampleCategory{}, b.Categories...)
	return b
}

func (s *Service) refreshExampleBrowserLocked() {
	b := s.exampleBrowser
	if b.Category == "" {
		b.Category = "start"
	}
	b.MatchingIDs = exampleMatches(b.Category, b.Query)
	b.Categories = ExampleCategories()
	if !slices.Contains(b.MatchingIDs, b.SelectedID) {
		b.SelectedID = ""
		if len(b.MatchingIDs) > 0 {
			b.SelectedID = b.MatchingIDs[0]
			if b.Category == "start" && strings.TrimSpace(b.Query) == "" {
				b.SelectedID = "led"
			}
		}
	}
	s.exampleBrowser = b
}

func (s *Service) applyExampleBrowserLocked(command string, raw json.RawMessage) error {
	switch command {
	case "examples.visibility.set":
		var p ScopeVisibilityParams
		if err := params(raw, &p, "visible"); err != nil {
			return err
		}
		s.exampleBrowser.Visible = p.Visible
	case "examples.category.set":
		var p PartCategoryParams
		if err := params(raw, &p, "category"); err != nil {
			return err
		}
		if !slices.ContainsFunc(ExampleCategories(), func(c ExampleCategory) bool { return c.ID == p.Category }) {
			return fmt.Errorf("choose a category from snapshot.example_browser.categories")
		}
		s.exampleBrowser.Category, s.exampleBrowser.Query = p.Category, ""
	case "examples.search.set":
		var p SearchParams
		if err := params(raw, &p, "query"); err != nil {
			return err
		}
		if len(p.Query) > 128 || !utf8.ValidString(p.Query) || strings.ContainsRune(p.Query, 0) {
			return fmt.Errorf("search needs valid text of at most 128 bytes without NUL")
		}
		s.exampleBrowser.Query = p.Query
	case "examples.select":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if !slices.Contains(s.exampleBrowser.MatchingIDs, p.ID) {
			return fmt.Errorf("choose an ID from snapshot.example_browser.matching_ids")
		}
		s.exampleBrowser.SelectedID = p.ID
	}
	// Recompute only when browser input changes, not on every live snapshot.
	s.refreshExampleBrowserLocked()
	return nil
}
