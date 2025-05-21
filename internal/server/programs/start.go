package programs

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/gar-id/queued/internal/server/config/caches"
	"github.com/gar-id/queued/internal/server/config/types"
	"github.com/gar-id/queued/tools"
)

type processCmd struct {
	Mutex sync.RWMutex
	Cmd   *exec.Cmd
	Exit  bool
}

func (c *processCmd) Do(run func()) {
	c.Mutex.Lock()
	run()
	defer c.Mutex.Unlock()
}

func start(programName, processName string, processIndex int) {
	// Add channel to process
	var stopChan = make(chan bool)
	caches.ProcessChannel.Do(func() {
		var processChannel = types.ProcessChannel{
			StopChannel: &stopChan,
			Name:        processName,
		}
		caches.ProcessChannel.Data[processName] = processChannel
	})

	// // Debugging
	// fmt.Printf("process %v with go id:", processName)
	// fmt.Println(tools.GoID())

	// Init var
	var process processCmd
	process.Exit = false

	caches.Data.Do(func() {
		caches.Data.WaitGroup.Add(1)
	})
	time.Sleep(time.Duration(caches.Data.ProgramConfig[programName].SlowStart) * time.Second)
	// If autorestart is true, then use for loop
	switch caches.Data.ProgramConfig[programName].AutoRestart {
	case true:
		// Possible race condition
		for !process.Exit {
			process.Do(func() {
				process.Cmd = runProcess(programName, processName, processIndex)
			})

			go func() {
				// // Debugging
				// fmt.Printf("process stopchan %v with go id:", processName)
				// fmt.Println(tools.GoID())

				<-stopChan
				// process.Cmd.Process.Kill()
				process.Exit = true
				process.Cmd.Process.Signal(syscall.SIGTERM)
			}()

			err := process.Cmd.Wait()
			logMessage := fmt.Sprintf(color.RedString("%s"), "stopped")
			if err != nil {
				// End log
				logMessage = fmt.Sprintf("%v is %v with %v", processName, logMessage, err.Error())
			} else {
				logMessage = fmt.Sprintf("%v is %v with %v", processName, logMessage, process.Cmd.ProcessState)
			}
			tools.ZapLogger("both").Warn(logMessage)

		}
	default:
		process.Cmd = runProcess(programName, processName, processIndex)

		go func() {
			// // Debugging
			// fmt.Printf("process cmd wait %v with go id:", processName)
			// fmt.Println(tools.GoID())

			process.Do(func() {
				err := process.Cmd.Wait()
				logMessage := fmt.Sprintf(color.RedString("%s"), "stopped")
				if err != nil {
					// End log
					logMessage = fmt.Sprintf("%v is %v with %v", processName, logMessage, err.Error())
				} else {
					logMessage = fmt.Sprintf("%v is %v with %v", processName, logMessage, process.Cmd.ProcessState)
				}
				tools.ZapLogger("both").Warn(logMessage)

			})
		}()

		<-stopChan
		// process.Cmd.Process.Kill()
		process.Exit = true
		process.Cmd.Process.Signal(syscall.SIGTERM)
	}

	// Update status
	caches.Data.Do(func() {
		caches.Data.ProgramConfig[programName].Process[processIndex].Status = "stopped"
		caches.Data.WaitGroup.Done()
	})
}
