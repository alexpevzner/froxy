//
// Event bus
//

package main

import (
	"reflect"
	"sync"
	"sync/atomic"
)

//
// The event
//
type Event uint

const (
	EventStartup = Event(iota)
	EventConnStateChanged
	EventCountersChanged
	EventServerParamsChanged
	EventSitesChanged
	EventKeysChanged
	EventShutdownRequested
)

//
// Event->string
//
func (e Event) String() string {
	switch e {
	case EventStartup:
		return "EventStartup"
	case EventConnStateChanged:
		return "EventConnStateChanged"
	case EventCountersChanged:
		return "EventCountersChanged"
	case EventServerParamsChanged:
		return "EventServerParamsChanged"
	case EventSitesChanged:
		return "EventSitesChanged"
	case EventKeysChanged:
		return "EventKeysChanged"
	case EventShutdownRequested:
		return "EventShutdownRequested"
	}

	panic("internal error")
}

//
// The event bus
//
type Ebus struct {
	lock        sync.Mutex                   // Access lock
	subscribers map[<-chan Event]*subscriber // Table of subscribers
	pending     uint32                       // Pending events
	wake        chan struct{}                // Wake-up delivery goroutine
}

//
// The event subscriber
//
type subscriber struct {
	out     reflect.Value // reflect.ValueOf from subscriber's channel send end
	mask    uint32        // Mask of events the subscriber interested in
	pending uint32        // Not delivered yet events
	last    Event         // Last delivered event
}

//
// Create new event bus
//
func NewEbus() *Ebus {
	ebus := &Ebus{
		subscribers: make(map[<-chan Event]*subscriber),
		wake:        make(chan struct{}),
	}

	go ebus.goroutine()
	return ebus
}

//
// Subscribe to events
// If no events are specified, subscriber will receive all events
//
func (ebus *Ebus) Sub(events ...Event) <-chan Event {
	mask := ^uint32(1)
	if len(events) != 0 {
		mask = 0
		for _, e := range events {
			mask |= 1 << e
		}
	}

	c := make(chan Event)
	s := &subscriber{
		out:  reflect.ValueOf(c),
		mask: mask,
	}

	ebus.lock.Lock()
	ebus.subscribers[c] = s
	ebus.lock.Unlock()

	return c
}

//
// Cancel events subscription
//
func (ebus *Ebus) Unsub(c <-chan Event) {
	ebus.lock.Lock()
	delete(ebus.subscribers, c)
	ebus.lock.Unlock()
}

//
// Raise an event
//
func (ebus *Ebus) Raise(e Event) {
	bit := uint32(1 << e)
	old := uint32(0)

	// This is equivalent of atomic bitwise or:
	//     old, ebus.pending = ebus.pending, ebus.pending|bit
	for ok := false; !ok; {
		old = atomic.LoadUint32(&ebus.pending)
		ok = atomic.CompareAndSwapUint32(&ebus.pending, old, old|bit)
	}

	if old == 0 {
		ebus.wake <- struct{}{}
	}
}

//
// Ebus goroutine
//
func (ebus *Ebus) goroutine() {
	// Acquire the lock
	ebus.lock.Lock()
	defer ebus.lock.Unlock()

	// Initialize buffers
	cases := make([]reflect.SelectCase, 16)
	backmap := make([]*subscriber, 16) // select case->subscriber

	// The event loop
	for {
		// Refill array of reflect.Select() cases
		cases = cases[:0]
		backmap = backmap[:0]

		cases = append(cases,
			reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ebus.wake),
			})

		backmap = append(backmap, nil)

		for _, s := range ebus.subscribers {
			if s.pending != 0 {
				event := s.next()
				cases = append(cases,
					reflect.SelectCase{
						Dir:  reflect.SelectSend,
						Chan: s.out,
						Send: reflect.ValueOf(event),
					})
				backmap = append(backmap, s)
			}
		}

		// Wait for something to happen
		ebus.lock.Unlock()
		choosen, _, _ := reflect.Select(cases)
		ebus.lock.Lock()

		// Dispatch the event
		if choosen == 0 {
			pending := atomic.SwapUint32(&ebus.pending, 0)

			for _, s := range ebus.subscribers {
				s.pending |= pending & s.mask
			}
		} else {
			s := backmap[choosen]
			s.pending &^= 1 << s.last
		}
	}
}

//
// Get next pending event
//
func (s *subscriber) next() Event {
	e := Event(0)
	for (s.pending & (1 << e)) == 0 {
		e++
	}
	s.last = e
	return e
}
