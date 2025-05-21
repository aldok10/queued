package events

import (
	"context"
	"log"
	goruntime "runtime"

	"github.com/gar-id/queued/internal/events/runner"
)

const contextError = `An invalid context was passed. This method requires the specific context given in the lifecycle hooks.`

type Events[T any] interface {
	On(eventName string, callback func(...T)) func()
	OnMultiple(eventName string, callback func(...T), counter int) func()
	Once(eventName string, callback func(...T)) func()
	Emit(eventName string, data ...T)
	Off(eventName string)
	OffAll()
	Notify(sender runner.Runner[T], name string, data ...T)
}

type EnvironmentInfo struct {
	BuildType string
	Platform  string
	Arch      string
}

func getEvents[T any](ctx context.Context) Events[T] {
	if ctx == nil {
		pc, _, _, _ := goruntime.Caller(1)
		funcName := goruntime.FuncForPC(pc).Name()
		log.Fatalf("cannot call '%s': %s", funcName, contextError)
	}
	result := ctx.Value("events")
	if result != nil {
		return result.(Events[T])
	}
	pc, _, _, _ := goruntime.Caller(1)
	funcName := goruntime.FuncForPC(pc).Name()
	log.Fatalf("cannot call '%s': %s", funcName, contextError)
	return nil
}

// EventsOn registers a listener for the given event name. It returns a function to cancel the listener
func EventsOn[T any](ctx context.Context, eventName string, callback func(optionalData ...T)) func() {
	events := getEvents[T](ctx)
	return events.On(eventName, callback)
}

// EventsOff unregisters a listener for the given event name, optionally multiple listeners can be unregistered via `additionalEventNames`
func EventsOff[T any](ctx context.Context, eventName string, additionalEventNames ...string) {
	events := getEvents[T](ctx)
	events.Off(eventName)

	if len(additionalEventNames) > 0 {
		for _, eventName := range additionalEventNames {
			events.Off(eventName)
		}
	}
}

// EventsOff unregisters a listener for the given event name, optionally multiple listeners can be unregistered via `additionalEventNames`
func EventsOffAll[T any](ctx context.Context) {
	events := getEvents[T](ctx)
	events.OffAll()
}

// EventsOnce registers a listener for the given event name. After the first callback, the
// listener is deleted. It returns a function to cancel the listener
func EventsOnce[T any](ctx context.Context, eventName string, callback func(optionalData ...T)) func() {
	events := getEvents[T](ctx)
	return events.Once(eventName, callback)
}

// EventsOnMultiple registers a listener for the given event name, that may be called a maximum of 'counter' times. It returns a function
// to cancel the listener
func EventsOnMultiple[T any](ctx context.Context, eventName string, callback func(optionalData ...T), counter int) func() {
	events := getEvents[T](ctx)
	return events.OnMultiple(eventName, callback, counter)
}

// EventsEmit pass through
func EventsEmit[T any](ctx context.Context, eventName string, optionalData ...T) {
	events := getEvents[T](ctx)
	events.Emit(eventName, optionalData...)
}
