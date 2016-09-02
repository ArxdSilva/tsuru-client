// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/cert"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/drivers/rpc"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec"
)

var (
	dockerHTTPSPort            = 2376
	storeBasePath              = cmd.JoinWithUserDir(".tsuru", "installs")
	defaultDockerMachineConfig = &DockerMachineConfig{
		DriverName: "virtualbox",
		Name:       "tsuru",
		DriverOpts: make(map[string]interface{}),
	}
)

type Machine struct {
	*host.Host
	IP        string
	Address   string
	CAPath    string
	OpenPorts []string
}

func (m *Machine) dockerClient() (*docker.Client, error) {
	return docker.NewTLSClient(
		m.Address,
		filepath.Join(m.CAPath, "cert.pem"),
		filepath.Join(m.CAPath, "key.pem"),
		filepath.Join(m.CAPath, "ca.pem"),
	)
}

func (m *Machine) GetIP() string {
	return m.IP
}

func (m *Machine) GetSSHUsername() string {
	return m.Driver.GetSSHUsername()
}

func (m *Machine) GetSSHKeyPath() string {
	return m.Driver.GetSSHKeyPath()
}

type DockerMachine struct {
	io.Closer
	driverOpts    map[string]interface{}
	driverName    string
	storePath     string
	certsPath     string
	Name          string
	client        libmachine.API
	machinesCount uint64
}

type DockerMachineConfig struct {
	DriverName string
	DriverOpts map[string]interface{}
	CAPath     string
	Name       string
}

func NewDockerMachine(config *DockerMachineConfig) (*DockerMachine, error) {
	storePath := filepath.Join(storeBasePath, config.Name)
	certsPath := filepath.Join(storePath, "certs")
	err := os.MkdirAll(certsPath, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create certs dir: %s", err)
	}
	if config.CAPath != "" {
		fmt.Printf("Copying CA file from %s to %s", filepath.Join(config.CAPath, "ca.pem"), filepath.Join(certsPath, "ca.pem"))
		err = copy(filepath.Join(config.CAPath, "ca.pem"), filepath.Join(certsPath, "ca.pem"))
		if err != nil {
			return nil, fmt.Errorf("failed to copy ca file: %s", err)
		}
		fmt.Printf("Copying CA key from %s to %s", filepath.Join(config.CAPath, "ca-key.pem"), filepath.Join(certsPath, "ca-key.pem"))
		err = copy(filepath.Join(config.CAPath, "ca-key.pem"), filepath.Join(certsPath, "ca-key.pem"))
		if err != nil {
			return nil, fmt.Errorf("failed to copy ca key file: %s", err)
		}
	}
	return &DockerMachine{
		driverOpts: config.DriverOpts,
		driverName: config.DriverName,
		storePath:  storePath,
		certsPath:  certsPath,
		Name:       config.Name,
		client:     libmachine.NewClient(storePath, certsPath),
	}, nil
}

