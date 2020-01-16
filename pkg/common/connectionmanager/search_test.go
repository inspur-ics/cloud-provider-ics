/*
Copyright 2019 The Kubernetes Authors.

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
	"math/rand"
	"strings"
	"testing"

	"github.com/inspur-ics/cloud-provider-ics/pkg/common/icslib"
	_"github.com/inspur-ics/ics-go-sdk/client/types"
)

func TestWhichICSandDCByNodeIdByUUID(t *testing.T) {
	config, cleanup := configFromEnvOrSim(true)
	defer cleanup()

	connMgr := NewConnectionManager(config, nil, nil)
	defer connMgr.Logout()

	// setup
	////FIXME TODO.WANGYONGCHAO
	//vm := simulator.Map.Any("VirtualMachine").(*simulator.VirtualMachine)
	//name := vm.Name
	//vm.Guest.HostName = strings.ToLower(name)
	//UUID := vm.Config.Uuid
	name := "vm-001"
	UUID := "2ed22c68-3777-4aab-8d8d-b6e5f8377b65"

	// context
	ctx := context.Background()

	info, err := connMgr.WhichICSandDCByNodeID(ctx, UUID, FindVMByUUID)
	if err != nil {
		t.Fatalf("WhichICSandDCByNodeID err=%v", err)
	}
	if info == nil {
		t.Fatalf("WhichICSandDCByNodeID info=nil")
	}

	if !strings.EqualFold(name, info.NodeName) {
		t.Fatalf("VM name mismatch %s=%s", name, info.NodeName)
	}
	if !strings.EqualFold(UUID, info.UUID) {
		t.Fatalf("VM name mismatch %s=%s", name, info.NodeName)
	}
}

func TestWhichICSandDCByNodeIdByName(t *testing.T) {
	config, cleanup := configFromEnvOrSim(true)
	defer cleanup()

	connMgr := NewConnectionManager(config, nil, nil)
	defer connMgr.Logout()

	// setup
	//FIXME TODO.WANGYONGCHAO
	//vm := simulator.Map.Any("VirtualMachine").(*simulator.VirtualMachine)
	//name := vm.Name
	//vm.Guest.HostName = strings.ToLower(name)
	//UUID := vm.Config.Uuid
	name := "vm-001"
	UUID := "2ed22c68-3777-4aab-8d8d-b6e5f8377b65"

	// context
	ctx := context.Background()

	info, err := connMgr.WhichICSandDCByNodeID(ctx, name, FindVMByName)
	if err != nil {
		t.Fatalf("WhichICSandDCByNodeID err=%v", err)
	}
	if info == nil {
		t.Fatalf("WhichICSandDCByNodeID info=nil")
	}

	if !strings.EqualFold(name, info.NodeName) {
		t.Fatalf("VM name mismatch %s=%s", name, info.NodeName)
	}
	if !strings.EqualFold(UUID, info.UUID) {
		t.Fatalf("VM name mismatch %s=%s", name, info.NodeName)
	}
}

func TestWhichICSandDCByFCDId(t *testing.T) {
	config, cleanup := configFromEnvOrSim(true)
	defer cleanup()

	connMgr := NewConnectionManager(config, nil, nil)
	defer connMgr.Logout()

	// context
	ctx := context.Background()

	/*
	 * Setup
	 */
	//// Get a simulator DS
	////FIXME TODO.WANGYONGCHAO
	//myds := simulator.Map.Any("Datastore").(*simulator.Datastore)

	items, err := connMgr.ListAllICSandDCPairs(ctx)
	if err != nil {
		t.Fatalf("ListAllICSandDCPairs err=%v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ListAllICSandDCPairs items should be 2 but count=%d", len(items))
	}

	randDC := items[rand.Intn(len(items))]

	datastoreName := "localdata01"
	datastoreType := icslib.TypeDatastore
	volName := "myfcd"
	volSizeMB := int64(1024) //1GB

	err = randDC.DataCenter.CreateFirstClassDisk(ctx, datastoreName, datastoreType, volName, volSizeMB)
	if err != nil {
		t.Fatalf("CreateFirstClassDisk err=%v", err)
	}

	firstClassDisk, err := randDC.DataCenter.GetFirstClassDisk(
		ctx, datastoreName, datastoreType, volName, icslib.FindFCDByName)
	if err != nil {
		t.Fatalf("GetFirstClassDisk err=%v", err)
	}

	//fcdID := firstClassDisk.Config.Id.Id
	fcdID := firstClassDisk.ID
	/*
	 * Setup
	 */

	// call WhichICSandDCByFCDId
	fcdObj, err := connMgr.WhichICSandDCByFCDId(ctx, fcdID)
	if err != nil {
		t.Fatalf("WhichVCandDCByFCDId err=%v", err)
	}
	if fcdObj == nil {
		t.Fatalf("WhichVCandDCByFCDId fcdObj=nil")
	}

	//if !strings.EqualFold(fcdID, fcdObj.FCDInfo.Config.Id.Id) {
	//	t.Errorf("FCD ID mismatch %s=%s", fcdID, fcdObj.FCDInfo.Config.Id.Id)
	//}
	//if datastoreType != fcdObj.FCDInfo.ParentType {
	//	t.Errorf("FCD DatastoreType mismatch %v=%v", datastoreType, fcdObj.FCDInfo.ParentType)
	//}
	//if !strings.EqualFold(datastoreName, fcdObj.FCDInfo.DatastoreInfo.Info.Name) {
	//	t.Errorf("FCD Datastore mismatch %s=%s", datastoreName, fcdObj.FCDInfo.DatastoreInfo.Info.Name)
	//}
	//if volSizeMB != fcdObj.FCDInfo.Config.CapacityInMB {
	//	t.Errorf("FCD Size mismatch %d=%d", volSizeMB, fcdObj.FCDInfo.Config.CapacityInMB)
	//}
}
