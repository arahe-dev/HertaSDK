package doe

import (
	"slices"
	"testing"
)

func TestStandardDesignsAreResolutionIV(t *testing.T) {
	for name, d := range map[string]Design{"Screen32": Screen32, "Screen64": Screen64} {
		if got := d.Resolution(); got != 4 {
			t.Errorf("%s: resolution %d, want 4", name, got)
		}
		sym := Symbols(Screening)
		for f := range d.K() {
			for _, a := range d.Aliases(1 << f)[1:] {
				if a.Order() < 3 {
					t.Errorf("%s: main effect %s aliased with %s", name, sym[f], a.Name(sym))
				}
			}
		}
		// Crossing with X keeps every X×factor interaction clear of 2fis.
		x := d.Cross()
		xs := append(slices.Clone(sym), ImplSymbol)
		for f := range d.K() {
			w := Word(1<<f | 1<<d.K())
			for _, a := range x.Aliases(w)[1:] {
				if a.Order() < 4 {
					t.Errorf("%s: %s aliased with %s", name, w.Name(xs), a.Name(xs))
				}
			}
		}
	}
}

func TestDefiningRelationOfKnownDesign(t *testing.T) {
	// 2^(4-1) with D=ABC: defining relation I=ABCD, so A=BCD, AB=CD.
	d := mustFractional(4, "ABC")
	sym := []string{"A", "B", "C", "D"}
	if def := d.Defining(); len(def) != 1 || def[0].Name(sym) != "ABCD" {
		t.Fatalf("defining relation %v", def)
	}
	if got := d.Aliases(0b0011); len(got) != 2 || got[1].Name(sym) != "CD" {
		t.Errorf("aliases of AB: %v", got)
	}
	if len(d.Contrasts()) != 7 {
		t.Errorf("contrasts: %d, want 7", len(d.Contrasts()))
	}
	if got := d.Generators(sym); !slices.Equal(got, []string{"D=ABC"}) {
		t.Errorf("generators %v", got)
	}
}

func TestDesignColumnsAreBalancedAndOrthogonal(t *testing.T) {
	for _, d := range []Design{Screen32, Screen64, Full(5), Screen32.Cross()} {
		for _, cl := range d.Contrasts() {
			var s int
			for r := range d.Runs() {
				s += int(d.Sign(r, cl[0]))
			}
			if s != 0 {
				t.Fatalf("contrast %v unbalanced: sum %d", cl[0], s)
			}
		}
		for f := range d.K() {
			for g := f + 1; g < d.K(); g++ {
				var dot int
				for r := range d.Runs() {
					dot += int(d.Level(r, f) * d.Level(r, g))
				}
				if dot != 0 {
					t.Fatalf("factors %d,%d not orthogonal", f, g)
				}
			}
		}
	}
}

func TestGeneratedColumnIsProductOfBase(t *testing.T) {
	d := mustFractional(5, "ABCD") // E = ABCD
	for r := range d.Runs() {
		p := d.Level(r, 0) * d.Level(r, 1) * d.Level(r, 2) * d.Level(r, 3)
		if d.Level(r, 4) != p {
			t.Fatalf("run %d: E=%d, ABCD=%d", r, d.Level(r, 4), p)
		}
	}
	if d.Level(0, 0) != -1 || d.Level(1, 0) != 1 {
		t.Error("run 0 must be all-low (Yates order)")
	}
}

func TestFractionalRejectsBadGenerators(t *testing.T) {
	for _, g := range []string{"AZ", "A", "AE"} {
		if _, err := Fractional(5, g); err == nil {
			t.Errorf("generator %q accepted", g)
		}
	}
}
