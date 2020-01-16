package icslib

import (
    "context"
    icsgo "github.com/inspur-ics/ics-go-sdk"
    "github.com/inspur-ics/ics-go-sdk/client/types"
    ht "github.com/inspur-ics/ics-go-sdk/host"
    "k8s.io/klog"
)

type HostSystem struct {
    Common
    *types.Host
}

// GetDatacenter returns the DataCenter Object for the given datacenterPath
// If datacenter is located in a folder, include full path to datacenter else just provide the datacenter name
func GetHostSystemListByDC(ctx context.Context, connection *icsgo.ICSConnection, datacenterPath string) ([]*HostSystem, error) {
    var hostSystems []*HostSystem
    client, err := connection.GetClient()
    if err != nil {
        return nil, err
    }
    hostService := ht.NewHostService(client)
    hosts, err := hostService.GetHostListByDC(ctx, datacenterPath)
    if err != nil {
        klog.Errorf("Failed to find the datacenter. err: %+v", err)
        return nil, err
    }
    for _, host := range hosts {
        hostSystems = append(hostSystems, &(HostSystem{Common{connection}, host}))
    }

    return hostSystems, nil
}