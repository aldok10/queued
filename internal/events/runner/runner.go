package runner

import "context"

type Runner[T any] interface {
	Run(ctx context.Context) error
	RunMainLoop()
	Hide()
	Show()
	Quit()

	// Events
	Notify(name string, data ...T)
}
