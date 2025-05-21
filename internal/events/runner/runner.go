package runner

import "context"

type Runner interface {
	Run(ctx context.Context) error
	RunMainLoop()
	Hide()
	Show()
	Quit()

	// Events
	Notify(name string, data ...interface{})
}
