package server

import (
	"context"
	"time"

	"k8s.io/klog"

	"google.golang.org/grpc"

	pb "github.com/inspur-ics/cloud-provider-ics/pkg/cloudprovider/ics/proto"
	vcfg "github.com/inspur-ics/cloud-provider-vsphere/pkg/common/config"
)

// NewIcsCloudProviderClient creates CloudProviderIcsClient
func NewIcsCloudProviderClient(ctx context.Context) (pb.CloudProviderIcsClient, error) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < RetryAttempts; i++ {
		conn, err = grpc.Dial(vcfg.DefaultAPIBinding, grpc.WithInsecure())
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		klog.Errorf("did not connect: %v", err)
		return nil, err
	}

	c := pb.NewCloudProviderIcsClient(conn)

	return c, nil
}
