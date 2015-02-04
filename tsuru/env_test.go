// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"launchpad.net/gocheck"
)

func (s *S) TestEnvGetInfo(c *gocheck.C) {
	e := envGet{}
	i := e.Info()
	desc := `retrieve environment variables for an app.

If you don't provide the app name, tsuru will try to guess it.`
	c.Assert(i.Name, gocheck.Equals, "env-get")
	c.Assert(i.Usage, gocheck.Equals, "env-get [-a/--app appname] [ENVIRONMENT_VARIABLE1] [ENVIRONMENT_VARIABLE2] ...")
	c.Assert(i.Desc, gocheck.Equals, desc)
	c.Assert(i.MinArgs, gocheck.Equals, 0)
}

func (s *S) TestEnvGetRun(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_HOST", "value": "somehost", "public": true}]`
	result := "DATABASE_HOST=somehost\n"
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: jsonResult, Status: http.StatusOK}}, nil, manager)
	command := envGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, result)
}

func (s *S) TestEnvGetRunWithMultipleParams(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_HOST", "value": "somehost", "public": true}, {"name": "DATABASE_USER", "value": "someuser", "public": true}]`
	result := "DATABASE_HOST=somehost\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: jsonResult, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			want := `["DATABASE_HOST","DATABASE_USER"]` + "\n"
			defer req.Body.Close()
			got, err := ioutil.ReadAll(req.Body)
			c.Assert(err, gocheck.IsNil)
			return req.URL.Path == "/apps/someapp/env" && req.Method == "GET" && string(got) == want
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, result)
}

func (s *S) TestEnvGetAlwaysPrintInAlphabeticalOrder(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_USER", "value": "someuser", "public": true}, {"name": "DATABASE_HOST", "value": "somehost", "public": true}]`
	result := "DATABASE_HOST=somehost\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: jsonResult, Status: http.StatusOK}}, nil, manager)
	command := envGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, result)
}

func (s *S) TestEnvGetPrivateVariables(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_USER", "value": "someuser", "public": true}, {"name": "DATABASE_HOST", "value": "somehost", "public": false}]`
	result := "DATABASE_HOST=*** (private variable)\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: jsonResult, Status: http.StatusOK}}, nil, manager)
	command := envGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, result)
}

func (s *S) TestEnvGetWithoutTheFlag(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_HOST", "value": "somehost", "public": true}, {"name": "DATABASE_USER", "value": "someuser", "public": true}]`
	result := "DATABASE_HOST=somehost\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: jsonResult, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/apps/seek/env" && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "seek"}
	err := (&envGet{cmd.GuessingCommand{G: fake}}).Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, result)
}

func (s *S) TestEnvSetInfo(c *gocheck.C) {
	e := envSet{}
	i := e.Info()
	desc := `set environment variables for an app.

If you don't provide the app name, tsuru will try to guess it.`
	c.Assert(i.Name, gocheck.Equals, "env-set")
	c.Assert(i.Usage, gocheck.Equals, "env-set <NAME=value> [NAME=value] ... [-a/--app appname]")
	c.Assert(i.Desc, gocheck.Equals, desc)
	c.Assert(i.MinArgs, gocheck.Equals, 1)
}

func (s *S) TestEnvSetRun(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST=somehost"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			want := `{"DATABASE_HOST":"somehost"}` + "\n"
			defer req.Body.Close()
			got, err := ioutil.ReadAll(req.Body)
			c.Assert(err, gocheck.IsNil)
			return req.URL.Path == "/apps/someapp/env" && req.Method == "POST" && string(got) == want
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestEnvSetRunWithMultipleParams(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST=somehost", "DATABASE_USER=user"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: string(result), Status: http.StatusOK}}, nil, manager)
	command := envSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestEnvSetValues(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args: []string{
			"DATABASE_HOST=some host",
			"DATABASE_USER=root",
			"DATABASE_PASSWORD=.1234..abc",
			"http_proxy=http://myproxy.com:3128/",
			"VALUE_WITH_EQUAL_SIGN=http://wholikesquerystrings.me/?tsuru=awesome",
			"BASE64_STRING=t5urur0ck5==",
		},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			want := map[string]string{
				"DATABASE_HOST":         "some host",
				"DATABASE_USER":         "root",
				"DATABASE_PASSWORD":     ".1234..abc",
				"http_proxy":            "http://myproxy.com:3128/",
				"VALUE_WITH_EQUAL_SIGN": "http://wholikesquerystrings.me/?tsuru=awesome",
				"BASE64_STRING":         "t5urur0ck5==",
			}
			defer req.Body.Close()
			var got map[string]string
			err := json.NewDecoder(req.Body).Decode(&got)
			c.Assert(err, gocheck.IsNil)
			c.Assert(got, gocheck.DeepEquals, want)
			return req.URL.Path == "/apps/someapp/env" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestEnvSetWithoutFlag(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST=somehost", "DATABASE_USER=user"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/apps/otherapp/env" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "otherapp"}
	err = (&envSet{cmd.GuessingCommand{G: fake}}).Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestEnvSetInvalidParameters(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST", "somehost"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := envSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, envSetValidationMessage)
}

func (s *S) TestEnvUnsetInfo(c *gocheck.C) {
	e := envUnset{}
	i := e.Info()
	desc := `unset environment variables for an app.

If you don't provide the app name, tsuru will try to guess it.`
	c.Assert(i.Name, gocheck.Equals, "env-unset")
	c.Assert(i.Usage, gocheck.Equals, "env-unset <ENVIRONMENT_VARIABLE1> [ENVIRONMENT_VARIABLE2] ... [ENVIRONMENT_VARIABLEN] [-a/--app appname]")
	c.Assert(i.Desc, gocheck.Equals, desc)
	c.Assert(i.MinArgs, gocheck.Equals, 1)
}

func (s *S) TestEnvUnsetRun(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully unset\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			want := `["DATABASE_HOST"]` + "\n"
			defer req.Body.Close()
			got, err := ioutil.ReadAll(req.Body)
			c.Assert(err, gocheck.IsNil)
			return req.URL.Path == "/apps/someapp/env" && req.Method == "DELETE" && string(got) == want
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}
func (s *S) TestEnvUnsetWithoutFlag(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully unset\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/apps/otherapp/env" && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "otherapp"}
	err = (&envUnset{cmd.GuessingCommand{G: fake}}).Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestRequestEnvURL(c *gocheck.C) {
	result := "DATABASE_HOST=somehost"
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	args := []string{"DATABASE_HOST"}
	g := cmd.GuessingCommand{G: &cmdtest.FakeGuesser{Name: "someapp"}}
	b, err := requestEnvURL("GET", g, args, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(b, gocheck.DeepEquals, []byte(result))
}
