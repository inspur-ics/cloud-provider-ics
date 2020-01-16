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

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"k8s.io/klog"

	"gopkg.in/gcfg.v1"

)

func getEnvKeyValue(match string, partial bool) (string, string, error) {
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		if partial && strings.Contains(key, match) {
			return key, value, nil
		}

		if strings.Compare(key, match) == 0 {
			return key, value, nil
		}
	}

	matchType := "match"
	if partial {
		matchType = "partial match"
	}

	return "", "", fmt.Errorf("Failed to find %s with %s", matchType, match)
}

// FromEnv initializes the provided configuratoin object with values
// obtained from environment variables. If an environment variable is set
// for a property that's already initialized, the environment variable's value
// takes precedence.
func (cfg *Config) FromEnv() error {

	//Init
	if cfg.ICSCenter == nil {
		cfg.ICSCenter = make(map[string]*ICSCenterConfig)
	}

	//Globals
	if env := os.Getenv("ICS_ICENTER"); env != "" {
		cfg.Global.ICenterIP = env
	}
	if env := os.Getenv("ICS_ICENTER_PORT"); env != "" {
		cfg.Global.ICenterPort = env
	}
	if env := os.Getenv("ICS_USER"); env != "" {
		cfg.Global.User = env
	}
	if env := os.Getenv("ICS_PASSWORD"); env != "" {
		cfg.Global.Password = env
	}
	if env := os.Getenv("ICS_DATACENTER"); env != "" {
		cfg.Global.Datacenters = env
	}
	if env := os.Getenv("ICS_SECRET_NAME"); env != "" {
		cfg.Global.SecretName = env
	}
	if env := os.Getenv("ICS_SECRET_NAMESPACE"); env != "" {
		cfg.Global.SecretNamespace = env
	}

	if env := os.Getenv("ICS_INSECURE"); env != "" {
		InsecureFlag, err := strconv.ParseBool(env)
		if err != nil {
			klog.Errorf("Failed to parse ICS_INSECURE: %s", err)
		} else {
			cfg.Global.InsecureFlag = InsecureFlag
		}
	}

	if env := os.Getenv("ICS_API_DISABLE"); env != "" {
		APIDisable, err := strconv.ParseBool(env)
		if err != nil {
			klog.Errorf("Failed to parse ICS_API_DISABLE: %s", err)
		} else {
			cfg.Global.APIDisable = APIDisable
		}
	}

	if env := os.Getenv("ICS_API_BINDING"); env != "" {
		cfg.Global.APIBinding = env
	}

	if env := os.Getenv("ICS_SECRETS_DIRECTORY"); env != "" {
		cfg.Global.SecretsDirectory = env
	}
	if cfg.Global.SecretsDirectory == "" {
		cfg.Global.SecretsDirectory = DefaultSecretDirectory
	}
	if _, err := os.Stat(cfg.Global.SecretsDirectory); os.IsNotExist(err) {
		cfg.Global.SecretsDirectory = "" //Dir does not exist, set to empty string
	}
	if env := os.Getenv("ICS_LABEL_REGION"); env != "" {
		cfg.Labels.Region = env
	}
	if env := os.Getenv("ICS_LABEL_ZONE"); env != "" {
		cfg.Labels.Zone = env
	}

	if env := os.Getenv("ICS_IP_FAMILY"); env != "" {
		cfg.Global.IPFamily = env
	}
	if cfg.Global.IPFamily == "" {
		cfg.Global.IPFamily = DefaultIPFamily
	}

	//Build ICSCenter from ENVs
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")

		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		if strings.HasPrefix(key, "ICS_ICENTER_") && len(value) > 0 {
			id := strings.TrimPrefix(key, "ICS_ICENTER_")
			icenter := value

			_, username, errUsername := getEnvKeyValue("ICENTER_"+id+"_USERNAME", false)
			if errUsername != nil {
				username = cfg.Global.User
			}
			_, password, errPassword := getEnvKeyValue("ICENTER_"+id+"_PASSWORD", false)
			if errPassword != nil {
				password = cfg.Global.Password
			}
			_, server, errServer := getEnvKeyValue("ICENTER_"+id+"_SERVER", false)
			if errServer != nil {
				server = ""
			}
			_, port, errPort := getEnvKeyValue("ICENTER_"+id+"_PORT", false)
			if errPort != nil {
				port = cfg.Global.ICenterPort
			}
			insecureFlag := false
			_, insecureTmp, errInsecure := getEnvKeyValue("ICENTER_"+id+"_INSECURE", false)
			if errInsecure != nil {
				insecureFlagTmp, errTmp := strconv.ParseBool(insecureTmp)
				if errTmp == nil {
					insecureFlag = insecureFlagTmp
				}
			}
			_, datacenters, errDatacenters := getEnvKeyValue("ICENTER_"+id+"_DATACENTERS", false)
			if errDatacenters != nil {
				datacenters = cfg.Global.Datacenters
			}
			_, secretName, secretNameErr := getEnvKeyValue("ICENTER_"+id+"_SECRET_NAME", false)
			_, secretNamespace, secretNamespaceErr := getEnvKeyValue("ICENTER_"+id+"_SECRET_NAMESPACE", false)

			if secretNameErr != nil || secretNamespaceErr != nil {
				secretName = ""
				secretNamespace = ""
			}
			secretRef := DefaultCredentialManager
			if secretName != "" && secretNamespace != "" {
				secretRef = icenter
			}

			_, ipFamily, errIPFamily := getEnvKeyValue("ICENTER_"+id+"_IP_FAMILY", false)
			if errIPFamily != nil {
				ipFamily = cfg.Global.IPFamily
			}

			// If server is explicitly set, that means the icenter value above is the TenantRef
			icenterIP := icenter
			tenantRef := icenter
			if server != "" {
				icenterIP = server
				tenantRef = icenter
			}

			cfg.ICSCenter[tenantRef] = &ICSCenterConfig{
				User:              username,
				Password:          password,
				TenantRef:         tenantRef,
				ICenterIP:         icenterIP,
				ICenterPort:       port,
				InsecureFlag:      insecureFlag,
				Datacenters:       datacenters,
				SecretRef:         secretRef,
				SecretName:        secretName,
				SecretNamespace:   secretNamespace,
				IPFamily:          ipFamily,
			}
		}
	}

	if cfg.Global.ICenterIP != "" && cfg.ICSCenter[cfg.Global.ICenterIP] == nil {
		cfg.ICSCenter[cfg.Global.ICenterIP] = &ICSCenterConfig{
			User:              cfg.Global.User,
			Password:          cfg.Global.Password,
			TenantRef:         cfg.Global.ICenterIP,
			ICenterIP:         cfg.Global.ICenterIP,
			ICenterPort:       cfg.Global.ICenterPort,
			InsecureFlag:      cfg.Global.InsecureFlag,
			Datacenters:       cfg.Global.Datacenters,
			SecretRef:         DefaultCredentialManager,
			SecretName:        cfg.Global.SecretName,
			SecretNamespace:   cfg.Global.SecretNamespace,
			IPFamily:          cfg.Global.IPFamily,
		}
	}

	err := cfg.validateConfig()
	if err != nil {
		return err
	}

	return nil
}

