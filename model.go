package dhcpmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	dhclient "github.com/digineo/go-dhclient"
	"github.com/google/uuid"
)

// AllocationState gives information regarding the state of an Allocation
type AllocationState int

const (
	// Unbound = Request received, but no IP allocation done yet
	Unbound AllocationState = 0

	// Bound = IP allocated and returned
	Bound AllocationState = 1

	// Stale = An IP has been assigned, but is not used (error state)
	Stale AllocationState = 2

	// Stopped = This allocation has been gracefully stopped
	Stopped AllocationState = 3
)

// Allocation is the central data structure that connects a DHCP lease with
// an hostname
type Allocation struct {
	ID        uuid.UUID
	Lease     *dhclient.Lease
	Hostname  string
	State     AllocationState
	Interface net.Interface
}

// AllocationWatcher can be used to watch for state changes
type AllocationWatcher struct {
	OnDelete func(*Allocation)
	OnCreate func(*Allocation)
	OnModify func(*Allocation)
}

// MACPoolWatcher can be used to watch MAC pool events
type MACPoolWatcher struct {
	OnPop  func(net.HardwareAddr)
	OnPush func(net.HardwareAddr)
}

// StateManager manages the application state
// It is the central data management object of the application
// It's responsibilities include:
// 	- Management of indices
//  - Allocation management
//  - MAC table management
type StateManager interface {

	// Lifecycle management
	// --------------------

	// MaintainIndices maintains integrity of the data indices such as
	// the available MAC table
	MaintainIndices()

	// Stop stops all life-cycle threads
	Stop()

	// State Watcher
	// -------------

	// WatchAllocation watches the state of a specific allocation. The function
	// returns a stop function that should be called as soon as the watcher is
	// not needed anymore
	WatchAllocation(allocationID uuid.UUID, watcher *AllocationWatcher) func()

	// Watch watches all allocations. The function
	// returns a stop function that should be called as soon as the watcher is
	// not needed anymore
	Watch(watcher *AllocationWatcher) func()

	// WatchMACPool watches the MAC pool. The function
	// returns a stop function that should be called as soon as the watcher is
	// not needed anymore
	WatchMACPool(watcher *MACPoolWatcher) func()

	// Allocation Management
	// ---------------------

	// Put persists an allocation
	Put(allocation *Allocation) error

	// Remove deletes an allocation
	Remove(allocation *Allocation) error

	// Allocations returns a list of all allocations
	Allocations() ([]*Allocation, error)

	// Get returns the allocation with id
	Get(id uuid.UUID) (*Allocation, error)

	// GetByIP returns the allocation assigned to ip
	GetByIP(ip *net.IP) (*Allocation, error)

	// MAC table management
	// --------------------

	// MacPool returns a list of all available MAC addresses
	MACPool() ([]string, error)

	// PutMAC adds a new MAC to the pool of available MAC addresses
	PutMAC(mac net.HardwareAddr) error

	// RemoveMAC removes a MAC address from the pool of available MAC addresses
	RemoveMAC(mac net.HardwareAddr) error

	// PopMAC take a MAC out of the pool and returns it
	PopMAC() (net.HardwareAddr, error)
}

// stateManager implements the StateManager interface
type stateManager struct {

	// Etcd3 kv
	kv             clientv3.KV
	cli            *clientv3.Client
	stopChan       []chan interface{}
	requestTimeout time.Duration
}

const etcdPrefix = "/kramergroup.science/dhcp-address-space-endpoint"

// NewAllocation creates a new Allocation record and assigns a UUID
func NewAllocation(hostname string) *Allocation {
	return &Allocation{
		ID:       uuid.New(),
		Hostname: hostname,
		State:    Unbound,
	}
}

// NewStateManager creates a new etcd3-backed application state
func NewStateManager(etcdEndpoints []string, dialTimeout, requestTimeout time.Duration) (StateManager, error) {

	sm := stateManager{
		stopChan:       make([]chan interface{}, 0),
		requestTimeout: requestTimeout,
	}

	if cli, err := clientv3.New(clientv3.Config{
		DialTimeout: dialTimeout,
		Endpoints:   etcdEndpoints,
	}); err == nil {
		sm.cli = cli
	} else {
		return nil, err
	}
	sm.kv = clientv3.NewKV(sm.cli)

	return &sm, nil
}

