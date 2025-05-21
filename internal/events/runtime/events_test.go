package runtime_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/gar-id/queued/internal/events/runtime"
	"github.com/gar-id/queued/internal/general"
)

type mockLogger struct {
	Log string
}

func (t *mockLogger) Trace(format string, args ...interface{}) {
	t.Log = fmt.Sprintf(format, args...)
}

func Test_EventsOn(t *testing.T) {
	l := &mockLogger{}

	assert := general.NewAssert(t, "EventsOn")
	manager := runtime.NewEvents(l)

	// Test On
	eventName := "test"
	counter := 0
	var wg sync.WaitGroup

	wg.Add(1)

	manager.On(eventName, func(args ...interface{}) {
		// This is called in a goroutine
		counter++

		assert.Equal("test payload", args[0])

		wg.Done()
	})

	manager.Emit(eventName, "test payload")

	wg.Wait()

	assert.Equal(1, counter)
}
