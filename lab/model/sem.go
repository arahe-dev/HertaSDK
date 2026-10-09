package model

// Sem is the reference budget semaphore (CONTRACT §4.1): a weighted
// semaphore with a strict FIFO waiter queue and head-of-line blocking.
type Sem struct {
	Cap, Used int64
	queue     []semWaiter
}

type semWaiter struct {
	id int
	n  int64
}

// TryAcquire grants n units iff they fit and nobody is queued.
func (s *Sem) TryAcquire(n int64) bool {
	if len(s.queue) > 0 || s.Used+n > s.Cap {
		return false
	}
	s.Used += n
	return true
}

// Acquire grants n units to id now, or queues id and reports false.
func (s *Sem) Acquire(id int, n int64) bool {
	if s.TryAcquire(n) {
		return true
	}
	s.queue = append(s.queue, semWaiter{id, n})
	return false
}

// Release returns n units and reports the waiters granted, in grant order.
func (s *Sem) Release(n int64) []int {
	s.Used -= n
	if s.Used < 0 {
		panic("model: semaphore released more than held")
	}
	return s.grant()
}

// Cancel removes a queued waiter. Removing the head may unblock the waiters
// behind it; those grants are returned.
func (s *Sem) Cancel(id int) []int {
	for i, w := range s.queue {
		if w.id == id {
			s.queue = append(s.queue[:i:i], s.queue[i+1:]...)
			if i == 0 {
				return s.grant()
			}
			return nil
		}
	}
	return nil
}

// Queued lists waiting ids from head to tail.
func (s *Sem) Queued() []int {
	ids := make([]int, len(s.queue))
	for i, w := range s.queue {
		ids[i] = w.id
	}
	return ids
}

// grant serves the queue from the head for as long as each waiter fits.
func (s *Sem) grant() []int {
	var granted []int
	for len(s.queue) > 0 && s.Used+s.queue[0].n <= s.Cap {
		s.Used += s.queue[0].n
		granted = append(granted, s.queue[0].id)
		s.queue = s.queue[1:]
	}
	return granted
}
