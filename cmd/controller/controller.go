package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	dhclient "github.com/digineo/go-dhclient"
	dhcpmanager "github.com/kramergroup/dhcpmanager"
)

// Controller handles state changes to DHCP leases
type Controller struct {
	sm                dhcpmanager.StateManager
	dhcp              *dhcpmanager.DHCPController
	watchStopFunc     func()
	createInterfaces  bool
	dynamicInterfaces bool
}

// NewController creates a new controller
func NewController(StateManager dhcpmanager.StateManager, client *dhcpmanager.DHCPController, manageInterfaces, dynamicInterfaces bool) *Controller {
	c := Controller{
		sm:                StateManager,
		dhcp:              client,
		createInterfaces:  manageInterfaces,
		dynamicInterfaces: dynamicInterfaces,
	}
	return &c
}

// Start the controller main loop
func (c *Controller) Start() {
	if c.watchStopFunc == nil {
		c.converge()
		go c.watch()
	}
}

// Stop stops the controller operation
func (c *Controller) Stop() {
	if c.watchStopFunc != nil {
		c.watchStopFunc()
	}
	if allocations, err := c.sm.Allocations(); err == nil {
		for _, allocation := range allocations {
			if allocation.State == dhcpmanager.Bound {
				if allocation.Lease != nil {
					c.dhcp.Stop(&allocation.Lease.FixedAddress)
				}
				if c.createInterfaces {
					dhcpmanager.RemoveDevice(&allocation.Interface)
				}
				allocation.State = dhcpmanager.Stopped
				c.sm.Put(allocation)
			}
		}
	}
	c.watchStopFunc = nil
}

func (c *Controller) converge() {

	// Enque all leases for renewal
	allocations, err := c.sm.Allocations()
	if err != nil {
		log.Fatalf("Could not read leases from kv store. [%s]", err.Error())
	}

	for _, allocation := range allocations {
		switch allocation.State {
		// This is a gracefully stopped allocation - try to resurrect
		case dhcpmanager.Stopped:
			c.processStoppedAllocation(allocation)
		// This is a new allocation that has never been assigned
		case dhcpmanager.Unbound:
			c.processUnboundAllocation(allocation)
		// This is a stale allocation
		case dhcpmanager.Stale:
			log.Printf("Stale allocation [%s] removed", allocation.ID)
			c.deleteAllocation(allocation)
		// This is an already bound allocation
		case dhcpmanager.Bound:

		}
	}
}

func (c *Controller) processUnboundAllocation(allocation *dhcpmanager.Allocation) {

	renewCallback := func(iface *net.Interface, lease *dhclient.Lease) {
		allocation.Lease = lease
		if err := c.sm.Put(allocation); err != nil {
			log.Printf("Warning: Error persisting allocation for IP %s = %s", lease.FixedAddress.String(), err.Error())
		}
	}

	var iface *net.Interface
	if c.createInterfaces {
		var err error
		ifName := fmt.Sprintf("vf-%s", randomString(6))
		mac, err := c.sm.PopMAC()
		if err != nil {
			if !c.dynamicInterfaces {
				log.Print("Warning: No valid MAC address")
				return
			}
			mac = nil // causes randomn MAC generation in dhclient
		}
		iface, err = c.dhcp.CreateDevice(ifName, &mac)
		if err != nil {
			log.Printf("Warning: Could not create device [%s] - %s", ifName, err.Error())

			// Not sure if the allocation should be deleted at this point. Probably not
			// to give others the option to process it
			// c.deleteAllocation(allocation)

			// make sure the mac is returned
			// At this point it is probably not be bound to the allocation and is, therefore,
			// not released when the allocation is deleted
			if allocation.Interface.HardwareAddr.String() != mac.String() {
				c.sm.PutMAC(mac)
			}
			return
		}
	} else {
		var err error
		iface, err = c.dhcp.Interface()
		if err != nil {
			log.Printf("Warning: Could not access device")
			// Not sure if the allocation should be deleted at this point. Probably not
			// to give others the option to process it. It's the job of the creator to
			// remove stale allocations
			// c.deleteAllocation(allocation)
			return
		}
	}

	lease, err := c.dhcp.BindAllocationToInterface(allocation, iface, renewCallback)
	if err != nil {
		log.Printf("Warning: Could not bind allocation [%s] to device [%s]", allocation.ID, allocation.Interface.Name)
		c.sm.Remove(allocation)
		return
	}
	allocation.Lease = lease
	allocation.Interface = *iface
	allocation.State = dhcpmanager.Bound

	if err := c.sm.Put(allocation); err != nil {
		log.Printf("Warning: Error persisting allocation for IP %s = %s", lease.FixedAddress.String(), err.Error())
	}

	log.Printf("Allocation %s bound to interface %s with IP %s (%s)",
		allocation.ID, allocation.Interface.Name, allocation.Lease.FixedAddress, allocation.Hostname)
}

