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

package icslib

import (
	"context"
	"strings"

	icsgo "github.com/inspur-ics/ics-go-sdk"
	"github.com/inspur-ics/ics-go-sdk/client/types"
	dc "github.com/inspur-ics/ics-go-sdk/datacenter"
	st "github.com/inspur-ics/ics-go-sdk/storage"
	"github.com/inspur-ics/ics-go-sdk/vm"
	"k8s.io/klog"
)

// Datacenter extends the govmomi Datacenter object
type Datacenter struct {
	Common
	*types.Datacenter
}

// GetDatacenter returns the DataCenter Object for the given datacenterPath
// If datacenter is located in a folder, include full path to datacenter else just provide the datacenter name
func GetDatacenter(ctx context.Context, connection *icsgo.ICSConnection, datacenterPath string) (*Datacenter, error) {
	client, err := connection.GetClient()
	if err != nil {
		return nil, err
	}
	datacenterService := dc.NewDatacenterService(client)
	datacenter, err := datacenterService.GetDatacenter(ctx, datacenterPath)
	if err != nil {
		klog.Errorf("Failed to find the datacenter: %s. err: %+v", datacenterPath, err)
		return nil, err
	}
	dc := Datacenter{Common{connection}, datacenter}
	return &dc, nil
}

// GetAllDatacenter returns all the DataCenter Objects
func GetAllDatacenter(ctx context.Context, connection *icsgo.ICSConnection) ([]*Datacenter, error) {
	var dcs []*Datacenter
	client, err := connection.GetClient()
	if err != nil {
		return nil, err
	}
	datacenterService := dc.NewDatacenterService(client)
	datacenters, err := datacenterService.GetAllDatacenters(ctx)
	if err != nil {
		klog.Errorf("Failed to find the datacenter. err: %+v", err)
		return nil, err
	}
	for _, datacenter := range datacenters {
		dcs = append(dcs, &(Datacenter{Common{connection}, datacenter}))
	}

	return dcs, nil
}

// GetNumberOfDatacenters returns the number of DataCenters in this vCenter
func GetNumberOfDatacenters(ctx context.Context, connection *icsgo.ICSConnection) (int, error) {
	client, err := connection.GetClient()
	if err != nil {
		return 0, err
	}
	datacenterService := dc.NewDatacenterService(client)
	datacenters, err := datacenterService.GetAllDatacenters(ctx)
	if err != nil {
		klog.Errorf("Failed to find the datacenter. err: %+v", err)
		return 0, err
	}
	return len(datacenters), nil
}

// GetVMByIP gets the VM object from the given IP address
func (dc *Datacenter) GetVMByIP(ctx context.Context, ipAddy string) (*VirtualMachine, error) {
	vmService := vm.NewVirtualMachineService(dc.Client())
	ipAddy = strings.ToLower(strings.TrimSpace(ipAddy))
	vm, err := vmService.GetVMByIP(ctx, ipAddy)
	if err != nil {
		klog.Errorf("Failed to find VM by IP. VM IP: %s, err: %+v", ipAddy, err)
		return nil, err
	}
	if vm == nil {
		klog.Errorf("Unable to find VM by IP. VM IP: %s", ipAddy)
		return nil, ErrNoVMFound
	}
	virtualMachine := VirtualMachine{Common{dc.Con}, vm, dc}
	return &virtualMachine, nil
}

// GetVMByDNSName gets the VM object from the given dns name
func (dc *Datacenter) GetVMByDNSName(ctx context.Context, dnsName string) (*VirtualMachine, error) {
	vmService := vm.NewVirtualMachineService(dc.Client())
	dnsName = strings.ToLower(strings.TrimSpace(dnsName))
	vm, err := vmService.GetVMByName(ctx, dnsName)
	if err != nil {
		klog.Errorf("Failed to find VM by DNS Name. VM DNS Name: %s, err: %+v", dnsName, err)
		return nil, err
	}
	if vm == nil {
		klog.Errorf("Unable to find VM by DNS Name. VM DNS Name: %s", dnsName)
		return nil, ErrNoVMFound
	}
	virtualMachine := VirtualMachine{Common{dc.Con}, vm, dc}
	return &virtualMachine, nil
}

// GetVMByUUID gets the VM object from the given vmUUID
func (dc *Datacenter) GetVMByUUID(ctx context.Context, vmUUID string) (*VirtualMachine, error) {
	vmService := vm.NewVirtualMachineService(dc.Client())
	vmUUID = strings.ToLower(strings.TrimSpace(vmUUID))
	vm, err := vmService.GetVMByUUID(ctx, vmUUID)
	if err != nil {
		klog.Errorf("Failed to find VM by UUID. VM UUID: %s, err: %+v", vmUUID, err)
		return nil, err
	}
	if vm == nil {
		klog.Errorf("Unable to find VM by UUID. VM UUID: %s", vmUUID)
		return nil, ErrNoVMFound
	}
	virtualMachine := VirtualMachine{Common{dc.Con}, vm, dc}
	return &virtualMachine, nil
}

