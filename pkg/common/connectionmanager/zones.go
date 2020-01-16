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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/inspur-ics/cloud-provider-ics/pkg/common/icslib"
	icsgo "github.com/inspur-ics/ics-go-sdk"
	rest "github.com/inspur-ics/ics-go-sdk/client"
	ht "github.com/inspur-ics/ics-go-sdk/host"
	tags "github.com/inspur-ics/ics-go-sdk/tag"
	"k8s.io/klog"
)

//Well-known keys for k/v maps
const (

	// ZoneLabel is the label for zones.
	ZoneLabel = "Zone"

	// RegionLabel is the label for regions.
	RegionLabel = "Region"
)

// WhichICSandDCByZone gets the corresponding ICS+DC combo that supports the availability zone
func (cm *ConnectionManager) WhichICSandDCByZone(ctx context.Context,
	zoneLabel string, regionLabel string, zoneLooking string, regionLooking string) (*ZoneDiscoveryInfo, error) {
	klog.V(4).Infof("WhichICSandDCByZone called with zone: %s and region: %s", zoneLooking, regionLooking)

	// Need at least one ICS
	numOfICSs := len(cm.ICSInstanceMap)
	if numOfICSs == 0 {
		err := ErrMustHaveAtLeastOneICSDC
		klog.Errorf("%v", err)
		return nil, err
	}

	if numOfICSs == 1 {
		klog.Info("Single ICS Detected")
		return cm.getDIFromSingleICS(ctx, zoneLabel, regionLabel, zoneLooking, regionLooking)
	}

	klog.Info("Multi ICS Detected")
	return cm.getDIFromMultiICSorDC(ctx, zoneLabel, regionLabel, zoneLooking, regionLooking)
}

func (cm *ConnectionManager) getDIFromSingleICS(ctx context.Context,
	zoneLabel string, regionLabel string, zoneLooking string, regionLooking string) (*ZoneDiscoveryInfo, error) {
	klog.V(4).Infof("getDIFromSingleICS called with zone: %s and region: %s", zoneLooking, regionLooking)

	if len(cm.ICSInstanceMap) != 1 {
		err := ErrUnsupportedConfiguration
		klog.Errorf("%v", err)
		return nil, err
	}

	var ics string

	// Get first inCloud Sphere Instance
	var tmpVsi *ICSInstance
	for _, tmpVsi = range cm.ICSInstanceMap {
		break //Grab the first one because there is only one
	}

	var err error
	for i := 0; i < NumConnectionAttempts; i++ {
		err = cm.Connect(ctx, tmpVsi)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(RetryAttemptDelaySecs) * time.Second)
	}

	//FIXME TODO.WANGYONGCHAO
	numOfDc, err := icslib.GetNumberOfDatacenters(ctx, tmpVsi.Conn)
	if err != nil {
		klog.Errorf("%v", err)
		return nil, err
	}

	// More than 1 DC in this ICS
	if numOfDc > 1 {
		klog.Info("Multi Datacenter configuration detected")
		return cm.getDIFromMultiICSorDC(ctx, zoneLabel, regionLabel, zoneLooking, regionLooking)
	}

	// We are sure this is single ICS and DC
	klog.Info("Single iCenter/Datacenter configuration detected")

	//FIXME TODO.WANGYONGCHAO
	datacenterObjs, err := icslib.GetAllDatacenter(ctx, tmpVsi.Conn)
	if err != nil {
		klog.Error("GetAllDatacenter failed. Err:", err)
		return nil, err
	}

	discoveryInfo := &ZoneDiscoveryInfo{
		IcsServer:   ics,
		DataCenter: datacenterObjs[0],
	}

	return discoveryInfo, nil
}