// IsSecretInfoProvided returns true if k8s secret is set or using generic CO secret method.
// If both k8s secret and generic CO both are true, we don't know which to use, so return false.
func (cfg *Config) IsSecretInfoProvided() bool {
	return (cfg.Global.SecretName != "" && cfg.Global.SecretNamespace != "" && cfg.Global.SecretsDirectory == "") ||
		(cfg.Global.SecretName == "" && cfg.Global.SecretNamespace == "" && cfg.Global.SecretsDirectory != "")
}

func validateIPFamily(value string) ([]string, error) {
	if len(value) == 0 {
		return []string{DefaultIPFamily}, nil
	}

	ipFamilies := strings.Split(value, ",")
	for i, ipFamily := range ipFamilies {
		ipFamily = strings.TrimSpace(ipFamily)
		if len(ipFamily) == 0 {
			copy(ipFamilies[i:], ipFamilies[i+1:])      // Shift a[i+1:] left one index.
			ipFamilies[len(ipFamilies)-1] = ""          // Erase last element (write zero value).
			ipFamilies = ipFamilies[:len(ipFamilies)-1] // Truncate slice.
			continue
		}
		if !strings.EqualFold(ipFamily, IPv4Family) && !strings.EqualFold(ipFamily, IPv6Family) {
			return nil, ErrInvalidIPFamilyType
		}
	}

	return ipFamilies, nil
}

