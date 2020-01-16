/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connectionmanager

import (
	"context"
	"crypto/tls"
	"strings"
	"testing"

	icfg "github.com/inspur-ics/cloud-provider-ics/pkg/common/config"
)

// configFromSim starts a vcsim instance and returns config for use against the vcsim instance.
// The vcsim instance is configured with an empty tls.Config.
func configFromSim(multiDc bool) (*icfg.Config, func()) {
	return configFromSimWithTLS(new(tls.Config), true, multiDc)
}

// configFromSimWithTLS starts a vcsim instance and returns config for use against the vcsim instance.
// The vcsim instance is configured with a tls.Config. The returned client
// config can be configured to allow/decline insecure connections.
func configFromSimWithTLS(tlsConfig *tls.Config, insecureAllowed bool, multiDc bool) (*icfg.Config, func()) {
	cfg := &icfg.Config{}
	var HostName = "127.0.0.1"

	cfg.Global.InsecureFlag = insecureAllowed

	cfg.Global.ICenterIP = HostName
	cfg.Global.ICenterPort = "443"
	cfg.Global.User = "admin"
	cfg.Global.Password = "123456"

	if multiDc {
		cfg.Global.Datacenters = "DC0,DC1"
	} else {
		cfg.Global.Datacenters = "DC0"
	}
	cfg.ICSCenter = make(map[string]*icfg.ICSCenterConfig)
	cfg.ICSCenter[HostName] = &icfg.ICSCenterConfig{
		User:         cfg.Global.User,
		Password:     cfg.Global.Password,
		TenantRef:    cfg.Global.ICenterIP,
		ICenterIP:    cfg.Global.ICenterIP,
		ICenterPort:  cfg.Global.ICenterPort,
		InsecureFlag: cfg.Global.InsecureFlag,
		Datacenters:  cfg.Global.Datacenters,
	}

	// Configure region and zone categories
	cfg.Labels.Region = "k8s-region"
	cfg.Labels.Zone = "k8s-zone"

	return cfg, func() {}
}

func configFromEnvOrSim(multiDc bool) (*icfg.Config, func()) {
	cfg := &icfg.Config{}
	if err := cfg.FromEnv(); err != nil {
		return configFromSim(multiDc)
	}
	return cfg, func() {}
}

func TestListAllVcPairs(t *testing.T) {
	config, cleanup := configFromEnvOrSim(true)
	defer cleanup()

	connMgr := NewConnectionManager(config, nil, nil)
	defer connMgr.Logout()

	// context
	ctx := context.Background()

	items, err := connMgr.ListAllICSandDCPairs(ctx)
	if err != nil {
		t.Fatalf("ListAllICSandDCPairs err=%v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ListAllICSandDCPairs items should be 2 but count=%d", len(items))
	}

	// item 0
	if !strings.EqualFold(items[0].IcsServer, config.Global.ICenterIP) {
		t.Errorf("item[0].IcsServer mismatch %s!=%s", items[0].IcsServer, config.Global.ICenterIP)
	}
	if !strings.EqualFold(items[0].DataCenter.Name, "DC0") && !strings.EqualFold(items[0].DataCenter.Name, "DC1") {
		t.Errorf("item[0].Datacenter.Name name=%s should either be DC0 or DC1", items[0].DataCenter.Name)
	}

	// item 1
	if !strings.EqualFold(items[1].IcsServer, config.Global.ICenterIP) {
		t.Errorf("item[1].IcsServer mismatch %s!=%s", items[1].IcsServer, config.Global.ICenterIP)
	}
	if !strings.EqualFold(items[1].DataCenter.Name, "DC0") && !strings.EqualFold(items[1].DataCenter.Name, "DC1") {
		t.Errorf("item[1].Datacenter.Name name=%s should either be DC0 or DC1", items[1].DataCenter.Name)
	}
}
