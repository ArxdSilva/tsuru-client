// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/globocom/tsuru/cmd"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type ServiceList struct{}

func (s *ServiceList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-list",
		Usage: "service-list",
		Desc:  "Get all available services, and user's instances for this services",
	}
}

func (s *ServiceList) Run(ctx *cmd.Context, client cmd.Doer) error {
	req, err := http.NewRequest("GET", cmd.GetUrl("/services/instances"), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	rslt, err := cmd.ShowServicesInstancesList(b)
	if err != nil {
		return err
	}
	n, err := ctx.Stdout.Write(rslt)
	if n != len(rslt) {
		return errors.New("Failed to write the output of the command")
	}
	return nil
}

type ServiceAdd struct{}

func (sa *ServiceAdd) Info() *cmd.Info {
	usage := `service-add <servicename> <serviceinstancename>
e.g.:

    $ tsuru service-add mongodb tsuru_mongodb

Will add a new instance of the "mongodb" service, named "tsuru_mongodb".`
	return &cmd.Info{
		Name:    "service-add",
		Usage:   usage,
		Desc:    "Create a service instance to one or more apps make use of.",
		MinArgs: 2,
	}
}

func (sa *ServiceAdd) Run(ctx *cmd.Context, client cmd.Doer) error {
	srvName, instName := ctx.Args[0], ctx.Args[1]
	fmtBody := fmt.Sprintf(`{"name": "%s", "service_name": "%s"}`, instName, srvName)
	b := bytes.NewBufferString(fmtBody)
	url := cmd.GetUrl("/services/instances")
	request, err := http.NewRequest("POST", url, b)
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Service successfully added.\n")
	return nil
}

type ServiceBind struct {
	GuessingCommand
}

func (sb *ServiceBind) Run(ctx *cmd.Context, client cmd.Doer) error {
	appName, err := sb.Guess()
	if err != nil {
		return err
	}
	instanceName := ctx.Args[0]
	url := cmd.GetUrl("/services/instances/" + instanceName + "/" + appName)
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var variables []string
	dec := json.NewDecoder(resp.Body)
	msg := fmt.Sprintf("Instance %s successfully binded to the app %s.", instanceName, appName)
	if err = dec.Decode(&variables); err == nil {
		msg += fmt.Sprintf(`

The following environment variables are now available for use in your app:

- %s

For more details, please check the documentation for the service, using service-doc command.
`, strings.Join(variables, "\n- "))
	}
	n, err := fmt.Fprint(ctx.Stdout, msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return io.ErrShortWrite
	}
	return nil
}

func (sb *ServiceBind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "bind",
		Usage: "bind <instancename> [--app appname]",
		Desc: `bind a service instance to an app

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
}

type ServiceUnbind struct{}

func (su *ServiceUnbind) Run(ctx *cmd.Context, client cmd.Doer) error {
	instanceName, appName := ctx.Args[0], ctx.Args[1]
	url := cmd.GetUrl("/services/instances/" + instanceName + "/" + appName)
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("Instance %s successfully unbinded from the app %s.\n", instanceName, appName)
	n, err := fmt.Fprint(ctx.Stdout, msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return errors.New("Failed to write to standard output.\n")
	}
	return nil
}

func (su *ServiceUnbind) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "unbind",
		Usage:   "unbind <instancename> <appname>",
		Desc:    "unbind a service instance from an app",
		MinArgs: 2,
	}
}

type ServiceInstanceStatus struct{}

func (c *ServiceInstanceStatus) Info() *cmd.Info {
	usg := `service-status <serviceinstancename>
e.g.:

    $ tsuru service-status my_mongodb
`
	return &cmd.Info{
		Name:    "service-status",
		Usage:   usg,
		Desc:    "Check status of a given service instance.",
		MinArgs: 1,
	}
}

func (c *ServiceInstanceStatus) Run(ctx *cmd.Context, client cmd.Doer) error {
	instName := ctx.Args[0]
	url := cmd.GetUrl("/services/instances/" + instName + "/status")
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bMsg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	msg := string(bMsg) + "\n"
	n, err := fmt.Fprint(ctx.Stdout, msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return errors.New("Failed to write to standard output.\n")
	}
	return nil
}

type ServiceInfo struct{}

func (c *ServiceInfo) Info() *cmd.Info {
	usg := `service-info <service>
e.g.:

    $ tsuru service-info mongodb
`
	return &cmd.Info{
		Name:    "service-info",
		Usage:   usg,
		Desc:    "List all instances of a service",
		MinArgs: 1,
	}
}

type ServiceInstanceModel struct {
	Name string
	Apps []string
}

func (c *ServiceInfo) Run(ctx *cmd.Context, client cmd.Doer) error {
	serviceName := ctx.Args[0]
	url := cmd.GetUrl("/services/" + serviceName)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var instances []ServiceInstanceModel
	err = json.Unmarshal(result, &instances)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte(fmt.Sprintf("Info for \"%s\"\n", serviceName)))
	if len(instances) > 0 {
		table := cmd.NewTable()
		table.Headers = cmd.Row([]string{"Instances", "Apps"})
		for _, instance := range instances {
			apps := strings.Join(instance.Apps, ", ")
			table.AddRow(cmd.Row([]string{instance.Name, apps}))
		}
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

type ServiceDoc struct{}

func (c *ServiceDoc) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-doc",
		Usage:   "service-doc <servicename>",
		Desc:    "Show documentation of a service",
		MinArgs: 1,
	}
}

func (c *ServiceDoc) Run(ctx *cmd.Context, client cmd.Doer) error {
	sName := ctx.Args[0]
	url := fmt.Sprintf("/services/c/%s/doc", sName)
	url = cmd.GetUrl(url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	ctx.Stdout.Write(result)
	return nil
}

type ServiceRemove struct{}

func (c *ServiceRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-remove",
		Usage:   "service-remove <serviceinstancename>",
		Desc:    "Removes a service instance",
		MinArgs: 1,
	}
}

func (c *ServiceRemove) Run(ctx *cmd.Context, client cmd.Doer) error {
	name := ctx.Args[0]
	url := fmt.Sprintf("/services/c/instances/%s", name)
	url = cmd.GetUrl(url)
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, _ := ioutil.ReadAll(resp.Body)
	result = append(result, []byte("\n")...)
	ctx.Stdout.Write(result)
	return nil
}
