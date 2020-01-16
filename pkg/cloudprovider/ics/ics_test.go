/*
Copyright 2016 The Kubernetes Authors.

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

package ics

import (
	"context"
	"crypto/tls"
	"strings"
	"testing"

	icfg "github.com/inspur-ics/cloud-provider-ics/pkg/common/config"
	cm "github.com/inspur-ics/cloud-provider-ics/pkg/common/connectionmanager"
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

// configFromEnvOrSim returns config from configFromEnv if set, otherwise returns configFromSim.
func configFromEnvOrSim(multiDc bool) (*icfg.Config, func()) {
	cfg := &icfg.Config{}
	if err := cfg.FromEnv(); err != nil {
		return configFromSim(multiDc)
	}
	return cfg, func() {}
}

func TestNewICS(t *testing.T) {
	cfg := &CPIConfig{}
	if err := cfg.FromCPIEnv(); err != nil {
		t.Skipf("No config found in environment")
	}

	_, err := newICS(cfg)
	if err != nil {
		t.Fatalf("Failed to construct/authenticate inCloud Sphere: %s", err)
	}
}

func TestICSLogin(t *testing.T) {
	initCfg, cleanup := configFromEnvOrSim(false)
	defer cleanup()
	cfg := &CPIConfig{}
	cfg.Config = *initCfg

	// Create inCloud Sphere configuration object
	ics, err := newICS(cfg)
	if err != nil {
		t.Fatalf("Failed to construct/authenticate inCloud Sphere: %s", err)
	}
	ics.connectionManager = cm.NewConnectionManager(&cfg.Config, nil, nil)
	defer ics.connectionManager.Logout()

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create inCloud Sphere client
	icsInstance, ok := ics.connectionManager.ICSInstanceMap[cfg.Global.ICenterIP]
	if !ok {
		t.Fatalf("Couldn't get inCloud Sphere instance: %s", cfg.Global.ICenterIP)
	}
	err = icsInstance.Conn.Connect(ctx)
	if err != nil {
		t.Errorf("Failed to connect to inCloud Sphere: %s", err)
	}
	icsInstance.Conn.Logout(ctx)
}

func TestSecretICSConfig(t *testing.T) {
	var ics *ICS
	var (
		username = "user"
		password = "password"
	)
	var testcases = []struct {
		testName                 string
		conf                     string
		expectedIsSecretProvided bool
		expectedUsername         string
		expectedPassword         string
		expectedError            error
		expectedThumbprints      map[string]string
	}{
		{
			testName: "Username and password with old configuration",
			conf: `[Global]
			server = 0.0.0.0
			user = user
			password = password
			datacenters = us-west
			`,
			expectedUsername: username,
			expectedPassword: password,
			expectedError:    nil,
		},
		{
			testName: "SecretName and SecretNamespace in old configuration",
			conf: `[Global]
			server = 0.0.0.0
			datacenters = us-west
			secret-name = "vccreds"
			secret-namespace = "kube-system"
			`,
			expectedIsSecretProvided: true,
			expectedError:            nil,
		},
		{
			testName: "SecretName and SecretNamespace with Username and Password in old configuration",
			conf: `[Global]
			server = 0.0.0.0
			user = user
			password = password
			datacenters = us-west
			secret-name = "vccreds"
			secret-namespace = "kube-system"
			`,
			expectedIsSecretProvided: true,
			expectedError:            nil,
		},
		{
			testName: "SecretName and SecretNamespace with Username missing in old configuration",
			conf: `[Global]
			server = 0.0.0.0
			password = password
			datacenters = us-west
			secret-name = "vccreds"
			secret-namespace = "kube-system"
			`,
			expectedIsSecretProvided: true,
			expectedError:            nil,
		},
		{
			testName: "SecretNamespace missing with Username and Password in old configuration",
			conf: `[Global]
			server = 0.0.0.0
			user = user
			password = password
			datacenters = us-west
			secret-name = "vccreds"
			`,
			expectedUsername: username,
			expectedPassword: password,
			expectedError:    nil,
		},
		{
			testName: "SecretNamespace and Username missing in old configuration",
			conf: `[Global]
			server = 0.0.0.0
			password = password
			datacenters = us-west
			secret-name = "vccreds"
			`,
			expectedPassword: password,
			expectedError:    icfg.ErrUsernameMissing,
		},
		{
			testName: "SecretNamespace and Password missing in old configuration",
			conf: `[Global]
			server = 0.0.0.0
			user = user
			datacenters = us-west
			secret-name = "vccreds"
			`,
			expectedUsername: username,
			expectedError:    icfg.ErrPasswordMissing,
		},
		{
			testName: "SecretNamespace, Username and Password missing in old configuration",
			conf: `[Global]
			server = 0.0.0.0
			datacenters = us-west
			secret-name = "vccreds"
			`,
			expectedError: icfg.ErrUsernameMissing,
		},
		{
			testName: "Username and password with new configuration but username and password in global section",
			conf: `[Global]
			user = user
			password = password
			datacenters = us-west
			[ICSCenter "0.0.0.0"]
			`,
			expectedUsername: username,
			expectedPassword: password,
			expectedError:    nil,
		},
		{
			testName: "Username and password with new configuration, username and password in virtualcenter section",
			conf: `[Global]
			server = 0.0.0.0
			port = 443
			insecure-flag = true
			datacenters = us-west
			[ICSCenter "0.0.0.0"]
			user = user
			password = password
			`,
			expectedUsername: username,
			expectedPassword: password,
			expectedError:    nil,
		},
		{
			testName: "SecretName and SecretNamespace with new configuration",
			conf: `[Global]
			server = 0.0.0.0
			secret-name = "vccreds"
			secret-namespace = "kube-system"
			datacenters = us-west
			[ICSCenter "0.0.0.0"]
			`,
			expectedIsSecretProvided: true,
			expectedError:            nil,
		},
		{
			testName: "SecretName and SecretNamespace with Username missing in new configuration",
			conf: `[Global]
			server = 0.0.0.0
			port = 443
			insecure-flag = true
			datacenters = us-west
			secret-name = "vccreds"
			secret-namespace = "kube-system"
			[ICSCenter "0.0.0.0"]
			password = password
			`,
			expectedIsSecretProvided: true,
			expectedError:            nil,
		},
		{
			testName: "virtual centers with a thumbprint",
			conf: `[Global]
			server = global
			user = user
			password = password
			datacenters = us-west
			thumbprint = "thumbprint:global"
			`,
			expectedUsername: username,
			expectedPassword: password,
			expectedError:    nil,
			expectedThumbprints: map[string]string{
				"global": "thumbprint:global",
			},
		},
		{
			testName: "Multiple virtual centers with different thumbprints",
			conf: `[Global]
			user = user
			password = password
			datacenters = us-west
			[ICSCenter "0.0.0.0"]
			thumbprint = thumbprint:0
			[ICSCenter "no_thumbprint"]
			[ICSCenter "1.1.1.1"]
			thumbprint = thumbprint:1
			`,
			expectedUsername: username,
			expectedPassword: password,
			expectedError:    nil,
			expectedThumbprints: map[string]string{
				"0.0.0.0": "thumbprint:0",
				"1.1.1.1": "thumbprint:1",
			},
		},
		{
			testName: "Multiple virtual centers use the global CA cert",
			conf: `[Global]
			user = user
			password = password
			datacenters = us-west
			ca-file = /some/path/to/my/trusted/ca.pem
			[ICSCenter "0.0.0.0"]
			user = user
			password = password
			[ICSCenter "1.1.1.1"]
			user = user
			password = password
			`,
			expectedUsername: username,
			expectedPassword: password,
			expectedError:    nil,
		},
	}

	for _, testcase := range testcases {
		t.Logf("Executing Testcase: %s", testcase.testName)
		cfg, err := ReadCPIConfig(strings.NewReader(testcase.conf))
		if err != nil {
			if testcase.expectedError != nil {
				if err != testcase.expectedError {
					t.Fatalf("readConfig: expected err: %s, received err: %s", testcase.expectedError, err)
				} else {
					continue
				}
			} else {
				t.Fatalf("readConfig: unexpected error returned: %v", err)
			}
		}
		ics, err = buildICSFromConfig(cfg)
		if err != nil { // testcase.expectedError {
			t.Fatalf("buildICSFromConfig: Should succeed when a valid config is provided: %v", err)
		}
		ics.connectionManager = cm.NewConnectionManager(&cfg.Config, nil, nil)

		if testcase.expectedIsSecretProvided && (ics.cfg.Global.SecretNamespace == "" || ics.cfg.Global.SecretName == "") {
			t.Fatalf("SecretName and SecretNamespace was expected in config %s. error: %s",
				testcase.conf, err)
		}
		if !testcase.expectedIsSecretProvided {
			for _, vsInstance := range ics.connectionManager.ICSInstanceMap {
				if vsInstance.Conn.Username != testcase.expectedUsername {
					t.Fatalf("Expected username %s doesn't match actual username %s in config %s. error: %s",
						testcase.expectedUsername, vsInstance.Conn.Username, testcase.conf, err)
				}
				if vsInstance.Conn.Password != testcase.expectedPassword {
					t.Fatalf("Expected password %s doesn't match actual password %s in config %s. error: %s",
						testcase.expectedPassword, vsInstance.Conn.Password, testcase.conf, err)
				}
			}
		}
	}
}
