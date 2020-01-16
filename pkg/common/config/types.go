/*
Copyright 2018 The Kubernetes Authors.

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

// Config is used to read and store information from the cloud configuration file
type Config struct {
	Global struct {
		// iCenter username.
		User string `gcfg:"user"`
		// iCenter password in clear text.
		Password string `gcfg:"password"`
		// Deprecated. Use ICSCenter to specify multiple iCenter Servers.
		// iCenter IP.
		ICenterIP string `gcfg:"server"`
		// iCenter port.
		ICenterPort string `gcfg:"port"`
		// True if iCenter uses self-signed cert.
		InsecureFlag bool `gcfg:"insecure-flag"`
		// Datacenter in which VMs are located.
		Datacenters string `gcfg:"datacenters"`
		// Name of the secret were iCenter credentials are present.
		SecretName string `gcfg:"secret-name"`
		// Secret Namespace where secret will be present that has iCenter credentials.
		SecretNamespace string `gcfg:"secret-namespace"`
		// Secret directory in the event that:
		// 1) we don't want to use the k8s API to listen for changes to secrets
		// 2) we are not in a k8s env, namely DC/OS, since CSI is CO agnostic
		// Default: /etc/cloud/credentials
		SecretsDirectory string `gcfg:"secrets-directory"`
		// Disable the inCloud Sphere CCM API
		// Default: true
		APIDisable bool `gcfg:"api-disable"`
		// Configurable inCloud Sphere CCM API port
		// Default: 43001
		APIBinding string `gcfg:"api-binding"`
		// IP Family enables the ability to support IPv4 or IPv6
		// Supported values are:
		// ipv4 - IPv4 addresses only (Default)
		// ipv6 - IPv6 addresses only
		IPFamily string `gcfg:"ip-family"`
	}

	// ICS Center configurations
	ICSCenter map[string]*ICSCenterConfig

	// Tag categories and tags which correspond to "built-in node labels: zones and region"
	Labels struct {
		Zone   string `gcfg:"zone"`
		Region string `gcfg:"region"`
	}
}

// ICSCenterConfig contains information used to access a remote iCenter
// endpoint.
type ICSCenterConfig struct {
	// iCenter username.
	User string `gcfg:"user"`
	// iCenter password in clear text.
	Password string `gcfg:"password"`
	// TenantRef (intentionally not exposed via the config) is a unique tenant ref to
	// be used in place of the icsServer as the primary connection key. If one label is set,
	// all virtual center configs must have a unique label.
	TenantRef string
	// vCenterIP - If this field in the config is set, it is assumed then that value in [ICSCenter "<value>"]
	// is now the TenantRef above and this field is the actual ICenterIP. Otherwise for backward
	// compatibility, the value by default is the IP or FQDN of the iCenter Server.
	ICenterIP string `gcfg:"server"`
	// iCenter port.
	ICenterPort string `gcfg:"port"`
	// True if iCenter uses self-signed cert.
	InsecureFlag bool `gcfg:"insecure-flag"`
	// Datacenter in which VMs are located.
	Datacenters string `gcfg:"datacenters"`
	// SecretRef (intentionally not exposed via the config) is a key to identify which
	// InformerManager holds the secret
	SecretRef string
	// Name of the secret where iCenter credentials are present.
	SecretName string `gcfg:"secret-name"`
	// Namespace where the secret will be present containing iCenter credentials.
	SecretNamespace string `gcfg:"secret-namespace"`
	// IP Family enables the ability to support IPv4 or IPv6
	// Supported values are:
	// ipv4 - IPv4 addresses only (Default)
	// ipv6 - IPv6 addresses only
	IPFamily string `gcfg:"ip-family"`
	// IPFamilyPriority (intentionally not exposed via the config) the list/priority of IP versions
	IPFamilyPriority []string
}