func (cm *ConnectionManager) getDIFromMultiICSorDC(ctx context.Context,
	zoneLabel string, regionLabel string, zoneLooking string, regionLooking string) (*ZoneDiscoveryInfo, error) {
	klog.V(4).Infof("getDIFromMultiICSorDC called with zone: %s and region: %s", zoneLooking, regionLooking)

	if len(zoneLabel) == 0 || len(regionLabel) == 0 || len(zoneLooking) == 0 || len(regionLooking) == 0 {
		err := ErrMultiICSRequiresZones
		klog.Errorf("%v", err)
		return nil, err
	}

	type zoneSearch struct {
		tenantRef  string
		ics         string
		//FIXME TODO.WANGYONGCHAO
		datacenter *icslib.Datacenter
		host       *icslib.HostSystem
	}

	var mutex = &sync.Mutex{}
	var globalErrMutex = &sync.Mutex{}
	var queueChannel chan *zoneSearch
	var wg sync.WaitGroup
	var globalErr *error

	queueChannel = make(chan *zoneSearch, QueueSize)

	zoneFound := false
	globalErr = nil

	setGlobalErr := func(err error) {
		globalErrMutex.Lock()
		globalErr = &err
		globalErrMutex.Unlock()
	}

	setZoneFound := func(found bool) {
		mutex.Lock()
		zoneFound = found
		mutex.Unlock()
	}

	getZoneFound := func() bool {
		mutex.Lock()
		found := zoneFound
		mutex.Unlock()
		return found
	}

	go func() {
		for _, vsi := range cm.ICSInstanceMap {
			//FIXME TODO.WANGYONGCHAO
			var datacenterObjs []*icslib.Datacenter

			if getZoneFound() {
				break
			}

			var err error
			for i := 0; i < NumConnectionAttempts; i++ {
				err = cm.Connect(ctx, vsi)
				if err == nil {
					break
				}
				time.Sleep(time.Duration(RetryAttemptDelaySecs) * time.Second)
			}

			if err != nil {
				klog.Error("getDIFromMultiICSorDC error ics:", err)
				setGlobalErr(err)
				continue
			}

			if vsi.Cfg.Datacenters == "" {
				//FIXME TODO.WANGYONGCHAO
				datacenterObjs, err = icslib.GetAllDatacenter(ctx, vsi.Conn)
				if err != nil {
					klog.Error("getDIFromMultiICSorDC error dc:", err)
					setGlobalErr(err)
					continue
				}
			} else {
				datacenters := strings.Split(vsi.Cfg.Datacenters, ",")
				for _, dc := range datacenters {
					dc = strings.TrimSpace(dc)
					if dc == "" {
						continue
					}
					//FIXME TODO.WANGYONGCHAO
					datacenterObj, err := icslib.GetDatacenter(ctx, vsi.Conn, dc)
					if err != nil {
						klog.Error("getDIFromMultiICSorDC error dc:", err)
						setGlobalErr(err)
						continue
					}
					datacenterObjs = append(datacenterObjs, datacenterObj)
				}
			}

			for _, datacenterObj := range datacenterObjs {
				if getZoneFound() {
					break
				}

				////FIXME TODO.WANGYONGCHAO
				//finder := find.NewFinder(datacenterObj.Client(), false)
				//finder.SetDatacenter(datacenterObj.Datacenter)
				//
				////FIXME TODO.WANGYONGCHAO
				//hostList, err := finder.HostSystemList(ctx, "*/*")
				//if err != nil {
				//	klog.Errorf("HostSystemList failed: %v", err)
				//	continue
				//}
				hostList, err := icslib.GetHostSystemListByDC(ctx, vsi.Conn, datacenterObj.ID)
				if err != nil {
					klog.Errorf("HostSystemList failed: %v", err)
					continue
				}

				for _, host := range hostList {
					klog.V(3).Infof("Finding zone in ics=%s and datacenter=%s for host: %s", vsi.Cfg.ICenterIP, datacenterObj.Name, host.HostName)
					queueChannel <- &zoneSearch{
						tenantRef:  vsi.Cfg.TenantRef,
						ics:         vsi.Cfg.ICenterIP,
						datacenter: datacenterObj,
						host:       host,
					}
				}
			}
		}
		close(queueChannel)
	}()

	var zoneInfo *ZoneDiscoveryInfo
	for i := 0; i < PoolSize; i++ {
		wg.Add(1)
		go func() {
			for res := range queueChannel {

				klog.V(3).Infof("Checking zones for host: %s", res.host.HostName)
				//FIXME TODO.WANGYONGCHAO
				result, err := cm.LookupZoneByMoref(ctx, res.tenantRef, res.host.ID, zoneLabel, regionLabel)
				if err != nil {
					klog.Errorf("Failed to find zone: %s and region: %s for host %s", zoneLabel, regionLabel, res.host.HostName)
					continue
				}

				if !strings.EqualFold(result[ZoneLabel], zoneLooking) ||
					!strings.EqualFold(result[RegionLabel], regionLooking) {
					klog.V(4).Infof("Does not match region: %s and zone: %s", result[RegionLabel], result[ZoneLabel])
					continue
				}

				klog.Infof("Found zone: %s and region: %s for host %s", zoneLooking, regionLooking, res.host.HostName)
				zoneInfo = &ZoneDiscoveryInfo{
					TenantRef:  res.tenantRef,
					IcsServer:   res.ics,
					DataCenter: res.datacenter,
				}

				setZoneFound(true)
				break
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if zoneFound {
		return zoneInfo, nil
	}
	if globalErr != nil {
		return nil, *globalErr
	}

	klog.V(4).Infof("getDIFromMultiICSorDC: zone: %s and region: %s not found", zoneLabel, regionLabel)
	return nil, icslib.ErrNoZoneRegionFound
}

//FIXME TODO.WANGYONGCHAO
func withTagsClient(ctx context.Context, connection *icsgo.ICSConnection, f func(c *rest.Client) error) error {
	c, err := connection.GetClient()
	if err != nil {
		return err
	}
	defer func() {
		if err := connection.Logout(ctx); err != nil {
			klog.Errorf("failed to logout: %v", err)
		}
	}()
	return f(c)
}

// LookupZoneByMoref searches for a zone using the provided managed object reference.
//FIXME TODO.WANGYONGCHAO
func (cm *ConnectionManager) LookupZoneByMoref(ctx context.Context, tenantRef string,
	hostRef string, zoneLabel string, regionLabel string) (map[string]string, error) {

	result := make(map[string]string)

	vsi := cm.ICSInstanceMap[tenantRef]
	if vsi == nil {
		err := ErrConnectionNotFound
		klog.Errorf("Unable to find Connection for tenantRef=%s", tenantRef)
		return nil, err
	}

	err := withTagsClient(ctx, vsi.Conn, func(c *rest.Client) error {
		client := tags.NewTagsService(c)

		hostService := ht.NewHostService(c)
		host, err := hostService.GetHost(ctx, hostRef)
		if err != nil {
			klog.Errorf("Ancestors failed for %s with err %v", hostRef, err)
			return err
		}
		objects := make(map[string]string, 10)
		objects["DATACENTER"] = host.DataCenterID
		objects["CLUSTER"] = host.ClusterID
		objects["HOST"] = host.ID

		// search the hierarchy, example order: ["Host", "Cluster", "Datacenter", "Folder"]
		for key, value := range objects {
			klog.V(4).Infof("Name: %s, Type: %s", value, key)
			tags, err := client.ListAttachedTags(ctx, key, value)
			if err != nil {
				klog.Errorf("Cannot list attached tags. Err: %v", err)
				return err
			}
			for _, value := range tags {
				tag, err := client.GetTag(ctx, value)
				if err != nil {
					klog.Errorf("Zones Get tag %s: %s", value, err)
					return err
				}

				found := func() {
					klog.V(2).Infof("Found %s tag attached to %s", tag.Name, hostRef)
				}
				switch {
				case tag.Description == zoneLabel:
					result[ZoneLabel] = tag.Name
					found()
				case tag.Description == regionLabel:
					result[RegionLabel] = tag.Name
					found()
				}

				if result[ZoneLabel] != "" && result[RegionLabel] != "" {
					return nil
				}
			}
		}

		if result[RegionLabel] == "" {
			if regionLabel != "" {
				return fmt.Errorf("inCloud Sphere region category %s does not match any tags for mo: %v", regionLabel, hostRef)
			}
		}
		if result[ZoneLabel] == "" {
			if zoneLabel != "" {
				return fmt.Errorf("inCloud Sphere zone category %s does not match any tags for mo: %v", zoneLabel, hostRef)
			}
		}

		return nil
	})
	if err != nil {
		klog.Errorf("Get zone for mo: %s: %s", hostRef, err)
		return nil, err
	}
	return result, nil
}
