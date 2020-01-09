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

package connectionmanager

import (
	"sync"

	clientset "k8s.io/client-go/kubernetes"
	icscfg "github.com/inspur-ics/cloud-provider-ics/pkg/common/config"
	cm "github.com/inspur-ics/cloud-provider-ics/pkg/common/credentialmanager"
	k8s "github.com/inspur-ics/cloud-provider-ics/pkg/common/kubernetes"
	icslib "github.com/inspur-ics/cloud-provider-ics/pkg/common/icslib"
)

// ConnectionManager encapsulates iCenter connections
type ConnectionManager struct {
	sync.Mutex

	// The k8s client init from the cloud provider service account
	client clientset.Interface

	// Maps the VC server to ICSInstance
	IcsInstanceMap map[string]*ICSInstance
	// CredentialManager per VC
	// The global CredentialManager will have an entry in this map with the key of "Global"
	credentialManagers map[string]*cm.CredentialManager
	// InformerManagers per VC
	// The global InformerManager will have an entry in this map with the key of "Global"
	informerManagers map[string]*k8s.InformerManager
}

// ICSInstance represents a ics instance where one or more kubernetes nodes are running.
type ICSInstance struct {
	Conn *icslib.ICSConnection
	Cfg  *icscfg.VirtualCenterConfig
}

// VMDiscoveryInfo contains VM info about a discovered VM
type VMDiscoveryInfo struct {
	TenantRef  string
	DataCenter *icslib.Datacenter
	VM         *icslib.VirtualMachine
	VcServer   string
	UUID       string
	NodeName   string
}

/*
// FcdDiscoveryInfo contains FCD info about a discovered FCD
type FcdDiscoveryInfo struct {
	TenantRef  string
	DataCenter *icslib.Datacenter
	FCDInfo    *icslib.FirstClassDiskInfo
	VcServer   string
}
*/

// ListDiscoveryInfo represents a VC/DC pair
type ListDiscoveryInfo struct {
	TenantRef  string
	VcServer   string
	DataCenter *icslib.Datacenter
}

// ZoneDiscoveryInfo contains VC+DC info based on a given zone
type ZoneDiscoveryInfo struct {
	TenantRef  string
	DataCenter *icslib.Datacenter
	VcServer   string
}