func (cfg *Config) validateConfig() error {
	//Fix default global values
	if cfg.Global.ICenterPort == "" {
		cfg.Global.ICenterPort = DefaultICenterPort
	}
	if cfg.Global.APIBinding == "" {
		cfg.Global.APIBinding = DefaultAPIBinding
	}
	if cfg.Global.IPFamily == "" {
		cfg.Global.IPFamily = DefaultIPFamily
	}

	ipFamilyPriority, err := validateIPFamily(cfg.Global.IPFamily)
	if err != nil {
		klog.Errorf("Invalid Global IPFamily: %s, err=%s", cfg.Global.IPFamily, err)
		return err
	}

	// Create a single instance of ICSInstance for the Global ICenterIP if the
	// ICSCenter does not already exist in the map
	if cfg.Global.ICenterIP != "" && cfg.ICSCenter[cfg.Global.ICenterIP] == nil {
		icsConfig := &ICSCenterConfig{
			User:              cfg.Global.User,
			Password:          cfg.Global.Password,
			TenantRef:         cfg.Global.ICenterIP,
			ICenterIP:         cfg.Global.ICenterIP,
			ICenterPort:       cfg.Global.ICenterPort,
			InsecureFlag:      cfg.Global.InsecureFlag,
			Datacenters:       cfg.Global.Datacenters,
			SecretRef:         DefaultCredentialManager,
			SecretName:        cfg.Global.SecretName,
			SecretNamespace:   cfg.Global.SecretNamespace,
			IPFamily:          cfg.Global.IPFamily,
			IPFamilyPriority:  ipFamilyPriority,
		}
		cfg.ICSCenter[cfg.Global.ICenterIP] = icsConfig
	}

	// Must have at least one iCenter defined
	if len(cfg.ICSCenter) == 0 {
		klog.Error(ErrMissingICenter)
		return ErrMissingICenter
	}

	// ics.conf is no longer supported in the old format.
	for icsServer, icsConfig := range cfg.ICSCenter {
		klog.V(4).Infof("Initializing ics server %s", icsServer)
		if icsServer == "" {
			klog.Error(ErrInvalidICenterIP)
			return ErrInvalidICenterIP
		}

		// If icsConfig.ICenterIP is explicitly set, that means the icsServer
		// above is the TenantRef
		if icsConfig.ICenterIP != "" {
			//icsConfig.ICenterIP is already set
			icsConfig.TenantRef = icsServer
		} else {
			icsConfig.ICenterIP = icsServer
			icsConfig.TenantRef = icsServer
		}

		if !cfg.IsSecretInfoProvided() && !icsConfig.IsSecretInfoProvided() {
			if icsConfig.User == "" {
				icsConfig.User = cfg.Global.User
				if icsConfig.User == "" {
					klog.Errorf("icsConfig.User is empty for ics %s!", icsServer)
					return ErrUsernameMissing
				}
			}
			if icsConfig.Password == "" {
				icsConfig.Password = cfg.Global.Password
				if icsConfig.Password == "" {
					klog.Errorf("icsConfig.Password is empty for ics %s!", icsServer)
					return ErrPasswordMissing
				}
			}
		} else if cfg.IsSecretInfoProvided() && !icsConfig.IsSecretInfoProvided() {
			icsConfig.SecretRef = DefaultCredentialManager
		} else if icsConfig.IsSecretInfoProvided() {
			icsConfig.SecretRef = icsConfig.SecretNamespace + "/" + icsConfig.SecretName
		}

		if icsConfig.ICenterPort == "" {
			icsConfig.ICenterPort = cfg.Global.ICenterPort
		}

		if icsConfig.Datacenters == "" {
			if cfg.Global.Datacenters != "" {
				icsConfig.Datacenters = cfg.Global.Datacenters
			}
		}

		if icsConfig.IPFamily == "" {
			icsConfig.IPFamily = cfg.Global.IPFamily
		}

		ipFamilyPriority, err := validateIPFamily(icsConfig.IPFamily)
		if err != nil {
			klog.Errorf("Invalid icsConfig IPFamily: %s, err=%s", icsConfig.IPFamily, err)
			return err
		}
		icsConfig.IPFamilyPriority = ipFamilyPriority

		insecure := icsConfig.InsecureFlag
		if !insecure {
			icsConfig.InsecureFlag = cfg.Global.InsecureFlag
		}
	}

	return nil
}

// ReadConfig parses inCloud Sphere cloud config file and stores it into ICSConfig.
// Environment variables are also checked
func ReadConfig(config io.Reader) (*Config, error) {
	if config == nil {
		return nil, fmt.Errorf("no inCloud Sphere cloud provider config file given")
	}

	cfg := &Config{}
	if err := gcfg.FatalOnly(gcfg.ReadInto(cfg, config)); err != nil {
		return nil, err
	}

	// Env Vars should override config file entries if present
	if err := cfg.FromEnv(); err != nil {
		return nil, err
	}

	return cfg, nil
}
