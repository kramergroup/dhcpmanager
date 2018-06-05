package main

import (
	"errors"
	"net"

	"github.com/google/uuid"
	"github.com/kramergroup/dhcpmanager"
)

// Mock implementations

// InMemoryStateManager is an in-memory implementation of the
// StateManager for testing and debugging
type InMemoryStateManager struct {
	allocations map[uuid.UUID]*dhcpmanager.Allocation
	macs        map[string]net.HardwareAddr
	chanCreate  map[chan *dhcpmanager.Allocation]bool
	chanDelete  map[chan *dhcpmanager.Allocation]bool
	chanChange  map[chan *dhcpmanager.Allocation]bool
	chanPop     map[chan *net.HardwareAddr]bool
	chanPush    map[chan *net.HardwareAddr]bool
}

func NewInMemoryStateManager() dhcpmanager.StateManager {
	return InMemoryStateManager{
		allocations: make(map[uuid.UUID]*dhcpmanager.Allocation),
		macs:        make(map[string]net.HardwareAddr),
		chanCreate:  make(map[chan *dhcpmanager.Allocation]bool),
		chanDelete:  make(map[chan *dhcpmanager.Allocation]bool),
		chanChange:  make(map[chan *dhcpmanager.Allocation]bool),
		chanPop:     make(map[chan *net.HardwareAddr]bool),
		chanPush:    make(map[chan *net.HardwareAddr]bool),
	}
}

func (s InMemoryStateManager) MaintainIndices() {}

func (s InMemoryStateManager) Stop() {}

func (s InMemoryStateManager) WatchAllocation(allocationID uuid.UUID, watcher *dhcpmanager.AllocationWatcher) func() {

	chanStop := make(chan bool, 1)
	chanCreate := make(chan *dhcpmanager.Allocation, 1)
	s.chanCreate[chanCreate] = true
	chanDelete := make(chan *dhcpmanager.Allocation, 1)
	s.chanDelete[chanDelete] = true
	chanChange := make(chan *dhcpmanager.Allocation, 1)
	s.chanChange[chanChange] = true

	go func() {
		defer delete(s.chanChange, chanChange)
		defer delete(s.chanCreate, chanCreate)
		defer delete(s.chanDelete, chanDelete)
		for {
			select {
			case al := <-chanCreate:
				if al.ID == allocationID {
					watcher.OnCreate(al)
				}
			case al := <-chanChange:
				if al.ID == allocationID {
					watcher.OnModify(al)
				}
			case al := <-chanDelete:
				if al.ID == allocationID {
					watcher.OnDelete(al)
				}
			case <-chanStop:
				return
			}
		}
	}()
	return func() {
		chanStop <- true
	}

}

func (s InMemoryStateManager) Watch(watcher *dhcpmanager.AllocationWatcher) func() {

	chanStop := make(chan bool, 1)
	chanCreate := make(chan *dhcpmanager.Allocation, 1)
	s.chanCreate[chanCreate] = true
	chanDelete := make(chan *dhcpmanager.Allocation, 1)
	s.chanDelete[chanDelete] = true
	chanChange := make(chan *dhcpmanager.Allocation, 1)
	s.chanChange[chanChange] = true

	go func() {
		defer delete(s.chanChange, chanChange)
		defer delete(s.chanCreate, chanCreate)
		defer delete(s.chanDelete, chanDelete)
		for {
			select {
			case al := <-chanCreate:
				watcher.OnCreate(al)
			case al := <-chanChange:
				watcher.OnModify(al)
			case al := <-chanDelete:
				watcher.OnDelete(al)
			case <-chanStop:
				return
			}
		}
	}()
	return func() {
		chanStop <- true
	}

}

func (s InMemoryStateManager) WatchMACPool(watcher *dhcpmanager.MACPoolWatcher) func() {

	if watcher == nil {
		return func() {}
	}

	chanStop := make(chan bool, 1)
	chanPop := make(chan *net.HardwareAddr, 1)
	chanPush := make(chan *net.HardwareAddr, 1)

	s.chanPop[chanPop] = true
	s.chanPush[chanPush] = true

	go func() {
		defer delete(s.chanPop, chanPop)
		defer delete(s.chanPush, chanPush)

		for {
			select {
			case mac := <-chanPop:
				watcher.OnPop(*mac)
			case mac := <-chanPush:
				watcher.OnPush(*mac)
			case <-chanStop:
				return
			}
		}
	}()

	return func() {
		chanStop <- true
	}
}

func (s InMemoryStateManager) Allocations() ([]*dhcpmanager.Allocation, error) {
	l := make([]*dhcpmanager.Allocation, 0)
	for _, v := range s.allocations {
		l = append(l, v)
	}
	return l, nil
}

func (s InMemoryStateManager) Put(al *dhcpmanager.Allocation) error {

	_, hasID := s.allocations[al.ID]
	s.allocations[al.ID] = al
	if hasID {
		for ch := range s.chanChange {
			ch <- al
		}
	} else {
		for ch := range s.chanCreate {
			ch <- al
		}
	}

	return nil
}

func (s InMemoryStateManager) Remove(al *dhcpmanager.Allocation) error {
	delete(s.allocations, al.ID)
	for ch := range s.chanDelete {
		ch <- al
	}
	return nil
}

func (s InMemoryStateManager) Get(uuid uuid.UUID) (*dhcpmanager.Allocation, error) {
	v, ok := s.allocations[uuid]
	if ok {
		return v, nil
	}
	return nil, errors.New("Allocation not found")
}

func (s InMemoryStateManager) GetByIP(ip *net.IP) (*dhcpmanager.Allocation, error) {

	if ip == nil {
		return nil, errors.New("invalid argument. ip must not be nil")
	}

	for _, v := range s.allocations {
		if v.Lease != nil {
			if v.Lease.FixedAddress.Equal(*ip) {
				return v, nil
			}
		}
	}
	return nil, errors.New("No allocation with ip " + ip.String())
}

func (s InMemoryStateManager) MACPool() ([]string, error) {
	r := make([]string, len(s.macs))
	i := 0
	for k := range s.macs {
		r[i] = k
		i++
	}
	return r, nil
}

func (s InMemoryStateManager) PutMAC(mac net.HardwareAddr) error {
	s.macs[mac.String()] = mac
	for ch := range s.chanPush {
		ch <- &mac
	}

	return nil
}

func (s InMemoryStateManager) RemoveMAC(mac net.HardwareAddr) error {
	if mac, hasID := s.macs[mac.String()]; hasID {
		delete(s.macs, mac.String())
		for ch := range s.chanPop {
			ch <- &mac
		}
		return nil
	}
	return errors.New("Unkown MAC address")
}

func (s InMemoryStateManager) PopMAC() (net.HardwareAddr, error) {
	for k, v := range s.macs {
		delete(s.macs, k)
		for ch := range s.chanPop {
			ch <- &v
		}
		return v, nil
	}
	return nil, errors.New("No available MAC")
}
