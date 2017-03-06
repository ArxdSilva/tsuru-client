// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import "github.com/tsuru/tsuru/cmd"

type Profile struct{}

func (c *Profile) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "profile",
		Usage: "profile",
		Desc:  `Starts a dashboard that contains basic info about your user.`,
	}
}

func (c *Profile) Run(context *cmd.Context, client *cmd.Client) error {
	// this func will call other funcs that will give this one data to start the UI
	return nil
}
