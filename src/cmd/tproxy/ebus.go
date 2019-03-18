//
// Event bus
//

package main

import (
	"reflect"
	"sync"
)

//
// The event
//
type Event uint

const (
	EventConnStateChanged = Event(iota)
	EventCountersChanged
)

//
// The event bus
//
type Ebus struct {
	lock        sync.Mutex                   // Access lock
	subscribers map[<-chan Event]*subscriber // Table of subscribers
	evchan      chan Event                   // Event chain
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
		evchan:      make(chan Event),
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
	ebus.evchan <- e
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
				Chan: reflect.ValueOf(ebus.evchan),
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
		choosen, val, _ := reflect.Select(cases)
		ebus.lock.Lock()

		// Dispatch the event
		if choosen == 0 {
			event := val.Interface().(Event)
			bit := uint32(1) << event

			for _, s := range ebus.subscribers {
				s.pending |= bit & s.mask
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