// GetVMByPath gets the VM object from the given vmPath
// vmPath should be the full path to VM and not just the name
func (dc *Datacenter) GetVMByPath(ctx context.Context, vmPath string) (*VirtualMachine, error) {
	vmService := vm.NewVirtualMachineService(dc.Client())
	vmPath = strings.ToLower(strings.TrimSpace(vmPath))
	vm, err := vmService.GetVMByPath(ctx, vmPath)
	if err != nil {
		klog.Errorf("Failed to find VM by Path. VM Path: %s, err: %+v", vmPath, err)
		return nil, err
	}
	virtualMachine := VirtualMachine{Common{dc.Con}, vm, dc}
	return &virtualMachine, nil
}

// GetAllDatastores gets the datastore URL to DatastoreInfo map for all the datastores in
// the datacenter.
func (dc *Datacenter) GetAllDatastores(ctx context.Context) (map[string]*DatastoreInfo, error) {
	sts := st.NewStorageService(dc.Client())
	datastores, err := sts.GetAllDatastores(ctx, dc.ID)
	if err != nil {
		klog.Errorf("Failed to get all the datastores. err: %+v", err)
		return nil, err
	}
	dsURLInfoMap := make(map[string]*DatastoreInfo)
	for _, dsMo := range datastores {
		dsURLInfoMap[dsMo.MountPath] = &DatastoreInfo{
			&Datastore{Common{Con: dc.Con},
				dsMo,
				dc}}
	}
	klog.V(9).Infof("dsURLInfoMap : %+v", dsURLInfoMap)
	return dsURLInfoMap, nil
}

// GetDatastoreByPath gets the Datastore object from the given vmDiskPath
func (dc *Datacenter) GetDatastoreByPath(ctx context.Context, vmDiskPath string) (*DatastoreInfo, error) {
	return dc.GetDatastoreByName(ctx, vmDiskPath)
}

// GetDatastoreByName gets the Datastore object for the given datastore name
func (dc *Datacenter) GetDatastoreByName(ctx context.Context, name string) (*DatastoreInfo, error) {
	sts := st.NewStorageService(dc.Client())
	datastore, err := sts.GetStorageInfoByName(ctx, name)
	if err != nil {
		klog.Errorf("Failed while searching for datastore: %s. err: %+v", name, err)
		return nil, err
	}

	return &DatastoreInfo{
		&Datastore{Common{Con: dc.Con},
			datastore,
			dc}}, nil
}

// GetDatastoreClusterByName gets the DatastoreCluster object for the given name
func (dc *Datacenter) GetDatastoreClusterByName(ctx context.Context, name string) (*StoragePodInfo, error) {
	return nil, nil
}

// CreateFirstClassDisk creates a new first class disk.
func (dc *Datacenter) CreateFirstClassDisk(ctx context.Context,
	datastoreName string, datastoreType ParentDatastoreType,
	diskName string, diskSize int64) error {
	return nil
}

// GetFirstClassDisk searches for an existing FCD.
func (dc *Datacenter) GetFirstClassDisk(ctx context.Context,
	datastoreName string, datastoreType ParentDatastoreType,
	diskID string, findBy FindFCD) (*FirstClassDiskInfo, error) {
	return nil, nil
}

// DoesFirstClassDiskExist returns information about an FCD if it exists.
func (dc *Datacenter) DoesFirstClassDiskExist(ctx context.Context, fcdID string) (*FirstClassDiskInfo, error) {
	datastores, err := dc.GetAllDatastores(ctx)
	if err != nil {
		klog.Errorf("GetAllDatastores failed. Err: %v", err)
		return nil, err
	}

	for _, datastore := range datastores {
		fcd, err := datastore.GetFirstClassDiskInfo(ctx, fcdID, FindFCDByID)
		if err == nil {
			klog.Infof("DoesFirstClassDiskExist(%s): FOUND", fcdID)
			return fcd, nil
		}
	}

	klog.Infof("DoesFirstClassDiskExist(%s): NOT FOUND", fcdID)
	return nil, ErrNoDiskIDFound
}

// GetFirstClassDiskInfo gets a specific first class disks (FCD) on this datastore
func (di *DatastoreInfo) GetFirstClassDiskInfo(ctx context.Context, diskID string, findBy FindFCD) (*FirstClassDiskInfo, error) {
	return nil, ErrNoDiskIDFound
}
