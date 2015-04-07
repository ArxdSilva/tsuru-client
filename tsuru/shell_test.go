// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/errors"
	"gopkg.in/check.v1"
)

func (s *S) TestShellToContainerCmdInfo(c *check.C) {
	var command ShellToContainerCmd
	info := command.Info()
	c.Assert(info, check.NotNil)
}

func (s *S) TestShellToContainerCmdRunWithApp(c *check.C) {
	var closeClientConn func()
	guesser := cmdtest.FakeGuesser{Name: "myapp"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/apps/myapp/shell" && r.Method == "GET" && r.Header.Get("Authorization") == "bearer abc123" {
			c.Assert(r.URL.Query().Get("term"), check.Equals, os.Getenv("TERM"))
			conn, _, err := w.(http.Hijacker).Hijack()
			c.Assert(err, check.IsNil)
			conn.Write([]byte("hello my friend\n"))
			conn.Write([]byte("glad to see you here\n"))
			closeClientConn()
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()
	closeClientConn = server.CloseClientConnections
	target := "http://" + server.Listener.Addr().String()
	targetRecover := cmdtest.SetTargetFile(c, []byte(target))
	defer cmdtest.RollbackFile(targetRecover)
	tokenRecover := cmdtest.SetTokenFile(c, []byte("abc123"))
	defer cmdtest.RollbackFile(tokenRecover)
	var stdout, stderr, stdin bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  &stdin,
	}
	var command ShellToContainerCmd
	command.GuessingCommand = cmd.GuessingCommand{G: &guesser}
	err := command.Flags().Parse(true, []string{"-a", "myapp"})
	c.Assert(err, check.IsNil)
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(http.DefaultClient, &context, manager)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "hello my friend\nglad to see you here\n")
}

func (s *S) TestShellToContainerWithUnit(c *check.C) {
	var closeClientConn func()
	guesser := cmdtest.FakeGuesser{Name: "myapp"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/apps/myapp/shell" && r.Method == "GET" && r.Header.Get("Authorization") == "bearer abc123" {
			c.Assert(r.URL.Query().Get("unit"), check.Equals, "containerid")
			conn, _, err := w.(http.Hijacker).Hijack()
			c.Assert(err, check.IsNil)
			conn.Write([]byte("hello my friend\n"))
			conn.Write([]byte("glad to see you here\n"))
			closeClientConn()
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()
	closeClientConn = server.CloseClientConnections
	target := "http://" + server.Listener.Addr().String()
	targetRecover := cmdtest.SetTargetFile(c, []byte(target))
	defer cmdtest.RollbackFile(targetRecover)
	tokenRecover := cmdtest.SetTokenFile(c, []byte("abc123"))
	defer cmdtest.RollbackFile(tokenRecover)
	var stdout, stderr, stdin bytes.Buffer
	context := cmd.Context{
		Args:   []string{"containerid"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  &stdin,
	}
	var command ShellToContainerCmd
	command.GuessingCommand = cmd.GuessingCommand{G: &guesser}
	err := command.Flags().Parse(true, []string{"-a", "myapp"})
	c.Assert(err, check.IsNil)
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(http.DefaultClient, &context, manager)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "hello my friend\nglad to see you here\n")
}

func (s *S) TestShellToContainerCmdNoToken(c *check.C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "You must provide a valid Authorization header", http.StatusUnauthorized)
	}))
	defer server.Close()
	target := "http://" + server.Listener.Addr().String()
	targetRecover := cmdtest.SetTargetFile(c, []byte(target))
	defer cmdtest.RollbackFile(targetRecover)
	var stdin, stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"af3332d"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  &stdin,
	}
	var command ShellToContainerCmd
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Unauthorized")
	e, ok := err.(*errors.HTTP)
	c.Assert(ok, check.Equals, true)
	c.Assert(e.Code, check.Equals, http.StatusUnauthorized)
}

func (s *S) TestShellToContainerCmdAppNotFound(c *check.C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not found", http.StatusNotFound)
	}))
	defer server.Close()
	target := "http://" + server.Listener.Addr().String()
	targetRecover := cmdtest.SetTargetFile(c, []byte(target))
	defer cmdtest.RollbackFile(targetRecover)
	var stdin, stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"af3332d"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  &stdin,
	}
	var command ShellToContainerCmd
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "App tsuru not found")
	e, ok := err.(*errors.HTTP)
	c.Assert(ok, check.Equals, true)
	c.Assert(e.Code, check.Equals, http.StatusNotFound)
}

func (s *S) TestShellToContainerCmdSmallData(c *check.C) {
	var closeClientConn func()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		c.Assert(err, check.IsNil)
		conn.Write([]byte("hello"))
		closeClientConn()
	}))
	defer server.Close()
	closeClientConn = server.CloseClientConnections
	target := "http://" + server.Listener.Addr().String()
	targetRecover := cmdtest.SetTargetFile(c, []byte(target))
	defer cmdtest.RollbackFile(targetRecover)
	var stdout, stderr, stdin bytes.Buffer
	context := cmd.Context{
		Args:   []string{"af3332d"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  &stdin,
	}
	var command ShellToContainerCmd
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "hello")
}

func (s *S) TestShellToContainerCmdLongNoNewLine(c *check.C) {
	var closeClientConn func()
	expected := fmt.Sprintf("%0200s", "x")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		c.Assert(err, check.IsNil)
		conn.Write([]byte(expected))
		closeClientConn()
	}))
	defer server.Close()
	closeClientConn = server.CloseClientConnections
	target := "http://" + server.Listener.Addr().String()
	targetRecover := cmdtest.SetTargetFile(c, []byte(target))
	defer cmdtest.RollbackFile(targetRecover)
	var stdout, stderr, stdin bytes.Buffer
	context := cmd.Context{
		Args:   []string{"af3332d"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  &stdin,
	}
	var command ShellToContainerCmd
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestShellToContainerCmdConnectionRefused(c *check.C) {
	server := httptest.NewServer(nil)
	addr := server.Listener.Addr().String()
	server.Close()
	targetRecover := cmdtest.SetTargetFile(c, []byte("http://"+addr))
	defer cmdtest.RollbackFile(targetRecover)
	tokenRecover := cmdtest.SetTokenFile(c, []byte("abc123"))
	defer cmdtest.RollbackFile(tokenRecover)
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"af3332d"},
		Stdout: &buf,
		Stderr: &buf,
		Stdin:  &buf,
	}
	var command ShellToContainerCmd
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	opErr, ok := err.(*net.OpError)
	c.Assert(ok, check.Equals, true)
	c.Assert(opErr.Net, check.Equals, "tcp")
	c.Assert(opErr.Op, check.Equals, "dial")
	c.Assert(opErr.Addr.String(), check.Equals, addr)
}