// MaintainIndices watches the state and ensures consistency of indices
func (s *stateManager) MaintainIndices() {

	updateIndex := func(a *Allocation) {
		// Update IP->Allocation.ID lookup table
		if a.Lease != nil {
			ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
			defer cancel()
			key := fmt.Sprintf("%s/lookup/%s", etcdPrefix, a.Lease.FixedAddress)
			_, err := s.kv.Put(ctx, key, a.ID.String())
			if err != nil {
				log.Printf("State: error updating IP<->ID lookup table [%s]", err.Error())
			}
		}
	}

	deleteIndex := func(a *Allocation) {
		// Update IP->Allocation.ID lookup table
		if a.Lease != nil {
			ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
			defer cancel()
			key := fmt.Sprintf("%s/lookup/%s", etcdPrefix, a.Lease.FixedAddress)
			_, err := s.kv.Delete(ctx, key)
			if err != nil {
				log.Printf("State: error deleteing IP<->ID mapping [%s]", err.Error())
			}
		}
	}

	// Ensure consistency of the IP<->ID lookup
	watcher := AllocationWatcher{
		OnModify: updateIndex,
		OnCreate: updateIndex,
		OnDelete: deleteIndex,
	}
	s.Watch(&watcher)

}

// Stop closes the etcd connection backing State
func (s *stateManager) Stop() {
	for _, c := range s.stopChan {
		c <- true
	}
	s.cli.Close()
}

// WatchAllocation watches state changes of the allocation with the given ID
func (s *stateManager) WatchAllocation(allocationID uuid.UUID, watcher *AllocationWatcher) func() {
	stopChan := make(chan interface{})
	s.stopChan = append(s.stopChan, stopChan)
	ctx := context.Background()

	key := fmt.Sprintf("%s/allocations/%s", etcdPrefix, allocationID)
	watchChan := s.cli.Watch(ctx, key, clientv3.WithPrevKV())

	stopFunc := func() {
		stopChan <- true
	}

	// Start a new thread and watch for changes in etcd
	go s.watchChannel(watchChan, stopChan, watcher)

	return stopFunc

}

// Watch uses the supplied AllocationWatcher to watch leases. It returns a function
// that can be used to stop the AllocationWatcher
func (s *stateManager) Watch(watcher *AllocationWatcher) func() {
	stopChan := make(chan interface{})
	s.stopChan = append(s.stopChan, stopChan)
	ctx := context.Background()

	key := fmt.Sprintf("%s/allocations", etcdPrefix)
	watchChan := s.cli.Watch(ctx, key, clientv3.WithPrefix(), clientv3.WithPrevKV())

	stopFunc := func() {
		stopChan <- true
	}

	// Start a new thread and watch for changes in etcd
	go s.watchChannel(watchChan, stopChan, watcher)

	return stopFunc
}

func (s *stateManager) watchChannel(watchChan clientv3.WatchChan, stopChan chan interface{}, watcher *AllocationWatcher) {
	for true {
		select {
		case w := <-watchChan:
			for _, ev := range w.Events {
				//log.Printf("Watch event - Key version: %d, createRev: %d, modRev: %d", ev.Kv.Version, ev.Kv.CreateRevision, ev.Kv.ModRevision)
				switch ev.Type {
				case clientv3.EventTypePut:
					if lease, err := decode(ev.Kv.Value); err == nil {
						if ev.IsCreate() {
							if watcher.OnCreate != nil {
								watcher.OnCreate(lease)
							}
						} else if ev.IsModify() {
							if watcher.OnModify != nil {
								watcher.OnModify(lease)
							}
						}
					} else {
						log.Printf("Error decoding allocation from etcd store: %s", err.Error())
					}
				case clientv3.EventTypeDelete:
					var lease *Allocation
					var err error
					// if ev.PrevKv == nil {
					// 	lease, err = decode(ev.Kv.Value)
					// } else {
					// 	lease, err = decode(ev.PrevKv.Value)
					// }
					lease, err = decode(ev.PrevKv.Value)
					if err == nil {
						if watcher.OnDelete != nil {
							watcher.OnDelete(lease)
						}
					} else {
						log.Printf("Error decoding allocation from etcd store: %s", err.Error())
					}
				}
			}
		case <-stopChan:
			// log.Print("State: watcher stopped")
			return
		}
	}
}

func (s *stateManager) WatchMACPool(watcher *MACPoolWatcher) func() {
	stopChan := make(chan interface{})
	s.stopChan = append(s.stopChan, stopChan)
	ctx := context.Background()

	key := fmt.Sprintf("%s/macs", etcdPrefix)
	watchChan := s.cli.Watch(ctx, key, clientv3.WithPrefix(), clientv3.WithPrevKV())

	stopFunc := func() {
		stopChan <- true
	}

	// Start a new thread and watch for changes in etcd
	go s.watchMACPool(watchChan, stopChan, watcher)

	return stopFunc
}

