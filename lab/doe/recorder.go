package doe

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

const (
	nClasses  = int(impl.ClassOther) + 1
	nOutcomes = int(impl.Uncertain) + 1
)

var outcomeNames = [nOutcomes]string{"success", "transient", "permanent", "throttled", "uncertain"}

// recorder collects per-call observations into shards (one per caller modulo
// the shard count) so the hot path is a few uncontended atomic adds.
type recorder struct {
	shards []shard
}

type shard struct {
	lat  Hist // Do latency of successful calls
	wait Hist // Do start → first handler entry
	exec Hist // first handler entry → Do return
	all  uint64
	ok   uint64
	errs [nClasses][nOutcomes]uint64
}

func newRecorder(callers int) *recorder {
	return &recorder{shards: make([]shard, min(callers, 2*runtime.GOMAXPROCS(0)))}
}

// call records one completed Do: arrival t0, Do start tDo, first handler
// entry (0 if none) and return t1, all on the run clock.
func (r *recorder) call(id int, im impl.Impl, t0, tDo, entered, t1 int64, err error) {
	sh := &r.shards[id%len(r.shards)]
	atomic.AddUint64(&sh.all, 1)
	if entered > 0 {
		sh.wait.Record(entered - tDo)
		sh.exec.Record(t1 - entered)
	}
	if err == nil {
		atomic.AddUint64(&sh.ok, 1)
		sh.lat.Record(t1 - t0)
		return
	}
	c, o := im.Classify(err), im.OutcomeOf(err)
	if o < 0 || int(o) >= nOutcomes {
		o = impl.Permanent // out-of-range tags read conservatively (A2)
	}
	atomic.AddUint64(&sh.errs[c][o], 1)
}

// totals merges the shards; call only after recording has stopped.
func (r *recorder) totals() (lat, wait, exec Hist, all, ok uint64, errs map[string]uint64) {
	errs = map[string]uint64{}
	for i := range r.shards {
		sh := &r.shards[i]
		lat.Merge(&sh.lat)
		wait.Merge(&sh.wait)
		exec.Merge(&sh.exec)
		all += sh.all
		ok += sh.ok
		for c, row := range sh.errs {
			for o, n := range row {
				if n > 0 {
					errs[fmt.Sprintf("%s/%s", impl.Class(c), outcomeNames[o])] += n
				}
			}
		}
	}
	return
}
