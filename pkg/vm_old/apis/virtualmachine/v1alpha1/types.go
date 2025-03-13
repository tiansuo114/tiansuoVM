package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachine is the Schema for the virtualmachines API
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec"`
	Status VirtualMachineStatus `json:"status"`
}

// VirtualMachineSpec defines the desired state of VirtualMachine
type VirtualMachineSpec struct {
	// Size of the virtual machine CPU in cores
	CPU int32 `json:"cpu"`

	// Size of the virtual machine memory in MB
	MemoryMB int32 `json:"memoryMB"`

	// Size of the virtual machine disk in GB
	DiskGB int32 `json:"diskGB"`

	// Name of the image to use for this virtual machine
	ImageName string `json:"imageName"`

	// Action to perform on the virtual machine
	Action VMAction `json:"action,omitempty"`

	// SSH public keys to inject into the virtual machine
	SSHKeys []string `json:"sshKeys,omitempty"`

	// NodeName is the name of the node where the virtual machine should run
	// +optional
	NodeName string `json:"nodeName,omitempty"`

	// Network configuration for the virtual machine
	Network NetworkConfig `json:"network,omitempty"`

	// Storage configuration for the virtual machine
	Storage StorageConfig `json:"storage,omitempty"`
}

// VMAction defines actions that can be performed on a virtual machine
type VMAction string

const (
	// VMActionStart starts the virtual machine
	VMActionStart VMAction = "start"
	// VMActionStop stops the virtual machine
	VMActionStop VMAction = "stop"
	// VMActionRestart restarts the virtual machine
	VMActionRestart VMAction = "restart"
	// VMActionPause pauses the virtual machine
	VMActionPause VMAction = "pause"
	// VMActionResume resumes the virtual machine
	VMActionResume VMAction = "resume"
)

// NetworkConfig defines network configuration for a virtual machine
type NetworkConfig struct {
	// Type of network to use
	Type NetworkType `json:"type"`

	// Whether to request a public IP
	PublicIP bool `json:"publicIP,omitempty"`
}

// NetworkType defines the type of network for a virtual machine
type NetworkType string

const (
	// NetworkTypeDefault uses the default network configuration
	NetworkTypeDefault NetworkType = "default"
	// NetworkTypeHost uses host network
	NetworkTypeHost NetworkType = "host"
)

// StorageConfig defines storage configuration for a virtual machine
type StorageConfig struct {
	// Type of storage to use
	Type StorageType `json:"type"`

	// Storage class name to use for persistent volume claims
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
}

// StorageType defines the type of storage for a virtual machine
type StorageType string

const (
	// StorageTypeEmptyDir uses emptyDir volume
	StorageTypeEmptyDir StorageType = "emptyDir"
	// StorageTypePVC uses PersistentVolumeClaim
	StorageTypePVC StorageType = "pvc"
)

// VMState defines the observed state of a virtual machine
type VMState string

const (
	// VMStatePending indicates the virtual machine is being created
	VMStatePending VMState = "Pending"
	// VMStateProvisioning indicates the virtual machine is being provisioned
	VMStateProvisioning VMState = "Provisioning"
	// VMStateRunning indicates the virtual machine is running
	VMStateRunning VMState = "Running"
	// VMStateStopped indicates the virtual machine is stopped
	VMStateStopped VMState = "Stopped"
	// VMStateStopping indicates the virtual machine is stopping
	VMStateStopping VMState = "Stopping"
	// VMStateStarting indicates the virtual machine is starting
	VMStateStarting VMState = "Starting"
	// VMStateRestarting indicates the virtual machine is restarting
	VMStateRestarting VMState = "Restarting"
	// VMStatePaused indicates the virtual machine is paused
	VMStatePaused VMState = "Paused"
	// VMStatePausing indicates the virtual machine is being paused
	VMStatePausing VMState = "Pausing"
	// VMStateResuming indicates the virtual machine is being resumed
	VMStateResuming VMState = "Resuming"
	// VMStateTerminating indicates the virtual machine is being deleted
	VMStateTerminating VMState = "Terminating"
	// VMStateFailed indicates the virtual machine is in a failed state
	VMStateFailed VMState = "Failed"
	// VMStateUnknown indicates the virtual machine state is unknown
	VMStateUnknown VMState = "Unknown"
)

// VirtualMachineStatus defines the observed state of VirtualMachine
type VirtualMachineStatus struct {
	// Current state of the virtual machine
	State VMState `json:"state"`

	// IP address assigned to the virtual machine
	IP string `json:"ip,omitempty"`

	// Node name where the virtual machine is running
	NodeName string `json:"nodeName,omitempty"`

	// Access endpoints for the virtual machine (SSH, VNC, etc.)
	Endpoints []VMEndpoint `json:"endpoints,omitempty"`

	// Time when the virtual machine was created
	CreationTime metav1.Time `json:"creationTime,omitempty"`

	// Time when the virtual machine was last updated
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// Message provides more details about the current state
	Message string `json:"message,omitempty"`

	// Reason provides the reason for the current state
	Reason string `json:"reason,omitempty"`
}

// VMEndpoint represents an endpoint to access the virtual machine
type VMEndpoint struct {
	// Type of endpoint
	Type VMEndpointType `json:"type"`

	// Address of the endpoint
	Address string `json:"address"`

	// Port of the endpoint
	Port int32 `json:"port"`
}

// VMEndpointType defines the type of endpoint
type VMEndpointType string

const (
	// VMEndpointTypeSSH is an SSH endpoint
	VMEndpointTypeSSH VMEndpointType = "SSH"
	// VMEndpointTypeVNC is a VNC endpoint
	VMEndpointTypeVNC VMEndpointType = "VNC"
	// VMEndpointTypeHTTP is an HTTP endpoint
	VMEndpointTypeHTTP VMEndpointType = "HTTP"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineList contains a list of VirtualMachine
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}
