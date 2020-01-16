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

package config

import (
	"os"
	"strings"
	"testing"
)

const basicConfig = `
[Global]
server = 0.0.0.0
port = 443
user = user
password = password
insecure-flag = true
datacenters = us-west
ca-file = /some/path/to/a/ca.pem
`

const multiICSDCsUsingSecretConfig = `
[Global]
port = 443
insecure-flag = true

[ICSCenter "tenant1"]
server = "10.0.0.1"
datacenters = "vic0dc"
secret-name = "tenant1-secret"
secret-namespace = "kube-system"
# port, insecure-flag will be used from Global section.

[ICSCenter "tenant2"]
server = "10.0.0.2"
datacenters = "vic1dc"
secret-name = "tenant2-secret"
secret-namespace = "kube-system"
# port, insecure-flag will be used from Global section.

[ICSCenter "10.0.0.3"]
datacenters = "vicdc"
secret-name = "eu-secret"
secret-namespace = "kube-system"
# port, insecure-flag will be used from Global section.
`

func TestReadConfigGlobal(t *testing.T) {
	_, err := ReadConfig(nil)
	if err == nil {
		t.Errorf("Should fail when no config is provided: %s", err)
	}

	cfg, err := ReadConfig(strings.NewReader(basicConfig))
	if err != nil {
		t.Fatalf("Should succeed when a valid config is provided: %s", err)
	}

	if cfg.Global.ICenterIP != "0.0.0.0" {
		t.Errorf("incorrect icenter ip: %s", cfg.Global.ICenterIP)
	}

	if cfg.Global.Datacenters != "us-west" {
		t.Errorf("incorrect datacenter: %s", cfg.Global.Datacenters)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	ip := "127.0.0.1"
	os.Setenv("ICS_ICENTER", ip)
	defer os.Unsetenv("ICS_ICENTER")

	cfg, err := ReadConfig(strings.NewReader(basicConfig))
	if err != nil {
		t.Fatalf("Should succeed when a valid config is provided: %s", err)
	}

	if cfg.Global.ICenterIP != ip {
		t.Errorf("expected IP: %s, got: %s", ip, cfg.Global.ICenterIP)
	}
}

func TestBlankEnvFails(t *testing.T) {
	cfg := &Config{}

	err := cfg.FromEnv()
	if err == nil {
		t.Fatalf("Env only config should fail if env not set")
	}
}

func TestIPFamilies(t *testing.T) {
	input := "ipv6"
	ipFamilies, err := validateIPFamily(input)
	if err != nil {
		t.Errorf("Valid ipv6 but yielded err: %s", err)
	}
	size := len(ipFamilies)
	if size != 1 {
		t.Errorf("Invalid family list expected: 1, actual: %d", size)
	}

	input = "ipv4"
	ipFamilies, err = validateIPFamily(input)
	if err != nil {
		t.Errorf("Valid ipv4 but yielded err: %s", err)
	}
	size = len(ipFamilies)
	if size != 1 {
		t.Errorf("Invalid family list expected: 1, actual: %d", size)
	}

	input = "ipv4, "
	ipFamilies, err = validateIPFamily(input)
	if err != nil {
		t.Errorf("Valid ipv4, but yielded err: %s", err)
	}
	size = len(ipFamilies)
	if size != 1 {
		t.Errorf("Invalid family list expected: 1, actual: %d", size)
	}

	input = "ipv6,ipv4"
	ipFamilies, err = validateIPFamily(input)
	if err != nil {
		t.Errorf("Valid ipv6/ipv4 but yielded err: %s", err)
	}
	size = len(ipFamilies)
	if size != 2 {
		t.Errorf("Invalid family list expected: 2, actual: %d", size)
	}

	input = "ipv7"
	_, err = validateIPFamily(input)
	if err == nil {
		t.Errorf("Invalid ipv7 but successful")
	}

	input = "ipv4,ipv7"
	_, err = validateIPFamily(input)
	if err == nil {
		t.Errorf("Invalid ipv4,ipv7 but successful")
	}
}

func TestTenantRefs(t *testing.T) {
	cfg, err := ReadConfig(strings.NewReader(multiICSDCsUsingSecretConfig))
	if err != nil {
		t.Fatalf("Should succeed when a valid config is provided: %s", err)
	}

	if cfg.IsSecretInfoProvided() {
		t.Error("IsSecretInfoProvided should not be set.")
	}

	icsConfig1 := cfg.ICSCenter["tenant1"]
	if icsConfig1 == nil {
		t.Fatalf("Should return a valid icsConfig1")
	}
	if !icsConfig1.IsSecretInfoProvided() {
		t.Error("icsConfig1.IsSecretInfoProvided() should be set.")
	}
	if !strings.EqualFold(icsConfig1.ICenterIP, "10.0.0.1") {
		t.Errorf("icsConfig1 ICenterIP should be 10.0.0.1 but actual=%s", icsConfig1.ICenterIP)
	}
	if !strings.EqualFold(icsConfig1.TenantRef, "tenant1") {
		t.Errorf("icsConfig1 TenantRef should be tenant1 but actual=%s", icsConfig1.TenantRef)
	}
	if !strings.EqualFold(icsConfig1.SecretRef, "kube-system/tenant1-secret") {
		t.Errorf("icsConfig1 SecretRef should be kube-system/tenant1-secret but actual=%s", icsConfig1.SecretRef)
	}

	icsConfig2 := cfg.ICSCenter["tenant2"]
	if icsConfig2 == nil {
		t.Fatalf("Should return a valid icsConfig2")
	}
	if !icsConfig2.IsSecretInfoProvided() {
		t.Error("icsConfig2.IsSecretInfoProvided() should be set.")
	}
	if !strings.EqualFold(icsConfig2.ICenterIP, "10.0.0.2") {
		t.Errorf("icsConfig2 ICenterIP should be 10.0.0.2 but actual=%s", icsConfig2.ICenterIP)
	}
	if !strings.EqualFold(icsConfig2.TenantRef, "tenant2") {
		t.Errorf("icsConfig2 TenantRef should be tenant2 but actual=%s", icsConfig2.TenantRef)
	}
	if !strings.EqualFold(icsConfig2.SecretRef, "kube-system/tenant2-secret") {
		t.Errorf("icsConfig2 SecretRef should be kube-system/tenant2-secret but actual=%s", icsConfig2.SecretRef)
	}

	icsConfig3 := cfg.ICSCenter["10.0.0.3"]
	if icsConfig3 == nil {
		t.Fatalf("Should return a valid icsConfig3")
	}
	if !icsConfig3.IsSecretInfoProvided() {
		t.Error("icsConfig3.IsSecretInfoProvided() should be set.")
	}
	if !strings.EqualFold(icsConfig3.ICenterIP, "10.0.0.3") {
		t.Errorf("icsConfig3 ICenterIP should be 10.0.0.3 but actual=%s", icsConfig3.ICenterIP)
	}
	if !strings.EqualFold(icsConfig3.TenantRef, "10.0.0.3") {
		t.Errorf("icsConfig3 TenantRef should be eu-secret but actual=%s", icsConfig3.TenantRef)
	}
	if !strings.EqualFold(icsConfig3.SecretRef, "kube-system/eu-secret") {
		t.Errorf("icsConfig3 SecretRef should be kube-system/eu-secret but actual=%s", icsConfig3.SecretRef)
	}
}
