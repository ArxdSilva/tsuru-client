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
	"strconv"

	ui "github.com/gizak/termui"
	"github.com/tsuru/tsuru/cmd"
)

type Dashboard struct{}

func (c *Dashboard) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "dashboard",
		Usage:   "dashboard",
		Desc:    "Displays a dashboard of useful information about the user's apps",
		MaxArgs: 0,
	}
}

func (c *Dashboard) Run(context *cmd.Context, client *cmd.Client) error {
	err := startDashboard(client)
	if err != nil {
		return err
	}
	return nil
}

func getUserEmail(client *cmd.Client) (string, error) {
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

func startDashboard(client *cmd.Client) error {
	usrEmail, err := getUserEmail(client)
	if err != nil {
		return err
	}
	appsList, err := getApps(client)
	if err != nil {
		return err
	}
	const height = 11
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

	currentTarget := ui.NewPar("target example")
	currentTarget.Height = 3
	currentTarget.BorderLabel = "Tsuru Target"

	apps := []string{}
	var appIndex = 1
	sort.Sort(appsList)
	for _, v := range appsList {
		item := fmt.Sprintf("[%v] %s", appIndex, v.Name)
		apps = append(apps, item)
		appIndex++
	}
	appsUI := ui.NewList()
	appsUI.BorderLabel = "Apps"
	appsUI.Height = height

	units := ui.NewBarChart()
	data := []int{}
	unitsLabels := []string{}
	var unitsIndex = 1
	for _, a := range appsList {
		unitsLabels = append(unitsLabels, strconv.Itoa(unitsIndex))
		data = append(data, len(a.Units))
		unitsIndex++
	}
	units.Height = height
	units.BorderLabel = "Units"

	poolUI := ui.NewList()
	poolData := []string{}
	poolList, err := getPools(client)
	if err != nil {
		return err
	}
	poolUI.BorderLabel = "    Pool    |    Kind    |    Provisioner"
	poolUI.Height = height
	for _, p := range poolList {
		var pData string
		if p.Public {
			pData = fmt.Sprintf("    %s    |   public  |   %s", p.Name, p.Provisioner)
		} else if p.Default {
			pData = fmt.Sprintf("    %s    |   default  |   %s", p.Name, p.Provisioner)
		} else {
			pData = fmt.Sprintf("    %s    |   -  |   %s", p.Name, p.Provisioner)
		}
		poolData = append(poolData, pData)
	}

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(4, 0, email),
			ui.NewCol(4, 0, help),
			ui.NewCol(4, 0, currentTarget),
		),
		ui.NewRow(
			ui.NewCol(6, 0, appsUI),
			ui.NewCol(6, 0, poolUI),
		),
		ui.NewRow(
			ui.NewCol(12, 0, units),
		),
	)
	ui.Body.Align()
	draw := func(t int) {
		appsUI.Items = apps[t%(len(apps)):]
		units.Data = data[t%(len(data)):]
		units.DataLabels = unitsLabels[t%(len(unitsLabels)):]
		poolUI.Items = poolData[t%(len(poolData)):]
		ui.Render(ui.Body)
	}
	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		ui.Body.Width = ui.TermWidth()
		ui.Body.Align()
		ui.Clear()
		ui.Render(ui.Body)
	})
	ui.Handle("/timer/1s", func(e ui.Event) {
		t := e.Data.(ui.EvtTimer)
		draw(int(t.Count))
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

func getPools(client *cmd.Client) ([]Pool, error) {
	url, err := cmd.GetURL("/pools")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	defer resp.Body.Close()
	var pools []Pool
	err = json.NewDecoder(resp.Body).Decode(&pools)
	if err != nil {
		return nil, err
	}
	return pools, nil
}
