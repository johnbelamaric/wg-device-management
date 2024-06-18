package api

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourcePool represents a collection of devices managed by a given driver. How
// devices are divided into pools is driver-specific, but typically the
// expectation would a be a pool per identical collection of devices, per node.
// It is fine to have more than one pool for a given node, for the same driver.
//
// Where a device gets published may change over time. The unique identifier
// for a device is the tuple `<driver name>/<node name>/<device name>`. Each
// of these names is a DNS label or domain, so it is okay to concatenate them
// like this in a string with a slash as separator.
//
// Consumers should be prepared to handle situations where the same device is
// listed in different pools, for example because the producer already added it
// to a new pool before removing it from an old one. Should this occur, then
// there is still only one such device instance. If the two device definitions
// disagree in any way, the one found in the newest ResourcePool, as determined
// by creationTimestamp, is preferred.
type ResourcePool struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec ResourcePoolSpec `json:"spec"`

	// Stretch goal for 1.31: define status
	//
	// To discuss:
	// - Who writes that status?
	//   After https://github.com/kubernetes/kubernetes/pull/125163 (implements
	//   https://github.com/kubernetes/enhancements/pull/4667), kubelet is not
	//   involved with publishing ResourcePools, they come directly from the driver.
	// - What information should be in it?
	// - Does it need to be a sub-resource? This would make it harder
	//   for a driver to publish a new device (spec) and its corresponding
	//   health information (status).
}

type ResourcePoolSpec struct {
	// NodeName identifies the node which provides the devices. All devices
	// are local to that node.
	//
	// This is currently required, but this might get relaxed in the future.
	NodeName string `json:"nodeName"`

	// POTENTIAL FUTURE EXTENSION: NodeSelector *v1.NodeSelector

	// DriverName identifies the DRA driver providing the capacity information.
	// A field selector can be used to list only ResourcePool
	// objects with a certain driver name.
	//
	// Must be a DNS subdomain and should end with a DNS domain owned by the
	// vendor of the driver.
	DriverName string `json:"driverName" protobuf:"bytes,3,name=driverName"`

	// CommonData represents information that is common across various devices,
	// to reduce redundancy in the devices list.
	//
	// +optional
	CommonData []CommonDeviceData `json:"commonData,omitempty"`

	// Devices lists all available devices in this pool.
	//
	// Must not have more than 128 entries.
	Devices []Device `json:"devices,omitempty"`

	// FUTURE EXTENSION: some other kind of list, should we ever need it.
	// Old clients seeing an empty Devices field can safely ignore the (to
	// them) empty pool.
}

const ResourcePoolMaxCommonData = 16
const ResourcePoolMaxDevices = 128

type CommonDeviceData struct {
	// Name identifies the particular set of common data represented
	// here.
	//
	// +required
	Name string `json:"name"`

	// Exactly one must be populated
	Attributes []DeviceAttribute
	Capacity   []DeviceCapacity

	// Future possible additions:
	// - SharedCapacity for use in allocating partitionable devices
	// - DeviceShape for encapsulating attributes, capacities, partion schemes
	// - DeviceShapeRef for storing those in a secondary object
}

// Device represents one individual hardware instance that can be selected based
// on its attributes.
type Device struct {
	// Name is unique identifier among all devices managed by
	// the driver on the node. It must be a DNS label.
	Name string `json:"name" protobuf:"bytes,1,name=name"`

	// CommonData is the list of names of entries from the common data that
	// should be included in this device.
	//
	// +listType=atomic
	// +optional
	//
	CommonData []string `json:"commonData,omitempty"`

	// Attributes defines the attributes of this device.
	// The name of each attribute must be unique.
	//
	// Must not have more than 32 entries.
	//
	// Values in this list whose name conflict with common attributes
	// will override the common attribute values.
	//
	// +listType=atomic
	// +optional
	//
	Attributes []DeviceAttribute `json:"attributes,omitempty" protobuf:"bytes,3,opt,name=attributes"`

	// Capacity defines the capacity values for this device.
	// The name of each capacity must be unique.
	//
	// Must not have more than 32 entries.
	//
	// Values in this list whose name conflict with common capacity
	// will override the common capacity values.
	//
	// +listType=atomic
	// +optional
	//
	Capacity []DeviceCapacity `json:"capacity,omitempty"`
}

const ResourcePoolMaxAttributesPerDevice = 32
const ResourcePoolMaxCapacityPerDevice = 32

// ResourcePoolMaxDevices and ResourcePoolMaxAttributesPerDevice where chosen
// so that with the maximum attribute length of 96 characters the total size of
// the ResourcePool object is around 420KB.

// DeviceAttribute is a combination of an attribute name and its value.
// Exactly one value must be set.
type DeviceAttribute struct {
	// Name is a unique identifier for this attribute, which will be
	// referenced when selecting devices.
	//
	// Attributes are defined either by the owner of the specific driver
	// (usually the vendor) or by some 3rd party (e.g. the Kubernetes
	// project). Because attributes are sometimes compared across devices,
	// a given name is expected to mean the same thing and have the same
	// type on all devices.
	//
	// Attribute names must be either a C-style identifier
	// (e.g. "the_name") or a DNS subdomain followed by a slash ("/")
	// followed by a C-style identifier
	// (e.g. "example.com/the_name"). Attributes whose name does not
	// include the domain prefix are assumed to be part of the driver's
	// domain. Attributes defined by 3rd parties must include the domain
	// prefix.
	//
	// The maximum length for the DNS subdomain is 63 characters (same as
	// for driver names) and the maximum length of the C-style identifier
	// is 32.
	Name string `json:"name" protobuf:"bytes,1,name=name"`

	// The Go field names below have a Value suffix to avoid a conflict between the
	// field "String" and the corresponding method. That method is required.
	// The Kubernetes API is defined without that suffix to keep it more natural.

	// QuantityValue is a quantity.
	QuantityValue *resource.Quantity `json:"quantity,omitempty" protobuf:"bytes,2,opt,name=quantity"`
	// BoolValue is a true/false value.
	BoolValue *bool `json:"bool,omitempty" protobuf:"bytes,3,opt,name=bool"`
	// StringValue is a string. Must not be longer than 64 characters.
	StringValue *string `json:"string,omitempty" protobuf:"bytes,4,opt,name=string"`
	// VersionValue is a semantic version according to semver.org spec 2.0.0.
	// Must not be longer than 64 characters.
	VersionValue *string `json:"version,omitempty" protobuf:"bytes,5,opt,name=version"`
}

type DeviceCapacity struct {
	// Name is a unique identifier among all capacities managed by the
	// driver in the pool.
	//
	// +required
	Name string `json:"name"`

	// Capacity is the total capacity of the named resource.
	//
	// +required
	Capacity resource.Quantity `json:"capacity"`
}

// DeviceAttributeMaxIDLength is the maximum length of the identifier in a device attribute name (`<domain>/<ID>`).
const DeviceAttributeMaxIDLength = 32

// DeviceAttributeMaxValueLength is the maximum length of a string or version attribute value.
const DeviceAttributeMaxValueLength = 64

// DeviceCapacityMaxNameLength is the maximum length of a shared capacity name.
const DeviceCapacityMaxNameLength = DeviceAttributeMaxIDLength
