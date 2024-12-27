package linux

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"syscall"
)

/*
* Results of the Exec.
* This is not designed to handle really large Stdout/Stderr strings.
 */
type ExecReturn struct {
	Stdout   string    // stdout of the executed binary
	Stderr   string    // stderr of the executed binary
	Err      error     // error or nil
	ExitCode int       // exit code of the executed binary
	Cmd      *exec.Cmd // cmd associated with the ExecReturn
	stdoutIn io.ReadCloser
	stderrIn io.ReadCloser
}

/*
* Run an executable and collect its output.  Should work anywhere but parking
* it in linux for the time being.
 */
func readOutput(r io.Reader) (string, error) {

	var out []byte
	buf := make([]byte, 1024) //1K line is probably OK
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			if err != nil {
				return string(out), err
			}
		}

		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return string(out), err
		}
	}
}

/*
* Extract exit code from error if possible.
 */
func getExitCodeFromError(err error) int {

	if exitError, ok := err.(*exec.ExitError); ok {
		ws := exitError.Sys().(syscall.WaitStatus)
		return ws.ExitStatus()
	}
	return 0
}

/*
* More typical exec function.  cmdLine[0] is binary and balance of array is
* arguments.
 */
func Exec(cmdLine []string) *ExecReturn {

	var cmd *exec.Cmd
	r := ExecReturn{}

	if len(cmdLine) > 1 {
		cmd = exec.Command(cmdLine[0], cmdLine[1:]...)
	} else {
		cmd = exec.Command(cmdLine[0])
	}
	r.Cmd = cmd

	var errStdout, errStderr error
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()
	r.Err = cmd.Start()
	if r.Err != nil {
		r.ExitCode = getExitCodeFromError(r.Err)
		slog.Error("linux.Exec failed", "error", r.Err)
		return &r
	}

	// Drain stdout in background.  Two separate variables in the same
	// struct should be OK in this multi-threaded sceanrio.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		r.Stdout, errStdout = readOutput(stdoutIn)
		wg.Done()
	}()

	// Drain stdin
	r.Stderr, errStderr = readOutput(stderrIn)

	wg.Wait()
	r.Err = cmd.Wait()

	// Check all error returns
	errors := []error{r.Err, errStderr, errStdout}
	for _, e := range errors {
		if e != nil {
			slog.Warn("exec failed", "process", cmdLine[0],
				"error", e,
				"stdout", r.Stdout, "stderr", r.Stderr)
			r.Err = e
			r.ExitCode = getExitCodeFromError(r.Err)
			return &r
		}
	}

	ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
	r.ExitCode = ws.ExitStatus()

	return &r
}

/*
* Similar to above.  'input' is written to exec'd processes stdin.
 */
func ExecWithInput(cmdLine []string, input string) *ExecReturn {

	var cmd *exec.Cmd
	r := ExecReturn{}

	if len(cmdLine) > 1 {
		cmd = exec.Command(cmdLine[0], cmdLine[1:]...)
	} else {
		cmd = exec.Command(cmdLine[0])
	}

	buffer := bytes.Buffer{}
	buffer.Write([]byte(input))
	cmd.Stdin = &buffer

	var errStdout, errStderr error
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()
	r.Err = cmd.Start()
	if r.Err != nil {
		r.ExitCode = getExitCodeFromError(r.Err)
		slog.Error("linux.Exec failed", "error", r.Err)
		return &r
	}
	r.Cmd = cmd

	// Drain stdout in background.  Two separate variables in the same
	// struct should be OK in this multi-threaded sceanrio.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		r.Stdout, errStdout = readOutput(stdoutIn)
		wg.Done()
	}()

	// Drain stdin
	r.Stderr, errStderr = readOutput(stderrIn)

	wg.Wait()
	r.Err = cmd.Wait()

	// Check all error returns
	errors := []error{r.Err, errStderr, errStdout}
	for _, e := range errors {
		if e != nil {
			slog.Warn("failed", "process", cmdLine[0],
				"error", e,
				"stdout", r.Stdout, "stderr", r.Stderr)
			r.Err = e
			r.ExitCode = getExitCodeFromError(r.Err)
			return &r
		}
	}

	ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
	r.ExitCode = ws.ExitStatus()

	return &r
}

func ExecStart(cmdLine []string) (*ExecReturn, error) {

	var cmd *exec.Cmd
	r := ExecReturn{}

	if len(cmdLine) > 1 {
		cmd = exec.Command(cmdLine[0], cmdLine[1:]...)
	} else {
		cmd = exec.Command(cmdLine[0])
	}
	r.Cmd = cmd
	r.stderrIn, _ = r.Cmd.StderrPipe()
	r.stdoutIn, _ = r.Cmd.StdoutPipe()

	r.Err = cmd.Start()
	if r.Err != nil {
		r.ExitCode = getExitCodeFromError(r.Err)
		slog.Error("linux.Exec failed", "error", r.Err)
		return &r, r.Err
	}

	return &r, nil
}

func (er *ExecReturn) Wait() *ExecReturn {

	// Drain stdout in background.  Two separate variables in the same
	// struct should be OK in this multi-threaded sceanrio.
	var errStdout, errStderr error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		er.Stdout, errStdout = readOutput(er.stdoutIn)
		wg.Done()
	}()

	// Drain stdin
	er.Stderr, errStderr = readOutput(er.stderrIn)

	wg.Wait()
	er.Err = er.Cmd.Wait()
	//
	// We're skipping checking er.Err as it will almost always be a complaint
	// about 'no waiting child processes'.  Termination happens very quickly
	// so there's nothing to wait on.
	//
	errors := []error{errStderr, errStdout}
	for _, e := range errors {
		if e != nil {
			slog.Info("wait error", "process", er.Cmd.Path,
				"error", e, "stdout", er.Stdout, "stderr", er.Stderr)
			er.Err = e
			er.ExitCode = getExitCodeFromError(er.Err)
			return er
		}
	}

	if er.Err != nil {
		slog.Debug("error",
			"Type allowed", fmt.Sprintf("%T", er.Err))
		return er
	}

	ws := er.Cmd.ProcessState.Sys().(syscall.WaitStatus)
	er.ExitCode = ws.ExitStatus()
	return er
}

/*
* Return the pid of the process we just launched.
 */
func (er *ExecReturn) GetPid() int {
	return er.Cmd.Process.Pid
}
