package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/kramergroup/dhcpmanager"
)

func obtainIP(w http.ResponseWriter, r *http.Request) {
	ipRequest := new(newIPRequest)
	json.NewDecoder(r.Body).Decode(ipRequest)

	log.Printf("API: ip for %s requested from %s", ipRequest.Service, r.RemoteAddr)

	hostname := hostnameForService(ipRequest.Service)

	allocation := dhcpmanager.NewAllocation(hostname)

	allocCh := make(chan net.IP)
	watcher := dhcpmanager.AllocationWatcher{
		OnModify: func(alloc *dhcpmanager.Allocation) {
			if alloc.Lease != nil {
				// We have obtained a lease. Report back IP
				allocCh <- alloc.Lease.FixedAddress
			}
		},
	}
	stopWatch := sm.WatchAllocation(allocation.ID, &watcher)
	sm.Put(allocation) // Only put allocation after watch is set to avoid race condition

	select {
	case ip := <-allocCh:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newIPRequestResponse{
			IP:     ip.String(),
			ID:     allocation.ID.String(),
			Status: newIPRequestResponseStatusOK,
		})
		log.Printf("API: ip %s assigned to %s ", ip, allocation.Hostname)
	case <-time.After(configuration.RequestTimeout):
		// No response from controller in time - Remove allocation and report back
		sm.Remove(allocation)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newIPRequestResponse{
			IP:     "",
			ID:     allocation.ID.String(),
			Status: "timeout",
		})
		log.Printf("API: ip request for %s timeout", allocation.Hostname)
	}

	stopWatch()
}

func returnIP(w http.ResponseWriter, r *http.Request) {

	request := new(invalidateIPRequest)
	json.NewDecoder(r.Body).Decode(request)
	ip := net.ParseIP(request.IP)

	w.Header().Set("Content-Type", "application/json")
	if ip == nil {
		log.Printf("API: ip return request without IP ignored")
		json.NewEncoder(w).Encode(invalidateIPRequestResponse{
			IP:     "invalid",
			Status: responseStatusError,
		})
		return
	}

	log.Printf("API: IP %s returned from %s", ip.String(), r.RemoteAddr)

	allocation, err := sm.GetByIP(&ip)
	if err != nil {
		log.Printf("API: error obtaining allocation for IP %s - %s", ip.String(), err.Error())
	} else {
		err = sm.Remove(allocation)
	}

	if err != nil {
		json.NewEncoder(w).Encode(invalidateIPRequestResponse{
			IP:     ip.String(),
			Status: responseStatusError,
		})
		return
	}

	json.NewEncoder(w).Encode(invalidateIPRequestResponse{
		IP:     ip.String(),
		ID:     allocation.ID.String(),
		Status: newIPRequestResponseStatusOK,
	})

}

func registerMACs(w http.ResponseWriter, r *http.Request) {
	request := new(registerMACRequest)
	json.NewDecoder(r.Body).Decode(request)

	rejected := make([]string, 0)
	for _, mac := range request.MACs {
		if mmac, err := net.ParseMAC(mac); err == nil {
			sm.PutMAC(mmac)
		} else {
			rejected = append(rejected, mac)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if len(rejected) == 0 {
		json.NewEncoder(w).Encode(registerMACRequestResponse{
			Status: newIPRequestResponseStatusOK,
		})
	} else {
		json.NewEncoder(w).Encode(registerMACRequestResponse{
			Status:   responseStatusError,
			Rejected: rejected,
		})
	}

}

func removeMACs(w http.ResponseWriter, r *http.Request) {
	request := new(removeMACRequest)
	json.NewDecoder(r.Body).Decode(request)

	unprocessed := make([]string, 0)
	for _, mac := range request.MACs {
		if mmac, err := net.ParseMAC(mac); err == nil {
			sm.RemoveMAC(mmac)
		} else {
			unprocessed = append(unprocessed, mac)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if len(unprocessed) == 0 {
		json.NewEncoder(w).Encode(removeMACRequestResponse{
			Status: responseStatusOK,
		})
	} else {
		json.NewEncoder(w).Encode(removeMACRequestResponse{
			Status:      responseStatusError,
			Unprocessed: unprocessed,
		})
	}

}

func returnStatus(w http.ResponseWriter, r *http.Request) {
	allocations, _ := sm.Allocations()
	macs, _ := sm.MACPool()
	status := statusRequestResponse{
		Allocations:   allocations,
		AvailableMACs: macs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// hostnameForService converts "namespace/service" service identifiers into
// proper hostnames of the form "service.namespace"
func hostnameForService(svc string) string {

	parts := strings.Split(svc, "/")
	if len(parts) < 2 {
		return parts[0]
	}
	if len(parts) > 2 {
		log.Printf("Malformated service identifier [%s] - Hostname will be truncated", svc)
	}
	return fmt.Sprintf("%s.%s", parts[1], parts[0])

}
