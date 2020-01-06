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
//	"os"

//	"github.com/inspur-ics/cloud-provider-ics/pkg/common/goicssdk"
	"k8s.io/klog"

	k8stypes "k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"

//	cm "github.com/inspur-ics/cloud-provider-ics/pkg/common/connectionmanager"
)

func newZones(nodeManager *NodeManager, zone string, region string) cloudprovider.Zones {
	return &zones{
		nodeManager: nodeManager,
		zone:        zone,
		region:      region,
	}
}

// GetZone implements Zones.GetZone for In-Tree providers
func (z *zones) GetZone(ctx context.Context) (cloudprovider.Zone, error) {
	klog.V(4).Info("zones.GetZone() called")

	zone := cloudprovider.Zone{}
/*
	nodeName, err := os.Hostname()
	if err != nil {
		klog.V(2).Info("Failed to get hostname. Err: ", err)
		return zone, err
	}

	node, ok := z.nodeManager.nodeNameMap[nodeName]
	if !ok {
		klog.V(2).Info("zones.GetZone() NOT FOUND with ", nodeName)
		return zone, ErrVMNotFound
	}
 */

//ics 
//vm's host summary
/*
	vmHost, err := node.vm.HostSystem(ctx)
	if err != nil {
		klog.Errorf("Failed to get host system for VM: %q. err: %+v", node.vm.InventoryPath, err)
		return zone, err
	}

	var oHost goicssdk.HostSystem
	err = vmHost.Properties(ctx, vmHost.Reference(), []string{"summary"}, &oHost)
	if err != nil {
		klog.Errorf("Failed to get host system properties. err: %+v", err)
		return zone, err
	}
	klog.V(4).Infof("Host owning VM is %s", oHost.Summary.Config.Name)

	zoneResult, err := z.nodeManager.connectionManager.LookupZoneByMoref(
		ctx, node.tenantRef, vmHost.Reference(), z.zone, z.region)
	if err != nil {
		klog.Errorf("Failed to get host system properties. err: %+v", err)
		return zone, err
	}

	zone.FailureDomain = zoneResult[cm.ZoneLabel]
	zone.Region = zoneResult[cm.RegionLabel]
*/
//ics
	return zone, nil
}

// GetZone implements Zones.GetZone for In-Tree providers

// GetZoneByNodeName implements Zones.GetZone for Out-Tree providers
func (z *zones) GetZoneByNodeName(ctx context.Context, nodeName k8stypes.NodeName) (cloudprovider.Zone, error) {
	klog.V(4).Info("zones.GetZoneByNodeName() called with ", string(nodeName))

	zone := cloudprovider.Zone{}
/*
	node, ok := z.nodeManager.nodeNameMap[string(nodeName)]
	if !ok {
		klog.V(2).Info("zones.GetZoneByNodeName() NOT FOUND with ", string(nodeName))
		return zone, ErrVMNotFound
	}
 */

//ics
//vm's host summary
/*
	vmHost, err := node.vm.HostSystem(ctx)
	if err != nil {
		klog.Errorf("Failed to get host system for VM: %q. err: %+v", node.vm.InventoryPath, err)
		return zone, err
	}

	var oHost goicssdk.HostSystem
	err = vmHost.Properties(ctx, vmHost.Reference(), []string{"summary"}, &oHost)
	if err != nil {
		klog.Errorf("Failed to get host system properties. err: %+v", err)
		return zone, err
	}
	klog.V(4).Infof("Host owning VM is %s", oHost.Summary.Config.Name)

	zoneResult, err := z.nodeManager.connectionManager.LookupZoneByMoref(
		ctx, node.tenantRef, vmHost.Reference(), z.zone, z.region)
	if err != nil {
		klog.Errorf("Failed to get host system properties. err: %+v", err)
		return zone, err
	}

	zone.FailureDomain = zoneResult[cm.ZoneLabel]
	zone.Region = zoneResult[cm.RegionLabel]
*/
//ics
	return zone, nil
}

// GetZoneByProviderID implements Zones.GetZone for Out-Tree providers
func (z *zones) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {
	klog.V(4).Info("zones.GetZoneByProviderID() called with ", providerID)

	zone := cloudprovider.Zone{}
/*
	uid := GetUUIDFromProviderID(providerID)

	node, ok := z.nodeManager.nodeUUIDMap[uid]
	if !ok {
		klog.V(2).Info("zones.GetZoneByProviderID() NOT FOUND with ", uid)
		return zone, ErrVMNotFound
	}
*/

//ics
//vm's host summary
/*
	vmHost, err := node.vm.HostSystem(ctx)
	if err != nil {
		klog.Errorf("Failed to get host system for VM: %q. err: %+v", node.vm.InventoryPath, err)
		return zone, err
	}

	var oHost goicssdk.HostSystem
	err = vmHost.Properties(ctx, vmHost.Reference(), []string{"summary"}, &oHost)
	if err != nil {
		klog.Errorf("Failed to get host system properties. err: %+v", err)
		return zone, err
	}
	klog.V(4).Infof("Host owning VM is %s", oHost.Summary.Config.Name)

	zoneResult, err := z.nodeManager.connectionManager.LookupZoneByMoref(
		ctx, node.tenantRef, vmHost.Reference(), z.zone, z.region)
	if err != nil {
		klog.Errorf("Failed to get host system properties. err: %+v", err)
		return zone, err
	}

	zone.FailureDomain = zoneResult[cm.ZoneLabel]
	zone.Region = zoneResult[cm.RegionLabel]
*/	
//ics
	return zone, nil
}
