// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
)

const envSetValidationMessage = `You must specify environment variables in the form "NAME=value".

Example:

  tsuru env-set NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'`

type envGet struct {
	cmd.GuessingCommand
}

func (c *envGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "env-get",
		Usage: "env-get [-a/--app appname] [ENVIRONMENT_VARIABLE1] [ENVIRONMENT_VARIABLE2] ...",
		Desc: `retrieve environment variables for an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 0,
	}
}

func (c *envGet) Run(context *cmd.Context, client *cmd.Client) error {
	b, err := requestEnvURL("GET", c.GuessingCommand, context.Args, client)
	if err != nil {
		return err
	}
	var variables []map[string]interface{}
	err = json.Unmarshal(b, &variables)
	if err != nil {
		return err
	}
	formatted := make([]string, 0, len(variables))
	for _, v := range variables {
		value := "*** (private variable)"
		if v["public"].(bool) {
			value = v["value"].(string)
		}
		formatted = append(formatted, fmt.Sprintf("%s=%s", v["name"], value))
	}
	sort.Strings(formatted)
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))
	return nil
}

type envSet struct {
	cmd.GuessingCommand
}

func (c *envSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "env-set",
		Usage: "env-set <NAME=value> [NAME=value] ... [-a/--app appname]",
		Desc: `set environment variables for an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
}

func (c *envSet) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	raw := strings.Join(context.Args, "\n")
	regex := regexp.MustCompile(`(\w+=[^\n$]+)(\n|$)`)
	decls := regex.FindAllStringSubmatch(raw, -1)
	if len(decls) < 1 || len(decls) != len(context.Args) {
		return errors.New(envSetValidationMessage)
	}
	variables := make(map[string]string, len(decls))
	for _, v := range decls {
		parts := strings.Split(v[1], "=")
		variables[parts[0]] = strings.Join(parts[1:], "=")
	}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(variables)
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/env", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, response.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

type envUnset struct {
	cmd.GuessingCommand
}

func (c *envUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "env-unset",
		Usage: "env-unset <ENVIRONMENT_VARIABLE1> [ENVIRONMENT_VARIABLE2] ... [ENVIRONMENT_VARIABLEN] [-a/--app appname]",
		Desc: `unset environment variables for an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
}

func (c *envUnset) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/env", appName))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(context.Args)
	request, err := http.NewRequest("DELETE", url, &buf)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, response.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

func requestEnvURL(method string, g cmd.GuessingCommand, args []string, client *cmd.Client) ([]byte, error) {
	appName, err := g.Guess()
	if err != nil {
		return nil, err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/env", appName))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(args)
	request, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, err
	}
	r, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
