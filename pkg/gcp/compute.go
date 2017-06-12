package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// Compute is a helper layer to wrap complex Google Compute API call logic.
type Compute struct {
	projectID string

	ctx    context.Context
	client *http.Client
}

// NewCompute returns a Google Cloud Compute client.
// Create/Download the key file from https://console.cloud.google.com/apis/credentials.
func NewCompute(ctx context.Context, scope string, key []byte) (*Compute, error) {
	// key must be JSON-format as {"project_id":...}
	credMap := make(map[string]string)
	if err := json.Unmarshal(key, &credMap); err != nil {
		return nil, fmt.Errorf("key has wrong format %q (%v)", string(key), err)
	}
	project, ok := credMap["project_id"]
	if !ok {
		return nil, fmt.Errorf("key has no project_id %q", string(key))
	}
	jwt, err := google.JWTConfigFromJSON(key, scope)
	if err != nil {
		return nil, err
	}
	return &Compute{projectID: project, ctx: ctx, client: jwt.Client(ctx)}, nil
}

// ListMachines lists virtual machines in a zone.
func (g *Compute) ListMachines(ctx context.Context, zone string) ([]*compute.Instance, error) {
	glog.Infof("listing machines in %q", zone)

	csrv, err := compute.New(g.client)
	if err != nil {
		return nil, err
	}
	l, err := csrv.Instances.
		List(g.projectID, zone).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	sort.Slice(l.Items, func(i, j int) bool { return l.Items[i].Name < l.Items[j].Name })
	return l.Items, nil
}

// CreateMacine creates a virtual machines in the zone.
// 'memory' is the number of GBs for RAM.
func (g *Compute) CreateMacine(ctx context.Context, cfg InstanceConfig) (st *compute.Instance, err error) {
	glog.Infof("creating %q", cfg.Name)

	csrv, err := compute.New(g.client)
	if err != nil {
		return nil, err
	}

	cfg.ctx = ctx
	cfg.csrv = csrv
	cfg.projectID = g.projectID
	cfg.expectStatus = "RUNNING"
	cfg.needDelete = false
	instanceToCreate := cfg.genInstance()

	_, err = csrv.Instances.
		Insert(g.projectID, cfg.Zone, &instanceToCreate).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	return cfg.watchStatus()
}

// SetMetadata sets metadata to the instance.
func (g *Compute) SetMetadata(ctx context.Context, cfg InstanceConfig) error {
	glog.Infof("setting %d metadata on %q", len(cfg.MetadataItems), cfg.Name)
	csrv, err := compute.New(g.client)
	if err != nil {
		return err
	}

	inst, err := csrv.Instances.Get(g.projectID, cfg.Zone, cfg.Name).Context(ctx).Do()
	if err != nil {
		return err
	}

	items := make([]*compute.MetadataItems, 0, len(cfg.MetadataItems))
	for k, v := range cfg.MetadataItems {
		// make sure to copy as value before passing as reference!
		copied := v
		items = append(items, &compute.MetadataItems{Key: k, Value: &copied})
	}
	metadata := &compute.Metadata{
		Items:       items,
		Fingerprint: inst.Metadata.Fingerprint,
	}

	var op *compute.Operation
	op, err = csrv.Instances.
		SetMetadata(g.projectID, cfg.Zone, cfg.Name, metadata).
		Context(ctx).
		Do()
	if err != nil {
		return err
	}

	// call is asynchronous; poll for the completion of op
	for {
		op, err = csrv.ZoneOperations.Get(g.projectID, cfg.Zone, op.Name).Context(ctx).Do()
		if err != nil {
			return err
		}
		if op.Status == "DONE" {
			break
		}
		time.Sleep(1 * time.Second)
	}

	glog.Infof("finished setting %d metadata on %q", len(cfg.MetadataItems), cfg.Name)
	return err
}

// StopMachine stops a virtual machines in the zone.
func (g *Compute) StopMachine(ctx context.Context, cfg InstanceConfig) (st *compute.Instance, err error) {
	glog.Infof("stopping %q", cfg.Name)

	csrv, err := compute.New(g.client)
	if err != nil {
		return nil, err
	}

	_, err = csrv.Instances.
		Stop(g.projectID, cfg.Zone, cfg.Name).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	cfg.ctx = ctx
	cfg.csrv = csrv
	cfg.projectID = g.projectID
	cfg.expectStatus = "TERMINATED"
	cfg.needDelete = false
	return cfg.watchStatus()
}

