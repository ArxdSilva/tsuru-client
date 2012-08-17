package main

import (
	"bytes"
	"github.com/timeredbull/tsuru/cmd"
	fs_test "github.com/timeredbull/tsuru/fs/testing"
	. "launchpad.net/gocheck"
	"net/http"
	"os/user"
	"path"
)

func (s *S) TestAddKey(c *C) {
	u, err := user.Current()
	c.Assert(err, IsNil)
	p := path.Join(u.HomeDir, ".ssh", "id_rsa.pub")
	expected := "Key successfully added!\n"
	context := cmd.Context{
		Cmds:   []string{},
		Args:   []string{},
		Stdout: manager.Stdout,
		Stderr: manager.Stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: "success", status: http.StatusOK}})
	fs := fs_test.RecordingFs{FileContent: "user-key"}
	command := AddKey{keyReader{fsystem: &fs}}
	err = command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
	c.Assert(fs.HasAction("open "+p), Equals, true)
}

func (s *S) TestAddKeySpecifyingKeyFile(c *C) {
	u, err := user.Current()
	c.Assert(err, IsNil)
	p := path.Join(u.HomeDir, ".ssh", "id_dsa.pub")
	expected := "Key successfully added!\n"
	context := cmd.Context{
		Cmds:   []string{},
		Args:   []string{p},
		Stdout: manager.Stdout,
		Stderr: manager.Stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: "success", status: http.StatusOK}})
	fs := fs_test.RecordingFs{FileContent: "user-key"}
	command := AddKey{keyReader{fsystem: &fs}}
	err = command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
	c.Assert(fs.HasAction("open "+p), Equals, true)
}

func (s *S) TestAddKeyReturnErrorIfTheKeyDoesNotExist(c *C) {
	context := cmd.Context{
		Cmds:   []string{},
		Args:   []string{},
		Stdout: manager.Stdout,
		Stderr: manager.Stderr,
	}
	fs := fs_test.FailureFs{RecordingFs: fs_test.RecordingFs{}}
	command := AddKey{keyReader{fsystem: &fs}}
	err := command.Run(&context, nil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "You need to have a public rsa key")
}

func (s *S) TestAddKeyReturnsProperErrorIfTheGivenKeyFileDoesNotExist(c *C) {
	context := cmd.Context{
		Cmds:   []string{},
		Args:   []string{"/unknown/key.pub"},
		Stdout: manager.Stdout,
		Stderr: manager.Stderr,
	}
	fs := fs_test.FailureFs{RecordingFs: fs_test.RecordingFs{}}
	command := AddKey{keyReader{fsystem: &fs}}
	err := command.Run(&context, nil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "File /unknown/key.pub does not exist!")
	c.Assert(context.Stderr.(*bytes.Buffer).String(), Equals, "File /unknown/key.pub does not exist!\n")
}

func (s *S) TestInfoAddKey(c *C) {
	expected := &cmd.Info{
		Name:  "add",
		Usage: "key add [path/to/key/file.pub]",
		Desc:  "add your public key ($HOME/.ssh/id_rsa.pub by default).",
	}
	c.Assert((&AddKey{}).Info(), DeepEquals, expected)
}

func (s *S) TestRemoveKey(c *C) {
	u, err := user.Current()
	c.Assert(err, IsNil)
	p := path.Join(u.HomeDir, ".ssh", "id_rsa.pub")
	expected := "Key successfully removed!\n"
	context := cmd.Context{
		Cmds:   []string{},
		Args:   []string{},
		Stdout: manager.Stdout,
		Stderr: manager.Stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: "success", status: http.StatusOK}})
	fs := fs_test.RecordingFs{FileContent: "user-key"}
	command := RemoveKey{keyReader{fsystem: &fs}}
	err = command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
	c.Assert(fs.HasAction("open "+p), Equals, true)
}

func (s *S) TestRemoveKeySpecifyingKeyFile(c *C) {
	u, err := user.Current()
	c.Assert(err, IsNil)
	p := path.Join(u.HomeDir, ".ssh", "id_dsa.pub")
	expected := "Key successfully removed!\n"
	context := cmd.Context{
		Cmds:   []string{},
		Args:   []string{p},
		Stdout: manager.Stdout,
		Stderr: manager.Stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: "success", status: http.StatusOK}})
	fs := fs_test.RecordingFs{FileContent: "user-key"}
	command := RemoveKey{keyReader{fsystem: &fs}}
	err = command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(manager.Stdout.(*bytes.Buffer).String(), Equals, expected)
	c.Assert(fs.HasAction("open "+p), Equals, true)
}

func (s *S) TestRemoveKeyReturnErrorIfTheKeyDoesNotExist(c *C) {
	context := cmd.Context{
		Cmds:   []string{},
		Args:   []string{},
		Stdout: manager.Stdout,
		Stderr: manager.Stderr,
	}
	fs := fs_test.FailureFs{RecordingFs: fs_test.RecordingFs{}}
	command := RemoveKey{keyReader{fsystem: &fs}}
	err := command.Run(&context, nil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "You need to have a public rsa key")
}

func (s *S) TestRemoveKeyReturnProperErrorIfTheGivenKeyFileDoesNotExist(c *C) {
	context := cmd.Context{
		Cmds:   []string{},
		Args:   []string{"/unknown/key.pub"},
		Stdout: manager.Stdout,
		Stderr: manager.Stderr,
	}
	fs := fs_test.FailureFs{RecordingFs: fs_test.RecordingFs{}}
	command := RemoveKey{keyReader{fsystem: &fs}}
	err := command.Run(&context, nil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "File /unknown/key.pub does not exist!")
	c.Assert(context.Stderr.(*bytes.Buffer).String(), Equals, err.Error()+"\n")
}

func (s *S) TestInfoRemoveKey(c *C) {
	expected := &cmd.Info{
		Name:  "remove",
		Usage: "key remove [path/to/key/file.pub]",
		Desc:  "remove your public key ($HOME/.id_rsa.pub by default).",
	}
	c.Assert((&RemoveKey{}).Info(), DeepEquals, expected)
}

func (s *S) TestKey(c *C) {
	expect := map[string]interface{}{
		"add":    &AddKey{},
		"remove": &RemoveKey{},
	}
	command := Key{}
	c.Assert(command.Subcommands(), DeepEquals, expect)
}
