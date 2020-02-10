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

package ics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	pb "github.com/inspur-ics/cloud-provider-ics/pkg/cloudprovider/ics/proto"
	icfg "github.com/inspur-ics/cloud-provider-ics/pkg/common/config"
	cm "github.com/inspur-ics/cloud-provider-ics/pkg/common/connectionmanager"
	v1 "k8s.io/api/core/v1"
	v1helper "k8s.io/cloud-provider/node/helpers"
	"k8s.io/klog"
)

// Errors
var (
	// ErrICenterNotFound is returned when the configured iCenter cannot be
	// found.
	ErrICenterNotFound = errors.New("iCenter not found")

	// ErrDatacenterNotFound is returned when the configured datacenter cannot
	// be found.
	ErrDatacenterNotFound = errors.New("Datacenter not found")

	// ErrVMNotFound is returned when the specified VM cannot be found.
	ErrVMNotFound = errors.New("VM not found")
)

func newNodeManager(cpiCfg *CPIConfig, cm *cm.ConnectionManager) *NodeManager {
	return &NodeManager{
		nodeNameMap:       make(map[string]*NodeInfo),
		nodeUUIDMap:       make(map[string]*NodeInfo),
		nodeRegUUIDMap:    make(map[string]*v1.Node),
		icsList:            make(map[string]*ICenterInfo),
		connectionManager: cm,
		cpiCfg:            cpiCfg,
	}
}

// RegisterNode is the handler for when a node is added to a K8s cluster.
func (nm *NodeManager) RegisterNode(node *v1.Node) {
	klog.V(4).Info("RegisterNode ENTER: ", node.Name)
	//uuid := ConvertK8sUUIDtoNormal(node.Status.NodeInfo.SystemUUID)
	nm.DiscoverNode(node.Status.NodeInfo.SystemUUID, cm.FindVMByUUID)
	nm.addNode(node.Status.NodeInfo.SystemUUID, node)
	klog.V(4).Info("RegisterNode LEAVE: ", node.Name)
}

// UnregisterNode is the handler for when a node is removed from a K8s cluster.
func (nm *NodeManager) UnregisterNode(node *v1.Node) {
	klog.V(4).Info("UnregisterNode ENTER: ", node.Name)
	//uuid := ConvertK8sUUIDtoNormal(node.Status.NodeInfo.SystemUUID)
	nm.removeNode(node.Status.NodeInfo.SystemUUID, node)
	klog.V(4).Info("UnregisterNode LEAVE: ", node.Name)
}

func (nm *NodeManager) addNodeInfo(node *NodeInfo) {
	nm.nodeInfoLock.Lock()
	klog.V(4).Info("addNodeInfo NodeName: ", node.NodeName, ", UUID: ", node.UUID)
	nm.nodeNameMap[node.NodeName] = node
	nm.nodeUUIDMap[node.UUID] = node
	nm.AddNodeInfoToICSList(node.icsServer, node.dataCenter.Name, node)
	nm.nodeInfoLock.Unlock()
}

func (nm *NodeManager) addNode(uuid string, node *v1.Node) {
	nm.nodeRegInfoLock.Lock()
	klog.V(4).Info("addNode NodeName: ", node.GetName(), ", UID: ", uuid)
	nm.nodeRegUUIDMap[uuid] = node
	nm.nodeRegInfoLock.Unlock()
}

func (nm *NodeManager) removeNode(uuid string, node *v1.Node) {
	nm.nodeRegInfoLock.Lock()
	klog.V(4).Info("removeNode NodeName: ", node.GetName(), ", UID: ", uuid)
	delete(nm.nodeRegUUIDMap, uuid)
	nm.nodeRegInfoLock.Unlock()
}

