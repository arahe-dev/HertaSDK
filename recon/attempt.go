package execution

import "context"

// run is the attempt loop (CONTRACT §5). The caller's context is checked
// before every attempt; once it has ended, Do returns ctx.Err() untagged.
// There is no backoff in v0.2.
func (op *Operation[I, O]) run(ctx context.Context, in I) (O, error) {
	var zero O
	for n := 1; ; n++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		if n == 1 {
			op.rt.stats.started.Add(1) // Δ9: only when the handler runs
		} else {
			op.rt.stats.retried.Add(1)
		}
		out, expired, err := op.attempt(ctx, in)
		if err == nil {
			return out, nil // a late result is returned, not discarded
		}
		o, promoted := classify(err, expired)
		if n >= op.attempts || !op.retry.has(o) {
			return zero, failed(err, promoted)
		}
	}
}

// attempt runs the handler once. With a Timeout the handler gets a fresh
// deadline derived from the caller's context, released on every path,
// panics included (Δ7). expired reports that this attempt's own deadline
// fired while the caller's context was still live (Δ4).
func (op *Operation[I, O]) attempt(ctx context.Context, in I) (out O, expired bool, err error) {
	if op.timeout == 0 {
		out, err = op.handler(ctx, in)
		return out, false, err
	}
	actx, cancel := context.WithTimeout(ctx, op.timeout)
	defer cancel()
	out, err = op.handler(actx, in)
	expired = err != nil && actx.Err() == context.DeadlineExceeded && ctx.Err() == nil
	return out, expired, err
}
