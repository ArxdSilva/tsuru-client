// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"reflect"

	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	check "gopkg.in/check.v1"
)

type FakeMachineProvisioner struct {
	hostsProvisioned int
}

func (p *FakeMachineProvisioner) ProvisionMachine(opts map[string]interface{}) (*dockermachine.Machine, error) {
	p.hostsProvisioned = p.hostsProvisioned + 1
	return &dockermachine.Machine{Base: &iaas.Machine{CustomData: opts}}, nil
}

func (s *S) TestBuildClusterTable(c *check.C) {
	i := &Installation{CoreCluster: &FakeServiceCluster{}}
	table := i.buildClusterTable()
	expected := `+-----------+---------+---------+
| IP        | State   | Manager |
+-----------+---------+---------+
| 127.0.0.1 | running | true    |
+-----------+---------+---------+
`
	c.Assert(table.String(), check.Equals, expected)
}

func (s *S) TestBuildComponentsTable(c *check.C) {
	i := &Installation{CoreCluster: &FakeServiceCluster{}, Components: []TsuruComponent{&MongoDB{}}}
	table := i.buildComponentsTable()
	expected := `+-----------+-------+----------+
| Component | Ports | Replicas |
+-----------+-------+----------+
| MongoDB   | 8080  | 1        |
+-----------+-------+----------+
`
	c.Assert(table.String(), check.Equals, expected)
}

func (s *S) TestProvisionPool(c *check.C) {
	opt1 := map[string]interface{}{"variable-opt": "opt1"}
	opt2 := map[string]interface{}{"variable-opt": "opt2"}
	tt := []struct {
		poolHosts           int
		dedicatedPool       bool
		machines            []*dockermachine.Machine
		expectedProvisioned int
		expectedDriverOpts  []map[string]interface{}
	}{
		{1, false, []*dockermachine.Machine{{}}, 0, []map[string]interface{}{}},
		{2, false, []*dockermachine.Machine{{}}, 1, []map[string]interface{}{opt1, {}}},
		{1, true, []*dockermachine.Machine{{}}, 1, []map[string]interface{}{opt1}},
		{2, true, []*dockermachine.Machine{{}, {}}, 2, []map[string]interface{}{opt1, opt2}},
		{3, true, []*dockermachine.Machine{{}}, 3, []map[string]interface{}{opt1, opt2, opt1}},
	}
	for ti, t := range tt {
		p := &FakeMachineProvisioner{}
		installer := &Installer{machineProvisioner: p}
		config := &InstallOpts{
			AppsHosts:          t.poolHosts,
			DedicatedAppsHosts: t.dedicatedPool,
			AppsDriversOpts: map[string][]interface{}{
				"variable-opt": {"opt1", "opt2"},
			},
		}
		machines, err := installer.ProvisionPool(config, t.machines)
		c.Assert(err, check.IsNil)
		c.Assert(p.hostsProvisioned, check.Equals, t.expectedProvisioned)
		for i := 0; i < t.expectedProvisioned; i++ {
			if !reflect.DeepEqual(machines[i].Base.CustomData, t.expectedDriverOpts[i]) {
				c.Errorf("Test case %d/%d failed. Expected %+v. Got %+v", ti, i, t.expectedDriverOpts[i], machines[i].Base.CustomData)
			}
		}
	}
}

func (s *S) TestCreateLoadBalancer(c *check.C) {
	fp := &FakeMachineProvisioner{}
	machine, err := fp.ProvisionMachine(nil)
	machine2, err := fp.ProvisionMachine(nil)
	machines := []*dockermachine.Machine{machine, machine2}
	c.Assert(err, check.IsNil)
	_, err = ProvisionLoadBalancer("fake", machines)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.Equals, errDriverNotSupportLB)
	addr, err := ProvisionLoadBalancer("amazonec2", machines)
	c.Assert(err, check.IsNil)
	c.Assert(addr, check.Equals, "lb.test.com")

}