func (nm *NodeManager) shakeOutNodeIDLookup(ctx context.Context, nodeID string, searchBy cm.FindVM) (*cm.VMDiscoveryInfo, error) {
	// Search by NodeName
	if searchBy == cm.FindVMByName {
		vmDI, err := nm.connectionManager.WhichICSandDCByNodeID(ctx, nodeID, cm.FindVM(searchBy))
		if err == nil {
			klog.Info("Discovered VM using FQDN or short-hand name")
			return vmDI, err
		}

		vmDI, err = nm.connectionManager.WhichICSandDCByNodeID(ctx, nodeID, cm.FindVMByIP)
		if err == nil {
			klog.Info("Discovered VM using IP address")
			return vmDI, err
		}

		klog.Errorf("WhichICSandDCByNodeID failed using VM name. Err: %v", err)
		return nil, err
	}

	// Search by UUID
	vmDI, err := nm.connectionManager.WhichICSandDCByNodeID(ctx, nodeID, cm.FindVM(searchBy))
	if err == nil {
		klog.Info("Discovered VM using normal UUID format")
		return vmDI, err
	}

	// Need to lookup the original format of the UUID because photon 2.0 formats the UUID
	// different from Photon 3, RHEL, CentOS, Ubuntu, and etc
	klog.Errorf("WhichICSandDCByNodeID failed using normally formatted UUID. Err: %v", err)
	//reverseUUID := ConvertK8sUUIDtoNormal(nodeID)
	vmDI, err = nm.connectionManager.WhichICSandDCByNodeID(ctx, nodeID, cm.FindVM(searchBy))
	if err == nil {
		klog.Info("Discovered VM using reverse UUID format")
		return vmDI, err
	}

	klog.Errorf("WhichICSandDCByNodeID failed using UUID. Err: %v", err)
	return nil, err
}

func returnIPsFromSpecificFamily(family string, ips []string) []string {
	var matching []string

	for _, ip := range ips {
		if err := ErrOnLocalOnlyIPAddr(ip); err != nil {
			klog.V(4).Infof("IP is local only or there was an error. ip=%q err=%v", ip, err)
			continue
		}

		if strings.EqualFold(family, icfg.IPv6Family) && net.ParseIP(ip).To4() == nil {
			matching = append(matching, ip)
		} else if strings.EqualFold(family, icfg.IPv4Family) && net.ParseIP(ip).To4() != nil {
			matching = append(matching, ip)
		}
	}

	return matching
}

