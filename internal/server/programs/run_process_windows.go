//go:build windows

package programs

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gar-id/queued/internal/server/config/caches"
	"github.com/gar-id/queued/tools"
)

func runProcess(programName, processName string, processIndex int) (programCmd *exec.Cmd) {
	// Prep to execute program
	var apps string
	var runChunk []string

	if strings.Contains(caches.Data.ProgramConfig[programName].Command, "bash -c") {
		apps = "bash"
		runChunk = append(runChunk, "-c")
		tempRun := strings.ReplaceAll(caches.Data.ProgramConfig[programName].Command, "bash -c ", "")
		tempRun, _ = strings.CutPrefix(tempRun, "'")
		tempRun, _ = strings.CutPrefix(tempRun, "\"")
		tempRun, _ = strings.CutSuffix(tempRun, "'")
		tempRun, _ = strings.CutSuffix(tempRun, "\"")
		runChunk = append(runChunk, tempRun)
	} else if strings.Contains(caches.Data.ProgramConfig[programName].Command, "sh -c") {
		apps = "sh"
		runChunk = append(runChunk, "-c")
		tempRun := strings.ReplaceAll(caches.Data.ProgramConfig[programName].Command, "sh -c ", "")
		tempRun, _ = strings.CutPrefix(tempRun, "'")
		tempRun, _ = strings.CutPrefix(tempRun, "\"")
		tempRun, _ = strings.CutSuffix(tempRun, "'")
		tempRun, _ = strings.CutSuffix(tempRun, "\"")
		runChunk = append(runChunk, tempRun)
	} else {
		runChunk = strings.Split(caches.Data.ProgramConfig[programName].Command, " ")
		apps = runChunk[0]
		runChunk[0] = ""
		runChunk = append(runChunk[:0], runChunk[0+1:]...)
	}
	programCmd = exec.Command(apps, runChunk...)

	// Where to store stderr
	if caches.Data.ProgramConfig[programName].Stderr != "" && !strings.Contains(caches.Data.ProgramConfig[programName].Stderr, "/dev/stderr") {
		stderrLog := tools.TextTemplate(caches.Data.ProgramConfig[programName].Stderr, caches.Data.ProgramConfig[programName].Process[processIndex])
		errLog, err := os.OpenFile(stderrLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			tools.ZapLogger("both").Fatal(err.Error())
		}
		defer errLog.Close()
		programCmd.Stderr = errLog
	} else {
		programCmd.Stderr = os.Stderr
	}

	// Where to store stdout
	if caches.Data.ProgramConfig[programName].Stdout != "" && !strings.Contains(caches.Data.ProgramConfig[programName].Stdout, "/dev/stdout") {
		stdoutLog := tools.TextTemplate(caches.Data.ProgramConfig[programName].Stdout, caches.Data.ProgramConfig[programName].Process[processIndex])
		outLog, err := os.OpenFile(stdoutLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			tools.ZapLogger("both").Fatal(err.Error())
		}
		defer outLog.Close()
		programCmd.Stdout = outLog
	} else {
		programCmd.Stdout = os.Stdout
	}

	// Run as user not supported for windows now
	// u, err := user.Lookup(caches.Data.ProgramConfig[programName].User)
	// if err != nil {
	// 	tools.ZapLogger("both").Error(fmt.Sprintf("%v: not found. %v process will be run under %v", err, processName, os.Getenv("USER")))
	// } else {
	// 	uid, _ := strconv.Atoi(u.Uid)
	// 	gid, _ := strconv.Atoi(u.Gid)
	// 	programCmd.SysProcAttr = &syscall.SysProcAttr{}
	// 	programCmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	// }

	// Add env from config
	programCmd.Env = os.Environ()
	programCmd.Env = append(programCmd.Env, caches.Data.ProgramConfig[programName].Env...)

	// Start command
	// programCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := programCmd.Start()
	if err != nil {
		tools.ZapLogger("both").Warn(err.Error())
	}

	// Update status
	caches.Data.Do(func() {
		caches.Data.ProgramConfig[programName].Process[processIndex].LastStart = time.Now()
		caches.Data.ProgramConfig[programName].Process[processIndex].Status = "running"
		caches.ProcessChannel.Do(func() {
			caches.Data.ProgramConfig[programName].Process[processIndex].PID = &programCmd.Process.Pid
		})
	})

	// Start message
	var logMessage string
	logMessage = fmt.Sprintf(color.GreenString("%s"), caches.Data.ProgramConfig[programName].Process[processIndex].Status)
	logMessage = fmt.Sprintf("%v is %v", processName, logMessage)
	tools.ZapLogger("both").Info(logMessage)

	// Possible race condition
	return programCmd
}
