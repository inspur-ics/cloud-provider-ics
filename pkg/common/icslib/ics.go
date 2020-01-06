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
	"errors"
//	"fmt"
//	"path/filepath"
//	"strings"
	"sync"

	"k8s.io/klog"
)

// Error Messages
const (
	FileAlreadyExistErrMsg         = "File requested already exist"
	NoDevicesFoundErrMsg           = "No devices found"
	DiskNotFoundErrMsg             = "No vSphere disk ID/Name found"
	InvalidVolumeOptionsErrMsg     = "VolumeOptions verification failed"
	NoVMFoundErrMsg                = "No VM found"
	NoZoneRegionFoundErrMsg        = "Unable to find the Zone/Region pair"
	NoDatastoreFoundErrMsg         = "Datastore not found"
	NoDatacenterFoundErrMsg        = "Datacenter not found"
	NoDataStoreClustersFoundErrMsg = "No DatastoreClusters Found"
)

// Error constants
var (
	ErrFileAlreadyExist         = errors.New(FileAlreadyExistErrMsg)
	ErrNoDevicesFound           = errors.New(NoDevicesFoundErrMsg)
	ErrNoDiskIDFound            = errors.New(DiskNotFoundErrMsg)
	ErrInvalidVolumeOptions     = errors.New(InvalidVolumeOptionsErrMsg)
	ErrNoVMFound                = errors.New(NoVMFoundErrMsg)
	ErrNoZoneRegionFound        = errors.New(NoZoneRegionFoundErrMsg)
	ErrNoDatastoreFound         = errors.New(NoDatastoreFoundErrMsg)
	ErrNoDatacenterFound        = errors.New(NoDatacenterFoundErrMsg)
	ErrNoDataStoreClustersFound = errors.New(NoDataStoreClustersFoundErrMsg)
)

type ICsConnection struct {
//	Client            *vim25.Client
    Client            string
	Username          string
	Password          string
	Hostname          string
	Port              string
	CACert            string
	Thumbprint        string
	Insecure          bool
	RoundTripperCount uint
	credentialsLock   sync.Mutex
}
// Connect makes connection to vCenter and sets VSphereConnection.Client.
// If connection.Client is already set, it obtains the existing user session.
// if user session is not valid, connection.Client will be set to the new client.
func (connection *ICsConnection) Connect(ctx context.Context) error {
	return nil
}

// Logout calls SessionManager.Logout for the given connection.
func (connection *ICsConnection) Logout(ctx context.Context) {
	return
}

// Datacenter extends the govmomi Datacenter object
type Datacenter struct {
//	*object.Datacenter
//ics added
	 dcname          string
}

// VirtualMachine extends the govmomi VirtualMachine object
type VirtualMachine struct {
//	*object.VirtualMachine
	Datacenter *Datacenter
}

// IsActive checks if the VM is active.
// Returns true if VM is in poweredOn state.
func (vm *VirtualMachine) IsActive(ctx context.Context) (bool, error) {
	return false, nil
}

// IsInvalidCredentialsError returns true if error is of type InvalidLogin
func IsInvalidCredentialsError(err error) bool {
	return false
}

// GetDatacenter returns the DataCenter Object for the given datacenterPath
// If datacenter is located in a folder, include full path to datacenter else just provide the datacenter name
func GetDatacenter(ctx context.Context, connection *ICsConnection, datacenterPath string) (*Datacenter, error) {
	dc := Datacenter{}
	return &dc, nil
}

// GetAllDatacenter returns all the DataCenter Objects
func GetAllDatacenter(ctx context.Context, connection *ICsConnection) ([]*Datacenter, error) {
	var dc []*Datacenter
	return dc, nil
}

// GetNumberOfDatacenters returns the number of DataCenters in this vCenter
func GetNumberOfDatacenters(ctx context.Context, connection *ICsConnection) (int, error) {
	return 10, nil
}

// FirstClassDiskInfo extends the govmomi FirstClassDisk object
type FirstClassDiskInfo struct {
//	*FirstClassDisk

//	DatastoreInfo  *DatastoreInfo
//	StoragePodInfo *StoragePodInfo
	//ics added
	name          string
}

// DoesFirstClassDiskExist returns information about an FCD if it exists.
func (dc *Datacenter) DoesFirstClassDiskExist(ctx context.Context, fcdID string) (*FirstClassDiskInfo, error) {
	klog.Infof("DoesFirstClassDiskExist(%s): NOT FOUND", fcdID)
	return nil, ErrNoDiskIDFound
}

// GetVMByIP gets the VM object from the given IP address
func (dc *Datacenter) GetVMByIP(ctx context.Context, ipAddy string) (*VirtualMachine, error) {
	return nil, nil
}

// GetVMByDNSName gets the VM object from the given dns name
func (dc *Datacenter) GetVMByDNSName(ctx context.Context, dnsName string) (*VirtualMachine, error) {
	return nil, nil
}

// GetVMByUUID gets the VM object from the given vmUUID
func (dc *Datacenter) GetVMByUUID(ctx context.Context, vmUUID string) (*VirtualMachine, error) {
	return nil, nil
}

// GetVMByUUID gets the VM object from the given vmUUID
func (dc *Datacenter) Name() string {
	return dc.dcname
}


// UpdateCredentials updates username and password.
// Note: Updated username and password will be used when there is no session active
func (connection *ICsConnection) UpdateCredentials(username string, password string) {
	connection.credentialsLock.Lock()
	defer connection.credentialsLock.Unlock()
	connection.Username = username
	connection.Password = password
}
/********************************************************************************************/