// DiscoverNode finds a node's VM using the specified search value and search
// type.
func (nm *NodeManager) DiscoverNode(nodeID string, searchBy cm.FindVM) error {
	ctx := context.Background()

	vmDI, err := nm.shakeOutNodeIDLookup(ctx, nodeID, searchBy)
	if err != nil {
		klog.Errorf("shakeOutNodeIDLookup failed. Err=%v", err)
		return err
	}
	oVM := vmDI.VM

	tenantRef := vmDI.IcsServer
	if vmDI.TenantRef != "" {
		tenantRef = vmDI.TenantRef
	}
	icsInstance := nm.connectionManager.ICSInstanceMap[tenantRef]

	ipFamily := []string{icfg.DefaultIPFamily}
	if icsInstance != nil {
		ipFamily = icsInstance.Cfg.IPFamilyPriority
	} else {
		klog.Warningf("Unable to find icsInstance for %s. Defaulting to ipv4.", tenantRef)
	}

	var internalNetworkSubnet *net.IPNet
	var externalNetworkSubnet *net.IPNet
	var internalVMNetworkName string
	var externalVMNetworkName string

	if nm.cpiCfg != nil {
		if nm.cpiCfg.Nodes.InternalNetworkSubnetCIDR != "" {
			_, internalNetworkSubnet, err = net.ParseCIDR(nm.cpiCfg.Nodes.InternalNetworkSubnetCIDR)
			if err != nil {
				return err
			}
		}
		if nm.cpiCfg.Nodes.ExternalNetworkSubnetCIDR != "" {
			_, externalNetworkSubnet, err = net.ParseCIDR(nm.cpiCfg.Nodes.ExternalNetworkSubnetCIDR)
			if err != nil {
				return err
			}
		}
		internalVMNetworkName = nm.cpiCfg.Nodes.InternalVMNetworkName
		externalVMNetworkName = nm.cpiCfg.Nodes.ExternalVMNetworkName
	}

	var addressMatchingEnabled bool
	if internalNetworkSubnet != nil && externalNetworkSubnet != nil {
		addressMatchingEnabled = true
	}

	found := false
	addrs := []v1.NodeAddress{}
	klog.V(2).Infof("Adding Hostname: %s", oVM.Name)
	v1helper.AddToNodeAddresses(&addrs,
		v1.NodeAddress{
			Type:    v1.NodeHostName,
			Address: oVM.Name,
		},
	)

	for _, v := range oVM.Nics {

		klog.V(6).Infof("internalVMNetworkName = %s", internalVMNetworkName)
		klog.V(6).Infof("externalVMNetworkName = %s", externalVMNetworkName)
		klog.V(6).Infof("v.Network = %s", v.Name)

		if (internalVMNetworkName != "" && !strings.EqualFold(internalVMNetworkName, v.Name)) &&
			(externalVMNetworkName != "" && !strings.EqualFold(externalVMNetworkName, v.Name)) {
			klog.V(4).Infof("Skipping device because vNIC Network=%s doesn't match internal=%s or external=%s network names",
				v.Name, internalVMNetworkName, externalVMNetworkName)
			continue
		}

		// Only return a single IP address based on the preference of IPFamily
		// Must break out of loop in the event of ipv6,ipv4 where the NIC does
		// contain a valid IPv6 and IPV4 address
		for _, family := range ipFamily {
			ips := returnIPsFromSpecificFamily(family, []string{v.IP})

			if addressMatchingEnabled {
				for _, ip := range ips {
					parsedIP := net.ParseIP(ip)
					if parsedIP == nil {
						return fmt.Errorf("can't parse IP: %s", ip)
					}

					if internalNetworkSubnet != nil && internalNetworkSubnet.Contains(parsedIP) {
						klog.V(2).Infof("Adding Internal IP by AddressMatching: %s", ip)
						v1helper.AddToNodeAddresses(&addrs,
							v1.NodeAddress{
								Type:    v1.NodeInternalIP,
								Address: ip,
							},
						)
					}

					if externalNetworkSubnet != nil && externalNetworkSubnet.Contains(parsedIP) {
						klog.V(2).Infof("Adding External IP by AddressMatching: %s", ip)
						v1helper.AddToNodeAddresses(&addrs,
							v1.NodeAddress{
								Type:    v1.NodeExternalIP,
								Address: ip,
							},
						)
					}
				}
			} else if internalVMNetworkName != "" && strings.EqualFold(internalVMNetworkName, v.Name) {
				for _, ip := range ips {
					klog.V(2).Infof("Adding Internal IP by NetworkName: %s", ip)
					v1helper.AddToNodeAddresses(&addrs,
						v1.NodeAddress{
							Type:    v1.NodeInternalIP,
							Address: ip,
						},
					)
					found = true
					break
				}
			} else if externalVMNetworkName != "" && strings.EqualFold(externalVMNetworkName, v.Name) {
				for _, ip := range ips {
					klog.V(2).Infof("Adding External IP by NetworkName: %s", ip)
					v1helper.AddToNodeAddresses(&addrs,
						v1.NodeAddress{
							Type:    v1.NodeExternalIP,
							Address: ip,
						},
					)
					found = true
					break
				}
			} else {
				for _, ip := range ips {
					klog.V(2).Infof("Adding IP: %s", ip)
					v1helper.AddToNodeAddresses(&addrs,
						v1.NodeAddress{
							Type:    v1.NodeExternalIP,
							Address: ip,
						}, v1.NodeAddress{
							Type:    v1.NodeInternalIP,
							Address: ip,
						},
					)
					found = true
					break
				}
			}

			if found {
				break
			}
		}
	}

	if !found {
		klog.Warningf("Unable to find a suitable IP address. ipFamily: %s", ipFamily)
	}
	klog.V(2).Infof("Found node %s as vm=%+v in ics=%s and datacenter=%s",
		nodeID, vmDI.VM, vmDI.IcsServer, vmDI.DataCenter.Name)
	klog.V(2).Info("Hostname: ", oVM.Name, " UUID: ", oVM.UUID)

	os := strings.Fields(oVM.GuestosLabel)[0]

	// store instance type in nodeinfo map
	instanceType := fmt.Sprintf("ics-vm.cpu-%d.mem-%dgb.os-%s",
		oVM.CPUNum,
		oVM.Memory / 1024,
		os,
	)

	nodeInfo := &NodeInfo{tenantRef: tenantRef, dataCenter: vmDI.DataCenter, vm: vmDI.VM, icsServer: vmDI.IcsServer,
		UUID: vmDI.UUID, NodeName: vmDI.NodeName, NodeType: instanceType, NodeAddresses: addrs}
	nm.addNodeInfo(nodeInfo)

	return nil
}

// GetNode gets the NodeInfo by UUID
func (nm *NodeManager) GetNode(UUID string, node *pb.Node) error {
	nodeInfo, err := nm.FindNodeInfo(UUID)
	if err != nil {
		klog.Errorf("GetNode failed err=%s", err)
		return err
	}

	node.Icenter = nodeInfo.icsServer
	node.Datacenter = nodeInfo.dataCenter.Name
	node.Name = nodeInfo.NodeName
	node.Dnsnames = make([]string, 0)
	node.Addresses = make([]string, 0)
	node.Uuid = nodeInfo.UUID

	for _, address := range nodeInfo.NodeAddresses {
		switch address.Type {
		case v1.NodeExternalIP:
			node.Addresses = append(node.Addresses, address.Address)
		case v1.NodeHostName:
			node.Dnsnames = append(node.Dnsnames, address.Address)
		default:
			klog.Warning("Unknown/unsupported address type:", address.Type)
		}
	}

	return nil
}

