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
	"github.com/inspur-ics/ics-go-sdk/client/types"
	"k8s.io/klog"
)

// VirtualMachine extends the govmomi VirtualMachine object
type VirtualMachine struct {
	Common
	*types.VirtualMachine
	Datacenter *Datacenter
}

// IsActive checks if the VM is active.
// Returns true if VM is in poweredOn state.
func (vm *VirtualMachine) IsActive(ctx context.Context) (bool, error) {
	vmMo, err := vm.Datacenter.GetVMByUUID(ctx, vm.UUID)
	if err != nil {
		klog.Errorf("Failed to get VM Managed object with property summary. err: +%v", err)
		return false, err
	}
	if vmMo.Status == "STARTED" {
		return true, nil
	}

	return false, nil
}