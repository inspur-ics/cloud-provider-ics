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
	"sort"
	"strings"
	"time"

	"k8s.io/klog"

	"github.com/inspur-ics/cloud-provider-ics/pkg/common/icslib"
)

// ListAllICSandDCPairs returns all ICS/DC pairs
func (cm *ConnectionManager) ListAllICSandDCPairs(ctx context.Context) ([]*ListDiscoveryInfo, error) {
	klog.V(4).Infof("ListAllICSandDCPairs called")

	listOfICSAndDCPairs := make([]*ListDiscoveryInfo, 0)

	for _, vsi := range cm.ICSInstanceMap {
		var datacenterObjs []*icslib.Datacenter

		var err error
		for i := 0; i < NumConnectionAttempts; i++ {
			err = cm.Connect(ctx, vsi)
			if err == nil {
				break
			}
			time.Sleep(time.Duration(RetryAttemptDelaySecs) * time.Second)
		}

		if err != nil {
			klog.Error("Connect error ics:", err)
			continue
		}

		if vsi.Cfg.Datacenters == "" {
			datacenterObjs, err = icslib.GetAllDatacenter(ctx, vsi.Conn)
			if err != nil {
				klog.Error("GetAllDatacenter error dc:", err)
				continue
			}
		} else {
			datacenters := strings.Split(vsi.Cfg.Datacenters, ",")
			for _, dc := range datacenters {
				dc = strings.TrimSpace(dc)
				if dc == "" {
					continue
				}
				datacenterObj, err := icslib.GetDatacenter(ctx, vsi.Conn, dc)
				if err != nil {
					klog.Error("GetDatacenter error dc:", err)
					continue
				}
				datacenterObjs = append(datacenterObjs, datacenterObj)
			}
		}

		for _, datacenterObj := range datacenterObjs {
			listOfICSAndDCPairs = append(listOfICSAndDCPairs, &ListDiscoveryInfo{
				TenantRef:  vsi.Cfg.TenantRef,
				IcsServer:   vsi.Cfg.ICenterIP,
				DataCenter: datacenterObj,
			})
		}
	}

	sort.Slice(listOfICSAndDCPairs, func(i, j int) bool {
		return strings.Compare(listOfICSAndDCPairs[i].IcsServer, listOfICSAndDCPairs[j].IcsServer) > 0 &&
			strings.Compare(listOfICSAndDCPairs[i].DataCenter.Name, listOfICSAndDCPairs[j].DataCenter.Name) > 0
	})

	return listOfICSAndDCPairs, nil
}