// StartMachine starts a virtual machines in the zone.
func (g *Compute) StartMachine(ctx context.Context, cfg InstanceConfig) (st *compute.Instance, err error) {
	glog.Infof("starting %q", cfg.Name)

	csrv, err := compute.New(g.client)
	if err != nil {
		return nil, err
	}

	_, err = csrv.Instances.
		Start(g.projectID, cfg.Zone, cfg.Name).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	cfg.ctx = ctx
	cfg.csrv = csrv
	cfg.projectID = g.projectID
	cfg.expectStatus = "RUNNING"
	cfg.needDelete = false
	return cfg.watchStatus()
}

// DeleteMachine deletes a virtual machines in the zone.
func (g *Compute) DeleteMachine(ctx context.Context, cfg InstanceConfig) (st *compute.Instance, err error) {
	glog.Infof("deleting %q", cfg.Name)

	csrv, err := compute.New(g.client)
	if err != nil {
		return nil, err
	}
	_, err = csrv.Instances.
		Delete(g.projectID, cfg.Zone, cfg.Name).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	cfg.ctx = ctx
	cfg.csrv = csrv
	cfg.projectID = g.projectID
	cfg.expectStatus = "TERMINATED"
	cfg.needDelete = true
	return cfg.watchStatus()
}

// Machine represents a virtual machine in Google Compute Engine.
type Machine struct {
	Created            string
	Name               string
	ID                 string
	Zone               string
	InternalIP         string
	ExternalIP         string
	Status             string
	SourceImageLicense string
	SourceImage        string
}

// ConvertToMachine converts to Machine (human-readable).
func ConvertToMachine(v compute.Instance) Machine {
	machine := Machine{
		Created: v.CreationTimestamp,
		Name:    v.Name,
		ID:      fmt.Sprint(v.Id),
		Zone:    strings.Replace(v.Zone, ComputeVersion, "", 1),
		Status:  v.Status,
	}
	if len(v.NetworkInterfaces) != 0 {
		machine.InternalIP = v.NetworkInterfaces[0].NetworkIP
		if len(v.NetworkInterfaces[0].AccessConfigs) != 0 {
			machine.ExternalIP = v.NetworkInterfaces[0].AccessConfigs[0].NatIP
		}
	}
	if len(v.Disks) != 0 {
		machine.SourceImageLicense = strings.Replace(v.Disks[0].Licenses[0], ComputeVersion, "", 1)
		if v.Disks[0].InitializeParams != nil {
			machine.SourceImage = v.Disks[0].InitializeParams.SourceImage
		}
	}
	return machine
}

// InstanceConfig represents GCP Compute Engine VM config.
type InstanceConfig struct {
	ctx  context.Context
	csrv *compute.Service

	projectID string
	Zone      string
	Name      string

	expectStatus string
	needDelete   bool

	OS         string
	CPU        int
	Memory     int
	DiskSizeGB int

	// OnHostMaintenance is either MIGRATE or TERMINATE.
	// If you do not want your instance to live migrate, you can
	// choose to terminate and optionally restart your instance.
	// With this option, Google Compute Engine will signal your
	// instance to shut down, wait for a short period of time
	// for your instance to shut down cleanly, terminate the instance,
	// and restart it away from the maintenance event.
	// https://cloud.google.com/compute/docs/instances/setting-instance-scheduling-options
	OnHostMaintenance string

	// MetadataItems maps metadata key to its value
	// (e.g. set Ignition configuration value with key 'user-data' for Container Linux).
	MetadataItems map[string]string

	Tags []string
}

const statusPollInterval = 5 * time.Second

// Possible values:
//   "PROVISIONING"
//   "RUNNING"
//   "STAGING"
//   "STOPPED"
//   "STOPPING"
//   "SUSPENDED"
//   "SUSPENDING"
//   "TERMINATED"
func (c *InstanceConfig) watchStatus() (st *compute.Instance, err error) {
	tick := time.NewTicker(statusPollInterval)
	defer tick.Stop()

	now := time.Now()

	var list *compute.InstanceList
	for {
		select {
		case <-c.ctx.Done():
			return nil, c.ctx.Err()
		case <-tick.C:
		}

		list, err = c.csrv.Instances.
			List(c.projectID, c.Zone).
			Filter(fmt.Sprintf("name eq %s", c.Name)).
			Do()
		if err != nil {
			return nil, err
		}

		if len(list.Items) != 1 {
			if c.needDelete {
				glog.Infof("%q(%s) does not exist or got deleted (took %v)", c.Name, c.Zone, time.Since(now))
				return nil, nil
			}
			return nil, fmt.Errorf("cannot find %q in %q", c.Name, c.Zone)
		}
		st = list.Items[0]

		glog.Infof("%q(%s) %q (want %q, deleting '%v', taking %v)", c.Name, c.Zone, st.Status, c.expectStatus, c.needDelete, time.Since(now))
		if st.Status == c.expectStatus {
			return st, nil
		}
	}
}

