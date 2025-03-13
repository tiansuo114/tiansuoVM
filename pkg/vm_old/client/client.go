package client

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	vmapi "tiansuoVM/pkg/vm_old/apis/virtualmachine/v1alpha1"
)

// VMClient is a client for VirtualMachine resources
type VMClient struct {
	restClient rest.Interface
	namespace  string
}

// NewVMClient creates a new client for VirtualMachine resources
func NewVMClient(cfg *rest.Config, namespace string) (*VMClient, error) {
	// Register our types with the Scheme so the client can encode and decode them
	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(
				schema.GroupVersion{
					Group:   vmapi.GroupName,
					Version: vmapi.Version,
				},
				&vmapi.VirtualMachine{},
				&vmapi.VirtualMachineList{},
			)
			metav1.AddToGroupVersion(scheme, schema.GroupVersion{
				Group:   vmapi.GroupName,
				Version: vmapi.Version,
			})
			return nil
		})

	// Create a new Scheme and register our types with it
	vmScheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(vmScheme)
	if err != nil {
		return nil, err
	}

	// Create a new config and set the scheme and codec
	config := *cfg
	config.GroupVersion = &schema.GroupVersion{
		Group:   vmapi.GroupName,
		Version: vmapi.Version,
	}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(vmScheme)
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	// Create a RESTClient for the given config
	restClient, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	// Return a client for our types
	return &VMClient{restClient: restClient, namespace: namespace}, nil
}

// Create creates a new VirtualMachine
func (c *VMClient) Create(ctx context.Context, vm *vmapi.VirtualMachine) (*vmapi.VirtualMachine, error) {
	result := &vmapi.VirtualMachine{}
	err := c.restClient.
		Post().
		Namespace(c.namespace).
		Resource("virtualmachines").
		Body(vm).
		Do(ctx).
		Into(result)
	return result, err
}

// Update updates an existing VirtualMachine
func (c *VMClient) Update(ctx context.Context, vm *vmapi.VirtualMachine) (*vmapi.VirtualMachine, error) {
	result := &vmapi.VirtualMachine{}
	err := c.restClient.
		Put().
		Namespace(c.namespace).
		Resource("virtualmachines").
		Name(vm.Name).
		Body(vm).
		Do(ctx).
		Into(result)
	return result, err
}

// Delete deletes a VirtualMachine
func (c *VMClient) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	return c.restClient.
		Delete().
		Namespace(c.namespace).
		Resource("virtualmachines").
		Name(name).
		Body(&options).
		Do(ctx).
		Error()
}

// Get gets a VirtualMachine by name
func (c *VMClient) Get(ctx context.Context, name string, options metav1.GetOptions) (*vmapi.VirtualMachine, error) {
	result := &vmapi.VirtualMachine{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("virtualmachines").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

// List lists all VirtualMachines in the namespace
func (c *VMClient) List(ctx context.Context, opts metav1.ListOptions) (*vmapi.VirtualMachineList, error) {
	result := &vmapi.VirtualMachineList{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("virtualmachines").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

// StartVM sends a start action to the VM
func (c *VMClient) StartVM(ctx context.Context, name string) (*vmapi.VirtualMachine, error) {
	return c.updateVMAction(ctx, name, vmapi.VMActionStart)
}

// StopVM sends a stop action to the VM
func (c *VMClient) StopVM(ctx context.Context, name string) (*vmapi.VirtualMachine, error) {
	return c.updateVMAction(ctx, name, vmapi.VMActionStop)
}

// RestartVM sends a restart action to the VM
func (c *VMClient) RestartVM(ctx context.Context, name string) (*vmapi.VirtualMachine, error) {
	return c.updateVMAction(ctx, name, vmapi.VMActionRestart)
}

// PauseVM sends a pause action to the VM
func (c *VMClient) PauseVM(ctx context.Context, name string) (*vmapi.VirtualMachine, error) {
	return c.updateVMAction(ctx, name, vmapi.VMActionPause)
}

// ResumeVM sends a resume action to the VM
func (c *VMClient) ResumeVM(ctx context.Context, name string) (*vmapi.VirtualMachine, error) {
	return c.updateVMAction(ctx, name, vmapi.VMActionResume)
}

// WaitForVMState waits for the VM to reach the given state
func (c *VMClient) WaitForVMState(ctx context.Context, name string, state vmapi.VMState, timeout time.Duration) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for the VM to reach the given state
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for VM %s to reach state %s", name, state)
		default:
			// Get the VM
			vm, err := c.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// Check if the VM is in the desired state
			if vm.Status.State == state {
				return nil
			}

			// Sleep for a short time before checking again
			time.Sleep(time.Second)
		}
	}
}

// updateVMAction updates the VM action
func (c *VMClient) updateVMAction(ctx context.Context, name string, action vmapi.VMAction) (*vmapi.VirtualMachine, error) {
	// Get the VM
	vm, err := c.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Update the action
	vm.Spec.Action = action

	// Update the VM
	return c.Update(ctx, vm)
}
