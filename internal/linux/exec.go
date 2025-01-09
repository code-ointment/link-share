package linux

/*
* Wrapper around exec.Cmd.  Objective is to run a binary, collect it's
* stdout/err, plus exit values.
 */
import (
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"syscall"
)

/*
* Collect the results of running a command here.
 */
type RunResult struct {
	Stdout   string // stdout of the executed binary
	Stderr   string // stderr of the executed binary
	Err      error  // error or nil
	ExitCode int    // exit code of the executed binary
	Cmd      *exec.Cmd
	stdoutIn io.ReadCloser
	stderrIn io.ReadCloser
}

/*
* Collect output of the passed reader and return as a string.
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
			// io.EOF is not an error in this case.
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
* cmdLine[0] is binary and balance of array is arguments.
 */
func Run(cmdLine []string) *RunResult {

	var cmd *exec.Cmd
	r := RunResult{}

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
* Return the pid of the Run.
 */
func (er *RunResult) GetPid() int {
	return er.Cmd.Process.Pid
}
