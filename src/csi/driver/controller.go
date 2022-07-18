/*
Copyright 2017 The Kubernetes Authors.

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

package driver

import (
	"fmt"
	"path"

	"github.com/drycc/storage/src/csi/client"
	"github.com/drycc/storage/src/csi/provider"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

const (
	bucketKey            = "bucket"
	maxAvailableCapacity = 1 * 1024 * 1024 * 1024 * 1024 * 1024
)

var (
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		csi.ControllerServiceCapability_RPC_GET_CAPACITY,
	}
)

type ControllerServer struct {
	provider provider.Provider
	driver   *CSIDriver
}

func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	params := req.GetParameters()
	capacityBytes := int64(req.GetCapacityRange().GetRequiredBytes())
	volumeID := sanitizeVolumeID(req.GetName())
	bucketName := volumeID
	prefix := ""

	// check if bucket name is overridden
	if params[bucketKey] != "" {
		bucketName = params[bucketKey]
		prefix = volumeID
		volumeID = path.Join(bucketName, prefix)
	}

	if err := cs.driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid create volume req: %v", req)
		return nil, err
	}

	// Check arguments
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "name missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "volume Capabilities missing in request")
	}

	glog.V(4).Infof("got a request to create volume %s", volumeID)
	client, err := client.NewClientFromSecret(req.GetSecrets())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %s", err)
	}

	exists, err := client.BucketExists(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket %s exists: %v", volumeID, err)
	}

	if !exists {
		if err = client.CreateBucket(bucketName); err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %v", bucketName, err)
		}
	}

	if err = client.CreatePrefix(bucketName, prefix); err != nil {
		return nil, fmt.Errorf("failed to create prefix %s: %v", prefix, err)
	}

	glog.V(4).Infof("create volume %s", volumeID)
	// DeleteVolume lacks VolumeContext, but publish&unpublish requests have it,
	// so we don't need to store additional metadata anywhere
	context := make(map[string]string)
	for k, v := range params {
		context[k] = v
	}
	context["capacity"] = fmt.Sprintf("%v", capacityBytes)
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: capacityBytes,
			VolumeContext: context,
		},
	}, nil
}

func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	bucketName, prefix := volumeIDToBucketPrefix(volumeID)

	// Check arguments
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	if err := cs.driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid delete volume req: %v", req)
		return nil, err
	}
	glog.V(4).Infof("deleting volume %s", volumeID)

	client, err := client.NewClientFromSecret(req.GetSecrets())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %s", err)
	}

	var deleteErr error
	if prefix == "" {
		// prefix is empty, we delete the whole bucket
		if err := client.RemoveBucket(bucketName); err != nil && err.Error() != "The specified bucket does not exist" {
			deleteErr = err
		}
		glog.V(4).Infof("bucket %s removed", bucketName)
	} else {
		if err := client.RemovePrefix(bucketName, prefix); err != nil {
			deleteErr = fmt.Errorf("unable to remove prefix: %w", err)
		}
		glog.V(4).Infof("prefix %s removed", prefix)
	}

	if deleteErr != nil {
		return nil, deleteErr
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}
	bucketName, _ := volumeIDToBucketPrefix(req.GetVolumeId())

	client, err := client.NewClientFromSecret(req.GetSecrets())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %s", err)
	}
	exists, err := client.BucketExists(bucketName)
	if err != nil {
		return nil, err
	}

	if !exists {
		// return an error if the bucket of the requested volume does not exist
		return nil, status.Error(codes.NotFound, fmt.Sprintf("bucket of volume with id %s does not exist", req.GetVolumeId()))
	}

	supportedAccessMode := &csi.VolumeCapability_AccessMode{
		Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	}

	for _, capability := range req.VolumeCapabilities {
		if capability.GetAccessMode().GetMode() != supportedAccessMode.GetMode() {
			return &csi.ValidateVolumeCapabilitiesResponse{Message: "Only single node writer is supported"}, nil

		}
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessMode: supportedAccessMode,
				},
			},
		},
	}, nil
}

func (cs *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(6).Infof("ControllerGetCapabilities: called with args %#v", req)
	var caps []*csi.ControllerServiceCapability
	for _, cap := range controllerCaps {
		c := &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}

// ControllerExpandVolume
func (cs *ControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	volSizeBytes := req.GetCapacityRange().GetRequiredBytes()
	volumeID := req.GetVolumeId()
	bucket, prefix := volumeIDToBucketPrefix(volumeID)

	if err := cs.provider.ControllerExpandVolume(bucket, prefix, uint64(volSizeBytes), req.GetSecrets()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.ControllerExpandVolumeResponse{CapacityBytes: volSizeBytes, NodeExpansionRequired: false}, nil
}

// GetCapacity
func (cs *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return &csi.GetCapacityResponse{
		AvailableCapacity: maxAvailableCapacity,
	}, nil
}

// ControllerGetVolume unimplemented
func (cs *ControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ControllerGetVolume not implemented")
}

// ControllerPublishVolume unimplemented
func (cs *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerUnpublishVolume unimplemented
func (cs *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// CreateSnapshot unimplemented
func (cs *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// DeleteSnapshot unimplemented
func (cs *ControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListSnapshots unimplemented
func (cs *ControllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListVolumes unimplemented
func (cs *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
