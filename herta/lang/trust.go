package lang

import (
	"fmt"
	"sort"
	"strings"
)

// TrustAssumption is an obligation, not a verified property or a diagnostic.
// Origin owns the obligation; Via is a canonical shortest call path from Op.
type TrustAssumption struct {
	Kind     string   `json:"kind"`
	Required bool     `json:"required"`
	Theorems []string `json:"theorems"`
	Origin   string   `json:"origin"`
	Via      []string `json:"via"`
	Detail   string   `json:"detail"`
}

type OpTrust struct {
	Op          string            `json:"op"`
	Assumptions []TrustAssumption `json:"assumptions"`
}

// Trust mirrors build's mode: classify clauses become Policy.Classify only
// with closed=true (gogen.go). It deliberately does not require Check to pass.
// Raw Go may hide calls, classifiers, key values and effects: absence from this
// ledger is not proof of safety. Callees covers declarations and visible DSL/
// typed-Op calls, not arbitrary helper/service methods.
func Trust(prog *Program, closed bool) []OpTrust {
	ops := map[string]*Op{}
	handlers := map[string]*Handler{}
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *Op:
			ops[opName(d)] = d
		case *Handler:
			handlers[d.Runtime+"::"+d.Name] = d
		}
	}
	names := make([]string, 0, len(ops))
	edges := map[string][]string{}
	for name, op := range ops {
		names = append(names, name)
		edges[name] = Callees(op, handlers[name])
		sort.Strings(edges[name])
	}
	sort.Strings(names)
	// BFS with sorted edges fixes provenance, terminates on cycles and avoids
	// exponential path enumeration. Each origin's assumptions propagate once.
	reachable := func(root string) [][]string {
		paths := [][]string{{root}}
		seen := map[string]bool{root: true}
		for i := 0; i < len(paths); i++ {
			path := paths[i]
			for _, child := range edges[path[len(path)-1]] {
				if !seen[child] {
					seen[child] = true
					paths = append(paths, append(append([]string(nil), path...), child))
				}
			}
		}
		return paths
	}
	local := map[string][]TrustAssumption{}
	add := func(name, kind string, required bool, theorems []string, detail string) {
		local[name] = append(local[name], TrustAssumption{
			Kind: kind, Required: required, Theorems: theorems, Origin: name, Detail: detail,
		})
	}
	for _, name := range names {
		op := ops[name]
		add(name, "opaque-handler", true, []string{"T1", "T4_composition"},
			"handler expressions and helpers are unchecked Go; effects and declared calls must be honest")
		legacy := !closed || op.Classify == nil
		if legacy {
			add(name, "legacy-tag-path", true, []string{"T2", "T5"},
				"handler tags win over own-deadline promotion; closed Policy.Classify is not installed")
		}
		retries := op.Retry != nil && op.Retry.Max > 1
		on := func(outcome string) bool {
			if retries {
				for _, o := range op.Retry.On {
					if o.Name == outcome {
						return true
					}
				}
			}
			return false
		}
		transient := legacy // opaque handler tags may supply Transient
		if !legacy {
			for _, arm := range op.Classify {
				transient = transient || arm.Outcome == "transient"
				if arm.Error == "_" {
					break // later classifier arms cannot be reached
				}
			}
		}
		if on("transient") && transient {
			add(name, "transient-honest", op.Effect == "nonidempotent", []string{"T1"},
				"every Transient-classified attempt must have applied no effect (potential raw-Go tags in legacy mode)")
		}
		if (op.Effect == "pure" || op.Effect == "idempotent") && (on("transient") || on("uncertain")) {
			for _, path := range reachable(name)[1:] {
				child := path[len(path)-1]
				if callee := ops[child]; callee != nil && callee.Effect == "nonidempotent" {
					add(name, "nested-retry-amplification", true, []string{"T4_composition"},
						"retry may replay nonidempotent callee along "+strings.Join(path, " -> "))
				}
			}
		}
		for _, child := range edges[name] {
			if ops[child] == nil {
				add(name, "unresolved-callee", true, []string{"T4_composition"}, "cannot inspect "+child)
			}
		}
	}
	// Reuse the checker's key-argument forwarding analysis instead of assuming
	// that equal/different parameter names prove equal/disjoint runtime strings.
	for _, d := range Check(prog) {
		if strings.HasPrefix(d.Msg, "HC001 path ") {
			for _, name := range names {
				if strings.HasPrefix(d.Msg, "HC001 path "+name+":") || strings.HasPrefix(d.Msg, "HC001 path "+name+" -> ") {
					add(name, "same-serialize-key", true, []string{"T4_key"}, d.String())
				}
			}
		}
	}
	report := make([]OpTrust, 0, len(names))
	for _, name := range names {
		entry := OpTrust{Op: name, Assumptions: []TrustAssumption{}}
		for _, path := range reachable(name) {
			for _, a := range local[path[len(path)-1]] {
				a.Via = path
				entry.Assumptions = append(entry.Assumptions, a)
			}
		}
		sort.Slice(entry.Assumptions, func(i, j int) bool {
			a, b := entry.Assumptions[i], entry.Assumptions[j]
			if a.Origin != b.Origin {
				return a.Origin < b.Origin
			}
			if a.Kind != b.Kind {
				return a.Kind < b.Kind
			}
			return a.Detail < b.Detail
		})
		report = append(report, entry)
	}
	return report
}

func TrustRequired(report []OpTrust) bool {
	for _, op := range report {
		for _, a := range op.Assumptions {
			if a.Required {
				return true
			}
		}
	}
	return false
}

func DumpTrust(report []OpTrust) string {
	var b strings.Builder
	for _, op := range report {
		fmt.Fprintf(&b, "%s\n", op.Op)
		opaqueOnly := len(op.Assumptions) > 0
		for _, a := range op.Assumptions {
			opaqueOnly = opaqueOnly && a.Kind == "opaque-handler"
		}
		if opaqueOnly {
			fmt.Fprintln(&b, "  trust: opaque-handler only [required; T1,T4_composition]")
			if len(op.Assumptions) == 1 {
				continue
			}
		}
		for _, a := range op.Assumptions {
			level := "informational"
			if a.Required {
				level = "required"
			}
			fmt.Fprintf(&b, "  trust: %s [%s; %s] via %s: %s\n",
				a.Kind, level, strings.Join(a.Theorems, ","), strings.Join(a.Via, " -> "), a.Detail)
		}
	}
	return b.String()
}
