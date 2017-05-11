// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"

	ui "github.com/gizak/termui"
	"github.com/tsuru/tsuru/cmd"
)

type Dashboard struct{}

func (c *Dashboard) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "dashboard",
		Usage:   "dashboard",
		Desc:    "Displays a dashboard of information about the user and his/her apps",
		MaxArgs: 0,
	}
}

func (c *Dashboard) Run(context *cmd.Context, client *cmd.Client) error {
	err := c.startDashboard(client)
	if err != nil {
		return err
	}
	return nil
}

func (c *Dashboard) userEmail(client *cmd.Client) (string, error) {
	u, err := cmd.GetURL("/users/info")
	if err != nil {
		return "", err
	}
	request, _ := http.NewRequest("GET", u, nil)
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var r struct{ Email string }
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return "", err
	}
	return r.Email, nil
}

type apps []app

func (a apps) Len() int           { return len(a) }
func (a apps) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a apps) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (c *Dashboard) startDashboard(client *cmd.Client) error {
	usrEmail, err := c.userEmail(client)
	if err != nil {
		return err
	}
	appsList, err := getApps(client)
	if err != nil {
		return err
	}
	const height = 22
	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()
	email := ui.NewPar(usrEmail)
	email.Height = 3
	email.BorderLabel = "User"
	help := ui.NewPar("To exit hit 'q' key on keyboard")
	help.Height = 3
	help.BorderLabel = "Help"

	targetList := [][]string{[]string{"target", "set"}}
	targetListUI := ui.NewTable()
	targetListUI.Rows = targetList
	targetListUI.Height = height

	apps := [][]string{[]string{"#", "app-list"}}
	var appIndex = 1
	sort.Sort(appsList)
	for _, v := range appsList {
		row := []string{fmt.Sprintf("%v", appIndex), v.Name}
		apps = append(apps, row)
		appIndex++
	}
	appsUI := ui.NewTable()
	appsUI.Rows = apps
	appsUI.Height = height

	units := ui.NewBarChart()
	data := []int{}
	unitsLabels := []string{}
	var unitsIndex = 1
	for _, a := range appsList {
		unitsLabels = append(unitsLabels, fmt.Sprintf("#%v", unitsIndex))
		data = append(data, len(a.Units))
		unitsIndex++
	}

	units.Data = data
	units.Height = 11
	units.BorderLabel = "Units"
	units.DataLabels = unitsLabels
	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(6, 0, email),
			ui.NewCol(6, 0, help),
		),
		ui.NewRow(
			ui.NewCol(6, 0, targetListUI),
			ui.NewCol(6, 0, appsUI),
		),
		ui.NewRow(
			ui.NewCol(12, 0, units),
		),
	)
	ui.Body.Align()
	ui.Render(ui.Body)
	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		ui.Body.Width = ui.TermWidth()
		ui.Body.Align()
		ui.Clear()
		ui.Render(ui.Body)
	})
	ui.Loop()
	return nil
}

func getApps(client *cmd.Client) (apps, error) {
	filter := appFilter{}
	qs, err := filter.queryString(client)
	if err != nil {
		return nil, err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps?%s", qs.Encode()))
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNoContent {
		return nil, err
	}
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var appsList apps
	err = json.Unmarshal(result, &appsList)
	if err != nil {
		return nil, err
	}
	return appsList, nil
}
