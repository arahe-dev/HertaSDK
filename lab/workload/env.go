// Package workload is the lab's shared harness: implementation selection,
// seeds and levels, the deterministic synctest rig, invariant monitors,
// leak and latency helpers. It holds no semantics of its own; those live in
// lab/model.
package workload

import (
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Full reports whether the industrial tier was requested (LAB_LEVEL=full).
func Full() bool { return os.Getenv("LAB_LEVEL") == "full" }

// Scale picks the quick or the full value of a knob.
func Scale[T any](quick, full T) T {
	if Full() {
		return full
	}
	return quick
}

// Seed returns LAB_SEED if set, else a fresh seed. It is logged whenever
// the test fails, so every random failure is one env var from a replay.
func Seed(t testing.TB) uint64 {
	seed := uint64(time.Now().UnixNano())
	if s := os.Getenv("LAB_SEED"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			t.Fatalf("LAB_SEED=%q: %v", s, err)
		}
		seed = v
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("reproduce with LAB_SEED=%d", seed)
		}
	})
	return seed
}

// Rand is the lab's deterministic generator for a seed.
func Rand(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, 0x6c6162)) }

// Impls returns the implementations named by LAB_IMPLS (default: all).
func Impls(t testing.TB) []impl.Impl {
	env := os.Getenv("LAB_IMPLS")
	if env == "" {
		return impl.All()
	}
	var out []impl.Impl
	for _, n := range strings.Split(env, ",") {
		im := impl.ByName(strings.TrimSpace(n))
		if im == nil {
			t.Fatalf("LAB_IMPLS: unknown implementation %q", n)
		}
		out = append(out, im)
	}
	return out
}

// Each runs fn as one subtest per selected implementation.
func Each(t *testing.T, fn func(t *testing.T, im impl.Impl)) {
	for _, im := range Impls(t) {
		t.Run(im.Name(), func(t *testing.T) { fn(t, im) })
	}
}
