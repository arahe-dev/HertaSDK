package doe

import (
	"fmt"
	"math/bits"
	"slices"
	"strings"
)

// Word is a product of factor columns, as a bitmask over factor indices
// (bit i = factor i). The empty word is the grand mean, "I" in DoE texts.
type Word uint32

func (w Word) Order() int { return bits.OnesCount32(uint32(w)) }

// Name spells w with the given factor symbols, e.g. "ABD"; "1" if empty.
func (w Word) Name(symbols []string) string {
	if w == 0 {
		return "1"
	}
	var b strings.Builder
	for i, s := range symbols {
		if w&(1<<i) != 0 {
			b.WriteString(s)
		}
	}
	return b.String()
}

// Design is a two-level (±1) factorial design with 2^Base runs in standard
// (Yates) order. Run r sets base column b to +1 when bit b of r is set; each
// factor's column is the product of the base columns in its mask, so a
// generated factor F = ABCD holds exactly (sign +) the product of A..D.
type Design struct {
	Base int
	Cols []uint32 // Cols[f]: base-column mask of factor f
}

// Full returns the 2^k full factorial.
func Full(k int) Design {
	d := Design{Base: k}
	for f := range k {
		d.Cols = append(d.Cols, 1<<f)
	}
	return d
}

// Fractional returns the 2^(k-p) design with base factors A, B, C, ... and
// one generated factor per generator, written over the base symbols
// ("BCDE" means the next factor = B·C·D·E).
func Fractional(k int, gens ...string) (Design, error) {
	d := Full(k - len(gens))
	for _, g := range gens {
		var c uint32
		for _, r := range g {
			i := int(r - 'A')
			if i < 0 || i >= d.Base {
				return Design{}, fmt.Errorf("doe: generator %q uses non-base factor %q", g, r)
			}
			c |= 1 << i
		}
		if bits.OnesCount32(c) < 2 {
			return Design{}, fmt.Errorf("doe: generator %q must be an interaction", g)
		}
		d.Cols = append(d.Cols, c)
	}
	return d, nil
}

// Cross appends one new base factor, fully crossed with every run; used to
// make the implementation a factor of any design.
func (d Design) Cross() Design {
	return Design{Base: d.Base + 1, Cols: append(slices.Clip(d.Cols), 1<<d.Base)}
}

func (d Design) K() int    { return len(d.Cols) }
func (d Design) Runs() int { return 1 << d.Base }

// col is the base-column mask of word w: its column's sign pattern.
func (d Design) col(w Word) uint32 {
	var c uint32
	for f, fc := range d.Cols {
		if w&(1<<f) != 0 {
			c ^= fc
		}
	}
	return c
}

// Sign is the ±1 value of word w's column in run r.
func (d Design) Sign(r int, w Word) int8 {
	c := d.col(w)
	if (bits.OnesCount32(c)-bits.OnesCount32(uint32(r)&c))%2 == 0 {
		return 1
	}
	return -1
}

// Level is the ±1 level of factor f in run r.
func (d Design) Level(r, f int) int8 { return d.Sign(r, 1<<f) }

// Generators spells each generated factor as "F=ABCD" with the symbols.
func (d Design) Generators(symbols []string) []string {
	var out []string
	for f, c := range d.Cols {
		if bits.OnesCount32(c) > 1 {
			out = append(out, symbols[f]+"="+d.baseWord(c).Name(symbols))
		}
	}
	return out
}

// baseWord maps a base-column mask back to the word over base factors.
func (d Design) baseWord(c uint32) Word {
	var w Word
	for f, fc := range d.Cols {
		if bits.OnesCount32(fc) == 1 && c&fc != 0 {
			w |= 1 << f
		}
	}
	return w
}

// classes buckets every non-empty word by column: each bucket is one alias
// class (one estimable contrast), its words shortest first.
func (d Design) classes() map[uint32][]Word {
	m := map[uint32][]Word{}
	for w := Word(1); w < 1<<d.K(); w++ {
		c := d.col(w)
		m[c] = append(m[c], w)
	}
	for _, ws := range m {
		slices.SortFunc(ws, byOrder)
	}
	return m
}

// Defining is the defining relation: every word whose column is all +1.
func (d Design) Defining() []Word { return d.classes()[0] }

// Resolution is the order of the shortest defining word; 0 means a full
// factorial (nothing aliased).
func (d Design) Resolution() int {
	if def := d.Defining(); len(def) > 0 {
		return def[0].Order()
	}
	return 0
}

// Aliases returns w's alias class (w included), shortest first.
func (d Design) Aliases(w Word) []Word { return d.classes()[d.col(w)] }

// Contrasts returns the 2^Base - 1 estimable alias classes, each shortest
// word first, ordered by their shortest word.
func (d Design) Contrasts() [][]Word {
	cl := d.classes()
	out := make([][]Word, 0, len(cl))
	for c, ws := range cl {
		if c != 0 {
			out = append(out, ws)
		}
	}
	slices.SortFunc(out, func(a, b []Word) int { return byOrder(a[0], b[0]) })
	return out
}

func byOrder(a, b Word) int {
	if c := a.Order() - b.Order(); c != 0 {
		return c
	}
	return int(a) - int(b)
}
