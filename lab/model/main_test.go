package model_test

import (
	"testing"

	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestSynctestProbe runs only as the child of workload.RequireBubbles.
func TestSynctestProbe(t *testing.T) { workload.Probe(t) }
