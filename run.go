// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/globocom/tsuru/cmd"
	"io"
	"net/http"
	"strings"
)

type AppRun struct {
	GuessingCommand
}

func (c *AppRun) Info() *cmd.Info {
	desc := `run a command in all instances of the app, and prints the output.
Notice that you may need quotes to run your command if you want to deal with
input and outputs redirects, and pipes.

If you don't provide the app name, tsuru will try to guess it.
`
	return &cmd.Info{
		Name:    "run",
		Usage:   `run <command> [commandarg1] [commandarg2] ... [commandargn] [--app appname]`,
		Desc:    desc,
		MinArgs: 1,
	}
}

func (c *AppRun) Run(context *cmd.Context, client cmd.Doer) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url := cmd.GetUrl(fmt.Sprintf("/apps/%s/run", appName))
	b := strings.NewReader(strings.Join(context.Args, " "))
	request, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	r, err := client.Do(request)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	_, err = io.Copy(context.Stdout, r.Body)
	return err
}
