package mutate

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

var header = []string{"id", "file", "line", "func", "operator", "nth", "status",
	"stage", "how", "killers", "flaky", "seconds", "from", "to", "reason"}

// WriteCSV writes one row per result.
func WriteCSV(w io.Writer, rs []Result) error {
	cw := csv.NewWriter(w)
	cw.Write(header)
	for _, r := range rs {
		cw.Write([]string{
			strconv.Itoa(r.ID), r.File, strconv.Itoa(r.Line), r.Func, string(r.Op),
			strconv.Itoa(r.Nth), string(r.Status), r.Stage, r.How,
			strings.Join(r.Killers, " "), strings.Join(r.Flaky, "; "), strconv.FormatFloat(r.Elapsed.Seconds(), 'f', 2, 64),
			r.From, r.To, r.Reason,
		})
	}
	cw.Flush()
	return cw.Error()
}
