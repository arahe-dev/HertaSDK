package doe

import "github.com/arahe-dev/hertasdk/lab/impl"

// nopImpl stands in for an implementation where only its name is needed.
type nopImpl struct{ impl.Impl }

func (nopImpl) Name() string { return "nop" }
