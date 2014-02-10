package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	flags "github.com/jessevdk/go-flags"
)

func TestCoreLoopRetryExitSuccess(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--retries=3 -- true"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoopRetry(args, &bytes.Buffer{})
	t.Log(output.String())
	if err != nil {
		t.Error("want no error, got", err)
	}
}

func TestCoreLoopRetryExitSuccessFunnyExitCodes(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--retries=3 --monitor-ok=1 -- false"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoopRetry(args, &bytes.Buffer{})
	t.Log(output.String())
	if err != nil {
		t.Error("want no error, got", err)
	}
}

func TestCoreLoopRetryExitError(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--retries=3 -- false"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoopRetry(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if _, ok := err.(*exec.ExitError); ok {
		t.Log("got", err)
	} else {
		t.Error("want exit error, got", err)
	}
}

func TestCoreLoopRetryDidRetry(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	statefile, err := ioutil.TempFile("./testdata", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		os.Remove(statefile.Name())
		statefile.Close()
	}()

	arguments := "--retries=3 -- ./testdata/works-after-two-failures.sh " + statefile.Name()
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	var logstream bytes.Buffer

	log.SetOutput(&output)

	err = CoreLoopRetry(args, &logstream)
	t.Log(output.String())
	t.Log(logstream.String())
	if err != nil {
		t.Error("want no error, got", err)
	}
}

func TestCoreLoopOnceExitSuccess(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "-- true"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoopOnce(args, &bytes.Buffer{})
	t.Log(output.String())
	if err != nil {
		t.Error("want no error, got", err)
	}
}

func TestCoreLoopOnceExitError(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "-- false"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoopOnce(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if _, ok := err.(*exec.ExitError); ok {
		t.Log("got", err)
	} else {
		t.Error("want exit error, got", err)
	}
}

func TestCoreLoopOnceWrongCommand(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "-- /var/run/nonexistant"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoopOnce(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if _, ok := err.(*NotAvailableError); ok {
		t.Log("got", err)
	} else {
		t.Error("want not available error, got", err)
	}
}

func TestCoreLoopOnceHardTimeout(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--timeout=100ms -- sleep 1"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoopOnce(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if timeout, ok := err.(*TimeoutError); ok && !timeout.soft {
		t.Log("got", err)
	} else {
		t.Error("want hard timeout error, got ", err)
	}
}

func TestCoreLoopOnceSoftTimeout(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--timeout=100ms --grace-time=50ms -- sleep 1"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoopOnce(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if timeout, ok := err.(*TimeoutError); ok && timeout.soft {
		t.Log("got", err)
	} else {
		t.Error("want soft timeout error, got ", err)
	}
}
