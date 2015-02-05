// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/tsuru/tsuru/fs"
	"github.com/tsuru/tsuru/fs/fstest"
	"launchpad.net/gocheck"
)

func (s *S) TestFileSystem(c *gocheck.C) {
	fsystem = &fstest.RecordingFs{}
	c.Assert(filesystem(), gocheck.DeepEquals, fsystem)
	fsystem = nil
	c.Assert(filesystem(), gocheck.DeepEquals, fs.OsFs{})
}
