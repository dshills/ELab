// Package uiaudit inventories native Go controls and command ingress sites.
// It is source evidence, not proof that a mounted gesture or every branch runs.
package uiaudit

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/dshills/elab/workbench"
)

type Site struct {
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Function   string   `json:"function"`
	Kind       string   `json:"kind"`
	Expression string   `json:"expression"`
	Commands   []string `json:"public_commands,omitempty"`
	Status     string   `json:"status"`
}

type Report struct {
	Format          string              `json:"format"`
	Version         int                 `json:"version"`
	Sources         map[string]string   `json:"source_sha256"`
	PublicCommands  []string            `json:"public_commands"`
	NativeAliases   map[string][]string `json:"native_alias_targets"`
	Sites           []Site              `json:"sites"`
	LiteralCoverage []string            `json:"commands_with_source_routes"`
	WithoutLiteral  []string            `json:"commands_without_source_routes"`
	Issues          []string            `json:"issues"`
	Limitations     []string            `json:"limitations"`
}

var callbackFields = map[string]bool{
	"Activate": true, "Change": true, "Dismiss": true, "FileDrop": true,
	"GridChange": true, "GridCommit": true, "TreeChange": true,
	"TreeDisclosure": true, "TabClose": true,
	"CloseRequest": true,
}

// These are reviewed semantic wrapper targets, not executable alias expansion.
// The file helpers can choose the asynchronous form of the same public action.
func nativeAliases() map[string][]string {
	return map[string][]string{
		"example.opamp-follower":    {"example.load"},
		"example.opamp-inverting":   {"example.load"},
		"example.rc-high-pass":      {"example.load"},
		"example.rl":                {"example.load"},
		"example.rlc":               {"example.load"},
		"example.flyback":           {"example.load"},
		"example.led-fade":          {"example.load"},
		"example.nmos":              {"example.load"},
		"example.pmos":              {"example.load"},
		"example.pnp":               {"example.load"},
		"example.emitter-follower":  {"example.load"},
		"example.bipolar-push-pull": {"example.load"},
		"example.ac-rc":             {"example.load"}, "example.ac-rlc": {"example.load"}, "example.ac-opamp": {"example.load"},
		"example.motor-dc": {"example.load"}, "example.motor-pwm": {"example.load"},
		"example.battery-led": {"example.load"}, "example.battery-regulator": {"example.load"},
		"example.night-light":          {"example.load"},
		"example.thermostat":           {"example.load"},
		"example.regulator-supply":     {"example.load"},
		"example.regulator-dropout":    {"example.load"},
		"example.relay-switch":         {"example.load"},
		"example.relay-flasher":        {"example.load"},
		"example.comparator-threshold": {"example.load"},
		"example.comparator-counter":   {"example.load"},
		"example.button-counter":       {"example.load"},
		"example.button-debounce":      {"example.load"},
		"example.display-counter":      {"example.load"},
		"example.d-flipflop":           {"example.load"},
		"example.jk-flipflop":          {"example.load"},
		"example.t-flipflop":           {"example.load"},
		"example.binary-counter":       {"example.load"},
		"example.decade-counter":       {"example.load"},
		"example.logic-inverter":       {"example.load"},
		"example.logic-switches":       {"example.load"},
		"example.logic-half-adder":     {"example.load"},
		"example.555-flasher":          {"example.load"},
		"example.555-oscillator":       {"example.load"},
		"example.555-one-shot":         {"example.load"},
		"example.zener":                {"example.load"},
		"example.schottky":             {"example.load"},
		"example.cmos":                 {"example.load"},
		"example.split-supply":         {"example.load"},
		"file.open":                    {"project.load", "project.load.start"},
		"file.save":                    {"project.save", "project.save.start"},
		"file.save-as":                 {"project.save", "project.save.start"},
		"appearance.light":             {"appearance.theme.set"},
		"appearance.dark":              {"appearance.theme.set"},
		"appearance.system":            {"appearance.theme.set"},
		"appearance.apollo":            {"appearance.theme.set"},
	}
}

