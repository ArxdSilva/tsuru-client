// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"github.com/globocom/tsuru/cmd"
	. "launchpad.net/gocheck"
	"net/http"
)

func (s *S) TestAppRun(c *C) {
	*appName = "ble"
	var stdout, stderr bytes.Buffer
	expected := "http.go		http_test.go"
	context := cmd.Context{
		Args:   []string{"ls"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg: "http.go		http_test.go",
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			b := make([]byte, 2)
			req.Body.Read(b)
			return req.URL.Path == "/apps/ble/run" && string(b) == "ls"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&AppRun{}).Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, expected)
}

func (s *S) TestAppRunShouldUseAllSubsequentArgumentsAsArgumentsToTheGivenCommand(c *C) {
	*appName = "ble"
	var stdout, stderr bytes.Buffer
	expected := "-rw-r--r--  1 f  staff  119 Apr 26 18:23 http.go"
	context := cmd.Context{
		Args:   []string{"ls", "-l"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg:    "-rw-r--r--  1 f  staff  119 Apr 26 18:23 http.go",
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			b := make([]byte, 5)
			req.Body.Read(b)
			return req.URL.Path == "/apps/ble/run" && string(b) == "ls -l"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&AppRun{}).Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, expected)
}

func (s *S) TestAppRunWithoutTheFlag(c *C) {
	var stdout, stderr bytes.Buffer
	expected := "-rw-r--r--  1 f  staff  119 Apr 26 18:23 http.go"
	context := cmd.Context{
		Args:   []string{"ls", "-lh"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &conditionalTransport{
		transport{
			msg:    "-rw-r--r--  1 f  staff  119 Apr 26 18:23 http.go",
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			b := make([]byte, 6)
			req.Body.Read(b)
			return req.URL.Path == "/apps/bla/run" && string(b) == "ls -lh"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &FakeGuesser{name: "bla"}
	err := (&AppRun{GuessingCommand{g: fake}}).Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, expected)
}

func (s *S) TestInfoAppRun(c *C) {
	desc := `run a command in all instances of the app, and prints the output.
Notice that you may need quotes to run your command if you want to deal with
input and outputs redirects, and pipes.

If you don't provide the app name, tsuru will try to guess it.
`
	expected := &cmd.Info{
		Name:    "run",
		Usage:   `run <command> [commandarg1] [commandarg2] ... [commandargn] [--app appname]`,
		Desc:    desc,
		MinArgs: 1,
	}
	command := AppRun{}
	c.Assert(command.Info(), DeepEquals, expected)
}
