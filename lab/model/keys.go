package model

// Keys is the reference key locker (CONTRACT §4.2): per-key mutual
// exclusion with FIFO hand-off. An entry exists only while the key has a
// holder, so idle keys cost nothing.
type Keys struct {
	entries map[string]*keyEntry
}

type keyEntry struct {
	holder int
	queue  []int
}

// Held reports whether key has a holder (a queued key always has one).
func (k *Keys) Held(key string) bool {
	_, ok := k.entries[key]
	return ok
}

// Acquire makes id the holder of a free key, or queues it and reports false.
func (k *Keys) Acquire(id int, key string) bool {
	if e, ok := k.entries[key]; ok {
		e.queue = append(e.queue, id)
		return false
	}
	if k.entries == nil {
		k.entries = map[string]*keyEntry{}
	}
	k.entries[key] = &keyEntry{holder: id}
	return true
}

// Release ends the current hold. It hands the key to the head waiter and
// returns its id, or deletes the entry and returns -1.
func (k *Keys) Release(key string) int {
	e := k.entries[key]
	if len(e.queue) == 0 {
		delete(k.entries, key)
		return -1
	}
	e.holder, e.queue = e.queue[0], e.queue[1:]
	return e.holder
}

// Cancel removes a queued waiter. A waiter never blocks anyone behind it,
// so cancelling one grants nothing.
func (k *Keys) Cancel(id int, key string) {
	e := k.entries[key]
	for i, w := range e.queue {
		if w == id {
			e.queue = append(e.queue[:i:i], e.queue[i+1:]...)
			return
		}
	}
}