func (s *stateManager) watchMACPool(watchChan clientv3.WatchChan, stopChan chan interface{}, watcher *MACPoolWatcher) {
	for true {
		select {
		case w := <-watchChan:
			for _, ev := range w.Events {
				//log.Printf("Watch event - Key version: %d, createRev: %d, modRev: %d", ev.Kv.Version, ev.Kv.CreateRevision, ev.Kv.ModRevision)
				switch ev.Type {
				case clientv3.EventTypePut:
					mac, err := net.ParseMAC(string(ev.Kv.Key))
					if err == nil && ev.IsCreate() {
						if watcher.OnPush != nil {
							watcher.OnPush(mac)
						}
					}
				case clientv3.EventTypeDelete:
					mac, err := net.ParseMAC(string(ev.PrevKv.Key))
					if err == nil && watcher.OnPop != nil {
						watcher.OnPop(mac)
					}
				}
			}
		case <-stopChan:
			// log.Print("State: watcher stopped")
			return
		}
	}
}

// Put a lease into the state store
func (s *stateManager) Put(allocation *Allocation) error {

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	// Encode
	var b []byte
	var err error
	if b, err = encode(allocation); err != nil {
		log.Printf("State: error econding lease [%s]", err.Error())
		return err
	}

	if allocation.Lease != nil {
		// If we have a lease, propagate expiry to the allocation record using etcd leases
		ttl := int64(time.Until(allocation.Lease.Expire).Seconds())
		ls, err := s.cli.Grant(ctx, ttl)
		if err != nil {
			log.Printf("State: %s", err.Error())
			return err
		}

		// Put
		key := fmt.Sprintf("%s/allocations/%s", etcdPrefix, allocation.ID)
		_, err = s.kv.Put(ctx, key, string(b), clientv3.WithLease(ls.ID))

		if err != nil {
			log.Printf("State: error writing to etcd [%s]", err.Error())
			return err
		}
	} else {
		// Allication has no lease yet, store without etcd lease
		// Put
		key := fmt.Sprintf("%s/allocations/%s", etcdPrefix, allocation.ID)
		_, err := s.kv.Put(ctx, key, string(b))

		if err != nil {
			log.Printf("State: error writing to etcd [%s]", err.Error())
			return err
		}
	}

	return nil
}

// Remove removes an Allocation from the KV store
func (s *stateManager) Remove(allocation *Allocation) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	_, err := s.kv.Delete(ctx,
		fmt.Sprintf("%s/allocations/%s", etcdPrefix, allocation.ID))
	if err != nil {
		log.Printf("State: error removing IP %s", err.Error())
		return err
	}

	if allocation.Lease != nil {
		_, err := s.kv.Delete(ctx,
			fmt.Sprintf("%s/lookup/%s", etcdPrefix, allocation.Lease.FixedAddress))
		if err != nil {
			log.Printf("State: error updating IP<->ID lookup table [%s]", err.Error())
			return err
		}
	}
	return nil
}

// Allocations returns an iterator over all allocations
func (s *stateManager) Allocations() ([]*Allocation, error) {

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	key := fmt.Sprintf("%s/allocations", etcdPrefix)
	gr, err := s.kv.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	allocations := make([]*Allocation, gr.Count)
	for i, item := range gr.Kvs {
		allocations[i], err = decode(item.Value)
		if err != nil {
			return nil, err
		}
	}

	return allocations, nil
}

// Get returns the allocation with ID. An error results from requesting the allocation
// of an unknown ID
func (s *stateManager) Get(id uuid.UUID) (*Allocation, error) {

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	gr, err := s.kv.Get(ctx, fmt.Sprintf("%s/allocations/%s", etcdPrefix, id))
	if err != nil {
		return nil, err
	}

	if gr.Count == 0 {
		return nil, fmt.Errorf("No allocation for D %s", id.String())
	}
	return decode(gr.Kvs[0].Value)
}

func decode(value []byte) (*Allocation, error) {
	allocation := &Allocation{}
	err := json.NewDecoder(bytes.NewBuffer(value)).Decode(allocation)
	return allocation, err
}

func encode(allocation *Allocation) ([]byte, error) {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(allocation)
	return b.Bytes(), err
}

// GetByIP returns the allocation by IP
func (s *stateManager) GetByIP(ip *net.IP) (*Allocation, error) {

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	gr, err := s.kv.Get(ctx, fmt.Sprintf("%s/lookup/%s", etcdPrefix, ip.String()))
	if err != nil {
		return nil, err
	}

	if gr.Count == 0 {
		return nil, fmt.Errorf("No allocation for IP %s in index", ip.String())
	}

	var uid uuid.UUID
	uid, err = uuid.ParseBytes(gr.Kvs[0].Value)
	if err != nil {
		return nil, err
	}
	return s.Get(uid)

}