func copy(src, dst string) error {
	fileSrc, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dst, fileSrc, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (d *DockerMachine) CreateMachine(openPorts []string) (*Machine, error) {
	rawDriver, err := json.Marshal(&drivers.BaseDriver{
		MachineName: d.generateMachineName(),
		StorePath:   d.storePath,
	})
	if err != nil {
		return nil, fmt.Errorf("Error creating docker-machine driver: %s", err)
	}
	host, err := d.client.NewHost(d.driverName, rawDriver)
	if err != nil {
		return nil, err
	}
	configureDriver(host.Driver, d.driverOpts, openPorts)
	err = d.client.Create(host)
	if err != nil {
		fmt.Printf("Ignoring error on machine creation: %s", err)
	}
	ip, err := host.Driver.GetIP()
	if err != nil {
		return nil, err
	}
	m := &Machine{
		IP:        ip,
		CAPath:    d.certsPath,
		Host:      host,
		Address:   fmt.Sprintf("https://%s:%d", ip, dockerHTTPSPort),
		OpenPorts: openPorts,
	}
	return m, nil
}

func (d *DockerMachine) generateMachineName() string {
	atomic.AddUint64(&d.machinesCount, 1)
	return fmt.Sprintf("%s-%d", d.Name, atomic.LoadUint64(&d.machinesCount))
}

type sshTarget interface {
	RunSSHCommand(string) (string, error)
	GetIP() string
	GetSSHUsername() string
	GetSSHKeyPath() string
}

func (d *DockerMachine) uploadRegistryCertificate(host sshTarget) error {
	if _, err := os.Stat(filepath.Join(d.certsPath, "registry-cert.pem")); os.IsNotExist(err) {
		errCreate := d.createRegistryCertificate(host.GetIP())
		if errCreate != nil {
			return errCreate
		}
	}
	fmt.Printf("Uploading registry certificate...\n")
	args := []string{
		"-o StrictHostKeyChecking=no",
		"-i",
		host.GetSSHKeyPath(),
		"-r",
		fmt.Sprintf("%s/", d.certsPath),
		fmt.Sprintf("%s@%s:/home/%s/", host.GetSSHUsername(), host.GetIP(), host.GetSSHUsername()),
	}
	stdout := bytes.NewBufferString("")
	opts := exec.ExecuteOptions{
		Cmd:    "scp",
		Args:   args,
		Stdout: stdout,
	}
	err := client.Executor().Execute(opts)
	if err != nil {
		return fmt.Errorf("Command: %s. Error:%s", stdout.String(), err.Error())
	}
	certsBasePath := fmt.Sprintf("/home/%s/certs/%s:5000", host.GetSSHUsername(), host.GetIP())
	_, err = host.RunSSHCommand(fmt.Sprintf("mkdir -p %s", certsBasePath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("cp /home/%s/certs/*.pem %s/", host.GetSSHUsername(), certsBasePath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("sudo mkdir /etc/docker/certs.d && sudo cp -r /home/%s/certs/* /etc/docker/certs.d/", host.GetSSHUsername()))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("cat %s/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt", certsBasePath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand("mkdir -p /var/lib/registry/")
	return err
}

func (d *DockerMachine) createRegistryCertificate(hosts ...string) error {
	fmt.Printf("Creating registry certificate...\n")
	caOrg := mcnutils.GetUsername()
	org := caOrg + ".<bootstrap>"
	generator := &cert.X509CertGenerator{}
	certOpts := &cert.Options{
		Hosts:       hosts,
		CertFile:    filepath.Join(d.certsPath, "registry-cert.pem"),
		KeyFile:     filepath.Join(d.certsPath, "registry-key.pem"),
		CAFile:      filepath.Join(d.certsPath, "ca.pem"),
		CAKeyFile:   filepath.Join(d.certsPath, "ca-key.pem"),
		Org:         org,
		Bits:        2048,
		SwarmMaster: false,
	}
	return generator.GenerateCert(certOpts)
}

func configureDriver(driver drivers.Driver, driverOpts map[string]interface{}, openPorts []string) error {
	openPortFlag := driver.DriverName() + "-open-port"
	opts := &rpcdriver.RPCFlags{Values: driverOpts}
	for _, c := range driver.GetCreateFlags() {
		_, ok := opts.Values[c.String()]
		if !ok {
			opts.Values[c.String()] = c.Default()
			if c.Default() == nil {
				opts.Values[c.String()] = false
			}
			if c.String() == openPortFlag {
				opts.Values[c.String()] = openPorts
			}
		}
	}

	if err := driver.SetConfigFromFlags(opts); err != nil {
		return fmt.Errorf("Error setting driver configurations: %s", err)
	}
	return nil
}

func (d *DockerMachine) DeleteAll() error {
	hosts, err := d.client.List()
	if err != nil {
		return err
	}
	for _, h := range hosts {
		errDel := d.DeleteMachine(h)
		if errDel != nil {
			return errDel
		}
	}
	return nil
}

func (d *DockerMachine) DeleteMachine(name string) error {
	h, err := d.client.Load(name)
	if err != nil {
		return err
	}
	err = h.Driver.Remove()
	if err != nil {
		return err
	}
	return d.client.Remove(name)
}

func (d *DockerMachine) Close() error {
	return d.client.Close()
}
