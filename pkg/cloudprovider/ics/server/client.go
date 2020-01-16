package server

import (
	"context"
	"time"

	"k8s.io/klog"

	"google.golang.org/grpc"

	pb "github.com/inspur-ics/cloud-provider-ics/pkg/cloudprovider/ics/proto"
	icfg "github.com/inspur-ics/cloud-provider-ics/pkg/common/config"
)

// NewICSCloudProviderClient creates CloudProviderICSClient
func NewICSCloudProviderClient(ctx context.Context) (pb.CloudProviderICSClient, error) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < RetryAttempts; i++ {
		conn, err = grpc.Dial(icfg.DefaultAPIBinding, grpc.WithInsecure())
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		klog.Errorf("did not connect: %v", err)
		return nil, err
	}

	c := pb.NewCloudProviderICSClient(conn)

	return c, nil
}
