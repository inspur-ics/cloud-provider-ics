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
	"context"
	"strings"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/inspur-ics/cloud-provider-ics/pkg/common/icslib"
)

// String returns the string representation of the FindVM constant.
func (f FindVM) String() string {
	switch f {
	case FindVMByUUID:
		return "byUUID"
	case FindVMByName:
		return "byName"
	case FindVMByIP:
		return "byIP"
	default:
		return "byUnknown"
	}
}

// WhichICSandDCByNodeID finds the ICS/DC combo that owns a particular VM
func (cm *ConnectionManager) WhichICSandDCByNodeID(ctx context.Context, nodeID string, searchBy FindVM) (*VMDiscoveryInfo, error) {
	if nodeID == "" {
		klog.V(3).Info("WhichICSandDCByNodeID called but nodeID is empty")
		return nil, icslib.ErrNoVMFound
	}
	type vmSearch struct {
		tenantRef  string
		ics        string
		datacenter *icslib.Datacenter
	}

	var mutex = &sync.Mutex{}
	var globalErrMutex = &sync.Mutex{}
	var queueChannel chan *vmSearch
	var wg sync.WaitGroup
	var globalErr *error

	queueChannel = make(chan *vmSearch, QueueSize)

	myNodeID := nodeID
	switch searchBy {
	case FindVMByUUID:
		klog.V(3).Info("WhichICSandDCByNodeID by UUID")
		myNodeID = strings.TrimSpace(strings.ToLower(nodeID))
	case FindVMByIP:
		klog.V(3).Info("WhichICSandDCByNodeID by IP")
	default:
		klog.V(3).Info("WhichICSandDCByNodeID by Name")
	}
	klog.V(2).Info("WhichICSandDCByNodeID nodeID: ", myNodeID)

	vmFound := false
	globalErr = nil

	setGlobalErr := func(err error) {
		globalErrMutex.Lock()
		globalErr = &err
		globalErrMutex.Unlock()
	}

	setVMFound := func(found bool) {
		mutex.Lock()
		vmFound = found
		mutex.Unlock()
	}

	getVMFound := func() bool {
		mutex.Lock()
		found := vmFound
		mutex.Unlock()
		return found
	}

	go func() {
		for _, instance := range cm.ICSInstanceMap {
			var datacenterObjs []*icslib.Datacenter

			if getVMFound() {
				break
			}

			var err error
			for i := 0; i < NumConnectionAttempts; i++ {
				err = cm.Connect(ctx, instance)
				if err == nil {
					break
				}
				time.Sleep(time.Duration(RetryAttemptDelaySecs) * time.Second)
			}

			if err != nil {
				klog.Errorf("WhichICSandDCByNodeID error ics:%+v  err:%v\n", instance.Cfg, err)
				setGlobalErr(err)
				continue
			}

			if instance.Cfg.Datacenters == "" {
				datacenterObjs, err = icslib.GetAllDatacenter(ctx, instance.Conn)
				if err != nil {
					klog.Error("WhichICSandDCByNodeID error dc:", err)
					setGlobalErr(err)
					continue
				}
			} else {
				datacenters := strings.Split(instance.Cfg.Datacenters, ",")
				for _, dc := range datacenters {
					dc = strings.TrimSpace(dc)
					if dc == "" {
						continue
					}
					datacenterObj, err := icslib.GetDatacenter(ctx, instance.Conn, dc)
					if err != nil {
						klog.Error("WhichICSandDCByNodeID error dc:", err)
						setGlobalErr(err)
						continue
					}
					datacenterObjs = append(datacenterObjs, datacenterObj)
				}
			}

			for _, datacenterObj := range datacenterObjs {
				if getVMFound() {
					break
				}

				klog.V(4).Infof("Finding node %s in ics=%s and datacenter=%s", myNodeID, instance.Cfg.ICenterIP, datacenterObj.Name)
				queueChannel <- &vmSearch{
					tenantRef:  instance.Cfg.TenantRef,
					ics:        instance.Cfg.ICenterIP,
					datacenter: datacenterObj,
				}
			}
		}
		close(queueChannel)
	}()

	var vmInfo *VMDiscoveryInfo
	for i := 0; i < PoolSize; i++ {
		wg.Add(1)
		go func() {
			for res := range queueChannel {
				var vm *icslib.VirtualMachine
				var err error

				switch searchBy {
				case FindVMByUUID:
					vm, err = res.datacenter.GetVMByUUID(ctx, myNodeID)
				case FindVMByIP:
					vm, err = res.datacenter.GetVMByIP(ctx, myNodeID)
				default:
					vm, err = res.datacenter.GetVMByDNSName(ctx, myNodeID)
				}

				if err != nil || vm == nil {
					klog.Errorf("Error while looking for vm=%s(%s) in ics=%s and datacenter=%s: %v",
						myNodeID, searchBy, res.ics, res.datacenter.Name, err)
					if err != icslib.ErrNoVMFound {
						setGlobalErr(err)
					} else {
						klog.V(2).Infof("Did not find node %s in ics=%s and datacenter=%s",
							myNodeID, res.ics, res.datacenter.Name)
					}
					continue
				}

				klog.V(5).Infof("ICS GetVMBy %s, vm=%+v and datacenter=%+v",
					searchBy, vm.VirtualMachine, vm.Datacenter)

				hostName := vm.VirtualMachine.Name
				if searchBy == FindVMByIP {
					klog.V(2).Infof("WhichICSandDCByNodeID by IP. Overriding VMName from=%s to to=%s", vm.VirtualMachine.Name, myNodeID)
					hostName = myNodeID
				}

				UUID := strings.ToLower(strings.TrimSpace(vm.VirtualMachine.UUID))

				klog.V(2).Infof("Found node %s as vm=%+v in ics=%s and datacenter=%s",
					nodeID, vm.VirtualMachine, res.ics, res.datacenter.Name)
				klog.V(2).Infof("Hostname: %s, UUID: %s", hostName, UUID)

				vmInfo = &VMDiscoveryInfo{TenantRef: res.tenantRef, DataCenter: res.datacenter, VM: vm, IcsServer: res.ics,
					UUID: UUID, NodeName: hostName}
				setVMFound(true)
				break
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if vmFound {
		return vmInfo, nil
	}
	if globalErr != nil {
		return nil, *globalErr
	}

	klog.V(4).Infof("WhichICSandDCByNodeID: %q vm not found", myNodeID)
	return nil, icslib.ErrNoVMFound
}

// WhichICSandDCByFCDId searches for an FCD using the provided ID.
func (cm *ConnectionManager) WhichICSandDCByFCDId(ctx context.Context, fcdID string) (*FcdDiscoveryInfo, error) {

	if fcdID == "" {
		klog.V(3).Info("WhichICSandDCByFCDId called but fcdID is empty")
		return nil, icslib.ErrNoDiskIDFound
	}
	klog.V(2).Info("WhichICSandDCByFCDId fcdID: ", fcdID)

	type fcdSearch struct {
		tenantRef  string
		ics        string
		datacenter *icslib.Datacenter
	}

	var mutex = &sync.Mutex{}
	var globalErrMutex = &sync.Mutex{}
	var queueChannel chan *fcdSearch
	var wg sync.WaitGroup
	var globalErr *error

	queueChannel = make(chan *fcdSearch, QueueSize)

	fcdFound := false
	globalErr = nil

	setGlobalErr := func(err error) {
		globalErrMutex.Lock()
		globalErr = &err
		globalErrMutex.Unlock()
	}

	setFCDFound := func(found bool) {
		mutex.Lock()
		fcdFound = found
		mutex.Unlock()
	}

	getFCDFound := func() bool {
		mutex.Lock()
		found := fcdFound
		mutex.Unlock()
		return found
	}

	go func() {
		for _, instance := range cm.ICSInstanceMap {
			var datacenterObjs []*icslib.Datacenter

			if getFCDFound() {
				break
			}

			var err error
			for i := 0; i < NumConnectionAttempts; i++ {
				err = cm.Connect(ctx, instance)
				if err == nil {
					break
				}
				time.Sleep(time.Duration(RetryAttemptDelaySecs) * time.Second)
			}

			if err != nil {
				klog.Error("WhichICSandDCByFCDId error ics:", err)
				setGlobalErr(err)
				continue
			}

			if instance.Cfg.Datacenters == "" {
				datacenterObjs, err = icslib.GetAllDatacenter(ctx, instance.Conn)
				if err != nil {
					klog.Error("WhichICSandDCByFCDId error dc:", err)
					setGlobalErr(err)
					continue
				}
			} else {
				datacenters := strings.Split(instance.Cfg.Datacenters, ",")
				for _, dc := range datacenters {
					dc = strings.TrimSpace(dc)
					if dc == "" {
						continue
					}
					datacenterObj, err := icslib.GetDatacenter(ctx, instance.Conn, dc)
					if err != nil {
						klog.Error("WhichICSandDCByFCDId error dc:", err)
						setGlobalErr(err)
						continue
					}
					datacenterObjs = append(datacenterObjs, datacenterObj)
				}
			}

			for _, datacenterObj := range datacenterObjs {
				if getFCDFound() {
					break
				}

				klog.V(4).Infof("Finding FCD %s in ics=%s and datacenter=%s", fcdID, instance.Cfg.ICenterIP, datacenterObj.Name)
				queueChannel <- &fcdSearch{
					tenantRef:  instance.Cfg.TenantRef,
					ics:        instance.Cfg.ICenterIP,
					datacenter: datacenterObj,
				}
			}
		}
		close(queueChannel)
	}()

	var fcdInfo *FcdDiscoveryInfo
	for i := 0; i < PoolSize; i++ {
		wg.Add(1)
		go func() {
			for res := range queueChannel {

				//FIXME TODO.WANGYONGCHAO
				fcd, err := res.datacenter.DoesFirstClassDiskExist(ctx, fcdID)
				if err != nil {
					klog.Errorf("Error while looking for FCD=%+v in ics=%s and datacenter=%s: %v",
						fcd, res.ics, res.datacenter.Name, err)
					if err != icslib.ErrNoDiskIDFound {
						setGlobalErr(err)
					} else {
						klog.V(2).Infof("Did not find FCD %s in ics=%s and datacenter=%s",
							fcdID, res.ics, res.datacenter.Name)
					}
					continue
				}

				klog.V(2).Infof("Found FCD %s as vm=%+v in ics=%s and datacenter=%s",
					fcdID, fcd, res.ics, res.datacenter.Name)

				fcdInfo = &FcdDiscoveryInfo{TenantRef: res.tenantRef, DataCenter: res.datacenter, FCDInfo: fcd, IcsServer: res.ics}
				setFCDFound(true)
				break
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if fcdFound {
		return fcdInfo, nil
	}
	if globalErr != nil {
		return nil, *globalErr
	}

	klog.V(4).Infof("WhichICSandDCByFCDId: %q FCD not found", fcdID)
	return nil, icslib.ErrNoDiskIDFound
}