// ComputeVersion is the API version of Google Cloud Compute.
const ComputeVersion = "https://www.googleapis.com/compute/v1"

func (c *InstanceConfig) genInstance() (instance compute.Instance) {
	bv := true
	instance = compute.Instance{
		Name:        c.Name,
		Zone:        getComputeZone(c.projectID, c.Zone),
		MachineType: getComputeMachineType(c.projectID, c.Zone, c.CPU, c.Memory),
		Scheduling: &compute.Scheduling{
			AutomaticRestart:  &bv,
			OnHostMaintenance: c.OnHostMaintenance, // GPU must be OnHostMaintenance: "TERMINATE"
			Preemptible:       false,
		},
		Disks: []*compute.AttachedDisk{
			{
				DeviceName: c.Name,
				AutoDelete: true,
				Boot:       true,
				Interface:  "SCSI",
				Kind:       "compute#attachedDisk",
				Licenses:   []string{getLicense(c.OS)},
				Mode:       "READ_WRITE",
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    c.Name,
					DiskType:    getComputeDiskType(c.projectID, c.Zone, "pd-ssd"),
					DiskSizeGb:  int64(c.DiskSizeGB),
					SourceImage: getSourceImage(c.OS),
				},
			},
		},
		CanIpForward: false,
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Name:    "nic0",
				Network: getComputeNetwork(c.projectID),
				AccessConfigs: []*compute.AccessConfig{
					// NatIP: An external IP address associated with this instance. Specify
					// an unused static external IP address available to the project or
					// leave this field undefined to use an IP from a shared ephemeral IP
					// address pool. If you specify a static external IP address, it must
					// live in the same region as the zone of the instance.
					{
						Name:  "External NAT",
						Type:  "ONE_TO_ONE_NAT",
						NatIP: "",
					},
				},
			},
		},
		// e.g. http-server,https-server
		Tags: &compute.Tags{Items: c.Tags},
	}
	if len(c.MetadataItems) != 0 {
		items := make([]*compute.MetadataItems, 0, len(c.MetadataItems))
		for k, v := range c.MetadataItems {
			// make sure to copy as value before passing as reference!
			copied := v
			items = append(items, &compute.MetadataItems{Key: k, Value: &copied})
		}
		instance.Metadata = &compute.Metadata{Items: items}
	}
	return instance
}

// https://www.googleapis.com/compute/v1/projects/etcd-development/zones/us-west1-a/disks/deep
func getComputeDiskSource(project, zone, name string) string {
	return fmt.Sprintf("%s/projects/%s/zones/%s/disks/%s", ComputeVersion, project, zone, name)
}

// https://www.googleapis.com/compute/v1/projects/etcd-development/zones/us-west1-a
func getComputeZone(project, zone string) string {
	return fmt.Sprintf("%s/projects/%s/zones/%s", ComputeVersion, project, zone)
}

// https://www.googleapis.com/compute/v1/projects/etcd-development/zones/us-west1-a/machineTypes/n1-standard-4
// zones/zone/machineTypes/custom-CPUS-MEMORY
func getComputeMachineType(project, zone string, cpu, memory int) string {
	mv := memory * 4 * 256 // convert GB to MB
	return fmt.Sprintf("%s/projects/%s/zones/%s/machineTypes/custom-%d-%d", ComputeVersion, project, zone, cpu, mv)
}

// https://www.googleapis.com/compute/v1/projects/etcd-development/zones/us-west1-a/diskTypes/pd-ssd
func getComputeDiskType(project, zone, dtype string) string {
	return fmt.Sprintf("%s/projects/%s/zones/%s/diskTypes/%s", ComputeVersion, project, zone, dtype)
}

// https://www.googleapis.com/compute/v1/projects/etcd-development/global/networks/default
func getComputeNetwork(project string) string {
	return fmt.Sprintf("%s/projects/%s/global/networks/default", ComputeVersion, project)
}

// https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/licenses/ubuntu-1604-xenial
func getLicense(os string) string {
	switch os {
	case "ubuntu":
		return fmt.Sprintf("%s/projects/ubuntu-os-cloud/global/licenses/ubuntu-1604-xenial", ComputeVersion)
	case "container-linux":
		return fmt.Sprintf("%s/projects/coreos-cloud/global/licenses/coreos-alpha", ComputeVersion)
	}
	return ""
}

// https://cloud.google.com/compute/docs/images#os-compute-support
// projects/coreos-cloud/global/images/family/coreos-alpha
func getSourceImage(os string) string {
	switch os {
	case "ubuntu":
		return "projects/ubuntu-os-cloud/global/images/family/ubuntu-1604-lts"
	case "container-linux":
		return "projects/coreos-cloud/global/images/family/coreos-alpha"
	}
	return ""
}