func Audit(directory string) (Report, error) {
	out := Report{Format: "elab.native-command-inventory", Version: 1, Sources: map[string]string{}, NativeAliases: nativeAliases(), Issues: []string{}, Limitations: []string{
		"All top-level desktop production Go files are parsed, including both platform adapters; test/smoke files are excluded.",
		"Literal public routes are source mappings, not execution coverage. Native aliases are reviewed declarations, not inferred call-graph proofs.",
		"Dynamic routing, callback factories, gestures, dialog decisions and field assignments require branch/state and mounted-runtime qualification.",
		"No Go type checking or interprocedural binding is performed; relay callbacks and generated IDs are retained as expressions.",
		"Vellum internals and Objective-C event delivery are outside this inventory; their contracts and actual accessibility/gesture evidence remain separate.",
		"Commands without literal ingress may be agent queries, headless lifecycle or dynamically routed UI actions; absence is not automatically a missing user feature.",
	}}
	known := map[string]bool{}
	for _, c := range workbench.Catalog() {
		known[c.ID] = true
		out.PublicCommands = append(out.PublicCommands, c.ID)
	}
	sort.Strings(out.PublicCommands)
	for alias, targets := range out.NativeAliases {
		for _, target := range targets {
			if !known[target] {
				return out, fmt.Errorf("native alias %s names unknown public command %s", alias, target)
			}
		}
	}
	files, err := os.ReadDir(directory)
	if err != nil {
		return out, err
	}
	fset := token.NewFileSet()
	covered := map[string]bool{}
	for _, entry := range files {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.Contains(name, "smoke") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return out, err
		}
		file := "desktop/" + name
		out.Sources[file] = fmt.Sprintf("%x", sha256.Sum256(data))
		parsed, err := parser.ParseFile(fset, file, data, 0)
		if err != nil {
			return out, err
		}
		add := func(fn, kind string, node ast.Node, routes []string, status string) {
			commands := []string{}
			for _, route := range routes {
				if known[route] {
					commands = append(commands, route)
					covered[route] = true
				} else if targets, ok := out.NativeAliases[route]; ok {
					commands = append(commands, targets...)
					for _, target := range targets {
						covered[target] = true
					}
					status = "declared-native-wrapper"
				} else {
					out.Issues = append(out.Issues, fmt.Sprintf("%s:%d %s: unknown literal route %q", file, fset.Position(node.Pos()).Line, fn, route))
					status = "unknown-route"
				}
			}
			out.Sites = append(out.Sites, Site{file, fset.Position(node.Pos()).Line, fn, kind, expression(fset, node), commands, status})
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				// Native menu groups are package declarations, not view callbacks.
				ast.Inspect(decl, func(node ast.Node) bool {
					if kv, ok := node.(*ast.KeyValueExpr); ok {
						if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Commands" {
							if list, ok := kv.Value.(*ast.CompositeLit); ok {
								var routes []string
								for _, item := range list.Elts {
									routes = append(routes, literalRoutes(item)...)
								}
								add("(package)", "native-menu-group", kv, routes, "menu-route-set")
							}
						}
					}
					return true
				})
				continue
			}
			function := fn.Name.Name
			if fn.Recv != nil {
				function = expression(fset, fn.Recv.List[0].Type) + "." + function
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				switch n := node.(type) {
				case *ast.RangeStmt:
					if fn.Name.Name == "registerCommands" || fn.Name.Name == "registerFileCommands" {
						if list, ok := n.X.(*ast.CompositeLit); ok {
							for _, item := range list.Elts {
								if row, ok := item.(*ast.CompositeLit); ok && len(row.Elts) > 0 {
									add(function, "declared-registry-entry", row, literalRoutes(row.Elts[0]), "declared-registry-route")
								}
							}
						}
					}
				case *ast.CallExpr:
					method := callName(n)
					switch method {
					case "action":
						if len(n.Args) == 4 {
							add(function, "action-factory", n, literalRoutes(n.Args[2]), routeStatus(n.Args[2]))
						}
					case "cursorAction", "recordedCursorAction", "cursorActionFor":
						if len(n.Args) == 5 || method == "cursorActionFor" && len(n.Args) == 6 {
							add(function, "action-factory", n, literalRoutes(n.Args[3]), routeStatus(n.Args[3]))
						}
					case "valueRelatedAction":
						if len(n.Args) == 7 {
							add(function, "action-factory", n, literalRoutes(n.Args[5]), routeStatus(n.Args[5]))
						}
					case "button":
						if len(n.Args) == 2 {
							add(function, "registry-button", n, literalRoutes(n.Args[0]), routeStatus(n.Args[0]))
						}
					case "commandMenu":
						if len(n.Args) == 3 {
							add(function, "command-menu", n, menuRoutes(n.Args[2]), "menu-route-set")
						}
					case "invoke", "invokeAtCommandRevision", "beginFile", "requestReplace", "changePlacementField":
						if len(n.Args) > 0 {
							add(function, "domain-ingress", n, literalRoutes(n.Args[0]), routeStatus(n.Args[0]))
						}
					case "withFileDecision":
						if len(n.Args) == 3 {
							add(function, "callback-factory", n, nil, "requires-binding-review")
						}
					case "timeField", "valueApplyAction", "valuePresetAction", "verificationAction", "componentStartupDefault", "recordedPlaybackAction":
						add(function, "callback-factory", n, nil, "requires-binding-review")
					case "OnFirst", "On", "Invoke", "Dispatch", "Register":
						add(function, "event-or-dispatch-router", n, nil, "requires-binding-review")
					}
				case *ast.CompositeLit:
					for _, elt := range n.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if ok && key.Name == "Command" {
							add(function, "request-command-field", kv, literalRoutes(kv.Value), routeStatus(kv.Value))
						}
						if ok && (callbackFields[key.Name] || key.Name == "Custom") {
							add(function, "node-"+key.Name, kv, nil, "requires-binding-review")
						}
					}
				case *ast.AssignStmt:
					for _, lhs := range n.Lhs {
						if field, ok := lhs.(*ast.SelectorExpr); ok && callbackFields[field.Sel.Name] {
							add(function, "callback-assignment", n, nil, "requires-binding-review")
							break
						}
					}
				}
				return true
			})
		}
	}
	if len(out.Sources) == 0 {
		return out, fmt.Errorf("no production native Go sources")
	}
	for _, id := range out.PublicCommands {
		if covered[id] {
			out.LiteralCoverage = append(out.LiteralCoverage, id)
		} else {
			out.WithoutLiteral = append(out.WithoutLiteral, id)
		}
	}
	return out, nil
}

func expression(fset *token.FileSet, node ast.Node) string {
	var out bytes.Buffer
	_ = printer.Fprint(&out, fset, node)
	return out.String()
}
func callName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fn.Sel.Name
	case *ast.Ident:
		return fn.Name
	}
	return ""
}
func literalRoutes(expr ast.Expr) []string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		value, err := strconv.Unquote(lit.Value)
		if err == nil {
			return []string{value}
		}
	}
	return nil
}
func routeStatus(expr ast.Expr) string {
	if len(literalRoutes(expr)) > 0 {
		return "literal-public-route"
	}
	return "requires-binding-review"
}
func menuRoutes(expr ast.Expr) []string {
	var out []string
	if list, ok := expr.(*ast.CompositeLit); ok {
		for _, item := range list.Elts {
			if pair, ok := item.(*ast.CompositeLit); ok && len(pair.Elts) == 2 {
				out = append(out, literalRoutes(pair.Elts[0])...)
			}
		}
	}
	return out
}
