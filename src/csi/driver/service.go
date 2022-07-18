package driver

import (
	"log"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/drycc/storage/src/csi/provider"
	"github.com/golang/glog"
)

type DriveService struct {
	driver   *CSIDriver
	provider provider.Provider
	endpoint string

	ids *IdentityServer
	ns  *NodeServer
	cs  *ControllerServer
}

// New initializes the driver
func New(nodeID, providerString, endpoint string) (*DriveService, error) {
	provider, err := provider.GetProvider(providerString)
	if err != nil {
		return nil, err
	}
	if err := provider.ParseFlag(); err != nil {
		return nil, err
	}
	d := NewCSIDriver(driverName, vendorVersion, nodeID)
	if d == nil {
		glog.Fatalln("failed to initialize CSI Driver.")
	}

	service := &DriveService{
		driver:   d,
		provider: provider,
		endpoint: endpoint,
	}
	return service, nil
}

func (service *DriveService) newIdentityServer(d *CSIDriver) *IdentityServer {
	return &IdentityServer{
		driver: d,
	}
}

func (service *DriveService) newControllerServer(d *CSIDriver) (*ControllerServer, error) {
	return &ControllerServer{
		provider: service.provider,
		driver:   d,
	}, nil
}

func (service *DriveService) newNodeServer(driver *CSIDriver) (*NodeServer, error) {
	return &NodeServer{
		provider: service.provider,
		driver:   driver,
	}, nil
}

func (service *DriveService) Run() {
	glog.Infof("driver: %v ", driverName)
	glog.Infof("version: %v ", vendorVersion)
	// Initialize default library driver

	service.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME})
	service.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER})

	// Create GRPC servers
	service.ids = service.newIdentityServer(service.driver)
	if nodeServer, err := service.newNodeServer(service.driver); err != nil {
		log.Fatal(err)
	} else {
		service.ns = nodeServer
	}

	if controllerServer, err := service.newControllerServer(service.driver); err != nil {
		log.Fatal(err)
	} else {
		service.cs = controllerServer
	}

	s := NewNonBlockingGRPCServer()
	s.Start(service.endpoint, service.ids, service.cs, service.ns)
	s.Wait()
}