// MACPool returns a list of available MAC addresses
func (s *stateManager) MACPool() ([]string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()
	key := fmt.Sprintf("%s/macs", etcdPrefix)
	gr, err := s.kv.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	macs := make([]string, gr.Count)
	for i, kv := range gr.Kvs {
		macs[i] = string(kv.Value)
	}

	return macs, nil
}

// PutMAC puts a MAC into the pool of available MAC addresses
func (s *stateManager) PutMAC(mac net.HardwareAddr) error {

	if len(mac) == 0 {
		return fmt.Errorf("Empty MAC")
	}

	amac := strings.ToLower(mac.String())
	if amac == "" {
		return fmt.Errorf("Invalid MAC format [%s]", mac.String())
	}

	// Check if MAC is already in use
	allocations, err := s.Allocations()
	if err != nil {
		return err
	}

	for _, al := range allocations {
		mmac := strings.ToLower(al.Interface.HardwareAddr.String())
		if mmac == amac {
			return fmt.Errorf("MAC address already in use by allocation [%s]", al.ID)
		}
	}

	// A genuinely new or currently unused MAC - persist
	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()
	key := fmt.Sprintf("%s/macs/%s", etcdPrefix, amac)
	_, err = s.kv.Put(ctx, key, amac)

	return err
}

// RemoveMAC removes a MAC from the pool
func (s *stateManager) RemoveMAC(mac net.HardwareAddr) error {
	amac := strings.ToLower(mac.String())

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()
	key := fmt.Sprintf("%s/macs/%s", etcdPrefix, amac)
	_, err := s.kv.Delete(ctx, key)

	return err
}

// PopMAC retrieves a MAC from the pool of available MAC addresses
func (s *stateManager) PopMAC() (net.HardwareAddr, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	key := fmt.Sprintf("%s/macs", etcdPrefix)
	gr, err := s.kv.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithLimit(1))
	if err != nil {
		return nil, err
	}
	if gr.Count < 1 {
		return nil, errors.New("No available MAC")
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel2()
	if _, err = s.kv.Delete(ctx2, string(gr.Kvs[0].Key)); err != nil {
		log.Printf("Error deleting MAC key [%s]", string(gr.Kvs[0].Key))
		return nil, err
	}

	return net.ParseMAC(string(gr.Kvs[0].Value))
}

// Custom JSON (un)mashalling
// Source: http://choly.ca/post/go-json-marshalling/
// We need to use a custom marshalling/unmarshalling approach to
// properly convert net.HardwareAddr in the net.Interface struct to
// a standard string representation

type InterfaceAlias net.Interface

func (a InterfaceAlias) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Index        int    // positive integer that starts at one, zero is never used
		MTU          int    // maximum transmission unit
		Name         string // e.g., "en0", "lo0", "eth0.100"
		HardwareAddr string // IEEE MAC-48, EUI-48 and EUI-64 form
		Flags        net.Flags
	}{
		Index:        a.Index,
		MTU:          a.MTU,
		Name:         a.Name,
		HardwareAddr: a.HardwareAddr.String(),
		Flags:        a.Flags,
	})
}

func (a *InterfaceAlias) UnmarshalJSON(data []byte) error {

	// Catch null values
	if len(data) == 0 {
		return nil
	}

	aux := &struct {
		Index        int    // positive integer that starts at one, zero is never used
		MTU          int    // maximum transmission unit
		Name         string // e.g., "en0", "lo0", "eth0.100"
		HardwareAddr string // IEEE MAC-48, EUI-48 and EUI-64 form
		Flags        net.Flags
	}{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	a.Index = aux.Index
	a.MTU = aux.MTU
	a.Name = aux.Name
	a.Flags = aux.Flags

	var err error
	if aux.HardwareAddr != "" { // Catch default "empty" value
		if a.HardwareAddr, err = net.ParseMAC(aux.HardwareAddr); err != nil {
			return err
		}
	}
	return nil
}

func (n *Allocation) MarshalJSON() ([]byte, error) {

	type AllocationAlias Allocation
	return json.Marshal(&struct {
		Interface InterfaceAlias
		*AllocationAlias
	}{
		Interface:       (InterfaceAlias)(n.Interface),
		AllocationAlias: (*AllocationAlias)(n),
	})
}

func (n *Allocation) UnmarshalJSON(data []byte) error {
	type AllocationAlias Allocation
	aux := &struct {
		Interface InterfaceAlias
		*AllocationAlias
	}{
		AllocationAlias: (*AllocationAlias)(n),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	n.Interface = (net.Interface)(aux.Interface)
	return nil
}
