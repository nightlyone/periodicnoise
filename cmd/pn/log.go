package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/Jimdo/periodicnoise/cmd/internal/syslog"
)

// this enables testing
var logNetwork, logRemoteAddress string

// derive logger
func getLogger(useSyslog bool) (logger io.Writer, err error) {
	if useSyslog {
		logger, err = syslog.Dial(logNetwork, logRemoteAddress, syslog.LOG_DAEMON|syslog.LOG_NOTICE, monitoringEvent)
	} else {
		logger = os.Stderr
		log.SetPrefix(monitoringEvent + ": ")
	}
	if err == nil {
		log.SetOutput(logger)
	}
	return &LineWriter{w: logger}, err
}

func canContinue(expire time.Time, progressed bool, err error) bool {
	if time.Now().After(expire) {
		return false
	}

	if netErr, ok := err.(net.Error); ok && (netErr.Temporary() || netErr.Timeout()) {
		return true
	} else if opErr, ok := err.(*net.OpError); ok && progressed {
		if errno, ok := opErr.Err.(syscall.Errno); ok {
			// These are only temporary errors, if we could ever connect.
			// If we could never connect, we cannot decide, whether this is a temporary failure,
			// or the address we connect to is simply wrong.
			switch errno {
			case syscall.ECONNREFUSED, syscall.ENETUNREACH, syscall.EHOSTUNREACH:
				return true
			}
		}
	}
	return false
}

// pipe r to logger in the background
func logStream(r io.Reader, logger io.Writer, wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		expire := time.Now().Add(opts.Timeout + 500*time.Millisecond)

		var written int64

		// Retry writing to logstream until it can be fully written.
		// If io.Copy returns nil, everything has been copied
		// successfully and this routine can be stopped.
		for {
			n, err := io.Copy(logger, r)
			written += n
			if err == nil {
				break
			} else if !canContinue(expire, written > 0, err) {
				fmt.Fprintln(os.Stderr, "permanent pn log error:", err)
				break
			}
			fmt.Fprintln(os.Stderr, "transient pn log error:", err)
		}
		wg.Done()
	}()
}

// Connect stderr/stdout of future child to logger and background copy jobs.
func connectOutputs(cmd *exec.Cmd, logger io.Writer, wg *sync.WaitGroup) error {
	if !opts.NoPipeStdout {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return &StartupError{"connecting stdout", err}
		}
		if opts.WrapNagiosPlugin {
			firstbytes = NewCapWriter(8192)
			stdout := io.TeeReader(stdout, firstbytes)
			logStream(stdout, logger, wg)
		} else {
			logStream(stdout, logger, wg)
		}
	} else if opts.WrapNagiosPlugin {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return &StartupError{"connecting stdout", err}
		}
		firstbytes = NewCapWriter(8192)
		logStream(stdout, firstbytes, wg)
	}

	if !opts.NoPipeStderr {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return &StartupError{"connecting stderr", err}
		}
		logStream(stderr, logger, wg)
	}
	return nil
}
