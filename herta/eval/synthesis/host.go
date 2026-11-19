// Package synthesis measures the Go/Bend/Lean synthesis: the same publish
// and checkout contracts as plain Go SDK, typed Go SDK with dispatch
// evidence, and generated Herta plans (main_gen.go, bare_gen.go), against
// one fake provider whose committed writes are the ground truth.
//
// Regenerate with herta build -closed -pkg synthesis {synthesis,bare}.he.
package synthesis

import (
	"context"
	"errors"
)

var (
	ErrAckLost = errors.New("ack lost after apply")
	ErrNotSent = errors.New("not sent")
)

// Fault is what the provider does on the first attempt only.
type Fault int

const (
	PreSend     Fault = iota // refused before sending
	AckHonest                // applied, ack lost, reported as ErrAckLost
	AckMislabel              // applied, ack lost, reported as ErrNotSent
	EarlySend                // applied before the dispatch point, reported as ErrNotSent
)

// provider is the backend plus its adapter; commits is ground truth.
type provider struct {
	fault          Fault
	prepares, sent int
	commits        int
}

type request struct{ id string }

var prov *provider

func prepareHost(_ context.Context, id string) (request, error) {
	prov.prepares++
	first := prov.prepares == 1
	switch {
	case first && prov.fault == PreSend:
		return request{}, ErrNotSent
	case first && prov.fault == EarlySend:
		prov.commits++
		return request{}, ErrNotSent
	}
	return request{id: id}, nil
}

func sendHost(_ context.Context, _ request) (uint64, error) {
	prov.sent++
	prov.commits++
	if prov.sent == 1 {
		switch prov.fault {
		case AckHonest:
			return 0, ErrAckLost
		case AckMislabel:
			return 0, ErrNotSent
		}
	}
	return uint64(prov.commits), nil
}