func (c *Controller) processStoppedAllocation(allocation *dhcpmanager.Allocation) {

	if allocation == nil {
		return
	}

	renewCallback := func(iface *net.Interface, lease *dhclient.Lease) {
		allocation.Lease = lease
		if err := c.sm.Put(allocation); err != nil {
			log.Printf("Warning: Error persisting allocation for IP %s = %s", lease.FixedAddress.String(), err.Error())
		}
	}

	if allocation.Lease != nil && allocation.Lease.Expire.Before(time.Now()) {
		log.Printf("Warning: lease for IP %s already expired.", allocation.Lease.FixedAddress)
		c.deleteAllocation(allocation)
		return
	}

	var iface *net.Interface
	if c.createInterfaces {
		var err error
		iface, err = c.dhcp.CreateDevice(allocation.Interface.Name, &allocation.Interface.HardwareAddr)
		if err != nil {
			log.Printf("Warning: Could not create device [%s] - %s", allocation.Interface.Name, err.Error())
			// Not sure if the allocation should be deleted at this point. Probably not
			// to give others the option to process it. It's the job of the creator to
			// remove stale allocations
			// c.deleteAllocation(allocation)
			return
		}
		allocation.Interface = *iface
	} else {
		var err error
		iface, err = c.dhcp.Interface()
		if err != nil {
			log.Printf("Warning: Could not access device")
			// Not sure if the allocation should be deleted at this point. Probably not
			// to give others the option to process it. It's the job of the creator to
			// remove stale allocations
			// c.deleteAllocation(allocation)
			return
		}
	}

	lease, err := c.dhcp.BindAllocationToInterface(allocation, iface, renewCallback)
	if err != nil {
		log.Printf("Warning: Could not bind stopped allocation [%s] to device [%s]", allocation.ID, allocation.Interface.Name)
		// Not sure if the allocation should be deleted at this point. Probably not
		// to give others the option to process it. It's the job of the creator to
		// remove stale allocations
		// c.deleteAllocation(allocation)
		return
	}
	allocation.Lease = lease
	allocation.State = dhcpmanager.Bound

	// Make sure the MAC is not left in the pool
	c.sm.RemoveMAC(allocation.Interface.HardwareAddr)

	if err := c.sm.Put(allocation); err != nil {
		log.Printf("Warning: Error persisting allocation for IP %s = %s", lease.FixedAddress.String(), err.Error())
	}

}

func (c *Controller) deleteAllocation(allocation *dhcpmanager.Allocation) {

	if allocation.Lease != nil {
		log.Printf("Controller: Stopping DHCP client for IP %s", allocation.Lease.FixedAddress.String())
		c.dhcp.Stop(&allocation.Lease.FixedAddress)
	}

	// Recover the MAC if we are managing interfaces
	if c.createInterfaces && len(allocation.Interface.HardwareAddr) > 0 {
		c.sm.PutMAC(allocation.Interface.HardwareAddr)
		dhcpmanager.RemoveDevice(&allocation.Interface)
	}
	allocation.State = dhcpmanager.Stale
}

// watch for changes in the allocation store
func (c *Controller) watch() {
	watcher := dhcpmanager.AllocationWatcher{

		// Remove DHCP client if allocation is removed
		OnDelete: func(alloc *dhcpmanager.Allocation) {
			c.deleteAllocation(alloc)
		},

		// Create DHCP client if allocation is created
		OnCreate: func(alloc *dhcpmanager.Allocation) {

			if alloc.State != dhcpmanager.Unbound {
				log.Printf("Created allocation %s not unbound. Ignore.", alloc.ID)
				return
			}
			c.processUnboundAllocation(alloc)

		},
	}
	c.watchStopFunc = c.sm.Watch(&watcher)
}

func randomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