// ExportNodes transforms the NodeInfoList to []*pb.Node
func (nm *NodeManager) ExportNodes(icenter string, datacenter string, nodeList *[]*pb.Node) error {
	nm.nodeInfoLock.Lock()
	defer nm.nodeInfoLock.Unlock()

	if icenter != "" && datacenter != "" {
		dc, err := nm.FindDatacenterInfoInICSList(icenter, datacenter)
		if err != nil {
			return err
		}

		nm.datacenterToNodeList(dc.vmList, nodeList)
	} else if icenter != "" {
		if nm.icsList[icenter] == nil {
			return ErrICenterNotFound
		}

		for _, dc := range nm.icsList[icenter].dcList {
			nm.datacenterToNodeList(dc.vmList, nodeList)
		}
	} else {
		for _, ics := range nm.icsList {
			for _, dc := range ics.dcList {
				nm.datacenterToNodeList(dc.vmList, nodeList)
			}
		}
	}

	return nil
}

func (nm *NodeManager) datacenterToNodeList(vmList map[string]*NodeInfo, nodeList *[]*pb.Node) {
	for UUID, node := range vmList {

		// is VM currently active? if not, skip
		UUIDlower := strings.ToLower(UUID)
		if nm.nodeRegUUIDMap[UUIDlower] == nil {
			klog.V(4).Infof("Node with UUID=%s not active. Skipping.", UUIDlower)
			continue
		}

		pbNode := &pb.Node{
			Icenter:    node.icsServer,
			Datacenter: node.dataCenter.Name,
			Name:       node.NodeName,
			Dnsnames:   make([]string, 0),
			Addresses:  make([]string, 0),
			Uuid:       node.UUID,
		}
		for _, address := range node.NodeAddresses {
			switch address.Type {
			case v1.NodeExternalIP:
				pbNode.Addresses = append(pbNode.Addresses, address.Address)
			case v1.NodeHostName:
				pbNode.Dnsnames = append(pbNode.Dnsnames, address.Address)
			default:
				klog.Warning("Unknown/unsupported address type:", address.Type)
			}
		}
		*nodeList = append(*nodeList, pbNode)
	}
}

// AddNodeInfoToICSList creates a relational mapping from ICS -> DC -> VM/Node
func (nm *NodeManager) AddNodeInfoToICSList(icenter string, datacenter string, node *NodeInfo) {
	if nm.icsList[icenter] == nil {
		nm.icsList[icenter] = &ICenterInfo{
			address: icenter,
			dcList:  make(map[string]*DatacenterInfo),
		}
	}
	ics := nm.icsList[icenter]

	if ics.dcList[datacenter] == nil {
		ics.dcList[datacenter] = &DatacenterInfo{
			name:   datacenter,
			vmList: make(map[string]*NodeInfo),
		}
	}
	dc := ics.dcList[datacenter]

	dc.vmList[node.UUID] = node
}

// FindDatacenterInfoInICSList retrieves the DatacenterInfo from the tree
func (nm *NodeManager) FindDatacenterInfoInICSList(icenter string, datacenter string) (*DatacenterInfo, error) {
	ics := nm.icsList[icenter]
	if ics == nil {
		return nil, ErrICenterNotFound
	}

	dc := ics.dcList[datacenter]
	if dc == nil {
		return nil, ErrDatacenterNotFound
	}

	return dc, nil
}

// FindNodeInfo retrieves the NodeInfo from the tree
func (nm *NodeManager) FindNodeInfo(UUID string) (*NodeInfo, error) {
	nm.nodeRegInfoLock.Lock()
	defer nm.nodeRegInfoLock.Unlock()

	UUIDlower := strings.ToLower(UUID)

	if nm.nodeRegUUIDMap[UUIDlower] == nil {
		klog.Errorf("FindNodeInfo( %s ) NOT ACTIVE", UUIDlower)
		return nil, ErrVMNotFound
	}

	nodeInfo := nm.nodeUUIDMap[UUIDlower]
	if nodeInfo == nil {
		klog.Errorf("FindNodeInfo( %s ) NOT FOUND", UUIDlower)
		return nil, ErrVMNotFound
	}

	klog.V(4).Infof("FindNodeInfo( %s ) FOUND", UUIDlower)
	return nodeInfo, nil
}
