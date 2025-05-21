package runtime

import (
	"sync"

	"github.com/gar-id/queued/internal/events/runner"
	"github.com/samber/lo"
)

type Logger interface {
	Trace(format string, v ...interface{})
}

// eventListener holds a callback function which is invoked when
// the event listened for is emitted. It has a counter which indicates
// how the total number of events it is interested in. A value of zero
// means it does not expire (default).
type eventListener[T any] struct {
	callback func(...T) // Function to call with emitted event data
	counter  int        // The number of times this callback may be called. -1 = infinite
	delete   bool       // Flag to indicate that this listener should be deleted
}

// Events handles eventing
type Events[T any] struct {
	log Logger
	run []runner.Runner[T]

	// Go event listeners
	listeners  map[string][]*eventListener[T]
	notifyLock sync.RWMutex
}

func (e *Events[T]) Notify(sender runner.Runner[T], name string, data ...T) {
	e.notifyBackend(name, data...)
	for _, thisRun := range e.run {
		if thisRun == sender {
			continue
		}
		thisRun.Notify(name, data...)
	}
}

func (e *Events[T]) On(eventName string, callback func(...T)) func() {
	return e.registerListener(eventName, callback, -1)
}

func (e *Events[T]) OnMultiple(eventName string, callback func(...T), counter int) func() {
	return e.registerListener(eventName, callback, counter)
}

func (e *Events[T]) Once(eventName string, callback func(...T)) func() {
	return e.registerListener(eventName, callback, 1)
}

func (e *Events[T]) Emit(eventName string, data ...T) {
	e.notifyBackend(eventName, data...)
	for _, thisRun := range e.run {
		thisRun.Notify(eventName, data...)
	}
}

func (e *Events[T]) Off(eventName string) {
	e.unRegisterListener(eventName)
}

func (e *Events[T]) OffAll() {
	e.notifyLock.Lock()
	for eventName := range e.listeners {
		delete(e.listeners, eventName)
	}
	e.notifyLock.Unlock()
}

// NewEvents creates a new log subsystem
func NewEvents[T any](log Logger) *Events[T] {
	result := &Events[T]{
		log:       log,
		listeners: make(map[string][]*eventListener[T]),
	}
	return result
}

// registerListener provides a means of subscribing to events of type "eventName"
func (e *Events[T]) registerListener(eventName string, callback func(...T), counter int) func() {
	// Create new eventListener
	thisListener := &eventListener[T]{
		callback: callback,
		counter:  counter,
		delete:   false,
	}
	e.notifyLock.Lock()
	// Append the new listener to the listeners slice
	e.listeners[eventName] = append(e.listeners[eventName], thisListener)
	e.notifyLock.Unlock()
	return func() {
		e.notifyLock.Lock()
		defer e.notifyLock.Unlock()

		if _, ok := e.listeners[eventName]; !ok {
			return
		}
		e.listeners[eventName] = lo.Filter(e.listeners[eventName], func(l *eventListener[T], i int) bool {
			return l != thisListener
		})
	}
}

// unRegisterListener provides a means of unsubscribing to events of type "eventName"
func (e *Events[T]) unRegisterListener(eventName string) {
	e.notifyLock.Lock()
	// Clear the listeners
	delete(e.listeners, eventName)
	e.notifyLock.Unlock()
}

// Notify backend for the given event name
func (e *Events[T]) notifyBackend(eventName string, data ...T) {
	e.notifyLock.Lock()
	defer e.notifyLock.Unlock()

	// Get list of event listeners
	listeners := e.listeners[eventName]
	if listeners == nil {
		e.log.Trace("No listeners for event '%s'", eventName)
		return
	}

	// We have a dirty flag to indicate that there are items to delete
	itemsToDelete := false

	// Callback in goroutine
	for _, listener := range listeners {
		if listener.counter > 0 {
			listener.counter--
		}
		go listener.callback(data...)

		if listener.counter == 0 {
			listener.delete = true
			itemsToDelete = true
		}
	}

	// Do we have items to delete?
	if itemsToDelete {

		// Create a new Listeners slice
		var newListeners []*eventListener[T]

		// Iterate over current listeners
		for _, listener := range listeners {
			// If we aren't deleting the listener, add it to the new list
			if !listener.delete {
				newListeners = append(newListeners, listener)
			}
		}

		// Save new listeners or remove entry
		if len(newListeners) > 0 {
			e.listeners[eventName] = newListeners
		} else {
			delete(e.listeners, eventName)
		}
	}
}

func (e *Events[T]) Addrun(apprun runner.Runner[T]) {
	e.run = append(e.run, apprun)
}
