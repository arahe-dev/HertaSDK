package workload

import (
	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
)

// Delta numbers a row of CONTRACT §8. Tests name the delta they depend on,
// so `grep Δ3` (or D3) finds every test that pins it.
type Delta int

const (
	D1  Delta = iota + 1 // Reject fails fast on a held key
	D2                   // Finished counted before the drain signal
	D3                   // one classifier for retry and report
	D4                   // caller-ended attempts are Permanent
	D5                   // Retry.On snapshotted
	D6                   // caller ctx.Err() beats ErrRuntimeStopping
	D7                   // attempt ctx cancelled on panic
	D8                   // no goroutine per timed-out Shutdown
	D9                   // Started counts handler entries only
	D10                  // no goroutines, few allocations per Do
	D11                  // a Reject overload is never reported as stopping
	// DNondet covers §8's last clause: behavior that is nondeterministic in
	// v0.2 (such as Shutdown racing "drained" against an expired ctx) must be
	// deterministic, per the contract, in every later implementation.
	DNondet
)

// Fixed reports whether im carries delta d. v0.2.0 carries none of them;
// every implementation after it (recon, v0.3 … v1.0) carries all of them.
func Fixed(im impl.Impl, d Delta) bool { return im.Name() != impl.Orig.Name() }

// Rules is the model mode im is judged by.
func Rules(im impl.Impl) model.Rules { return model.Rules{Fixed: Fixed(im, D1)} }
