package main

import (
	"github.com/kramergroup/dhcpmanager"
)

// newIPRequest is send to Endpoint to request minting of a new IP
type newIPRequest struct {
	Service string `json:"service"` // Name of the service the IP is intended for
}

// invalidateIPRequest is send to Endpoint to inform the service that
// this IP is no longer in use
type invalidateIPRequest struct {
	IP string `json:"ip"` // IP that is released
}

// validateIPRequest is send to Endpoint to validate that Endpoint is aware
// of the IP and manages it
type validateIPRequest struct {
	IP string `json:"string"` // IP that should be validated
}

// newIPRequestResponse is send as response to newIPRequest requests
type newIPRequestResponse struct {
	IP     string `json:"ip"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

// newIPRequestResponse is send as response to newIPRequest requests
type validateIPRequestResponse struct {
	IP     string `json:"ip"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

type invalidateIPRequestResponse struct {
	IP     string `json:"ip"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

type registerMACRequest struct {
	MACs []string
}

type removeMACRequest struct {
	MACs []string
}

type registerMACRequestResponse struct {
	Status   string
	Rejected []string
}

type removeMACRequestResponse struct {
	Status      string
	Unprocessed []string
}

type statusRequestResponse struct {
	Allocations   []*dhcpmanager.Allocation
	AvailableMACs []string
}

type apiEndpoint struct {
	TemplateURL string
	Method      string
}

const (
	// newIPRequestResponseStatusOK indicates a successful response to an newIPRequest
	newIPRequestResponseStatusOK = "success"

	// validateIPRequestResponseVALID indicates a valid IP
	validateIPRequestResponseVALID = "valid"

	// responseStatusOK indicates successful execution of the request
	responseStatusOK = "success"

	// responseStatusError indicates an error during processing of the request
	responseStatusError = "error"

	// rresp
	responseStatusTimeout = "timeout"
)

var (
	apiEndpointObtainIP = apiEndpoint{
		TemplateURL: "%s/v1/ip",
		Method:      "POST",
	}

	apiEndpointReturnIP = apiEndpoint{
		TemplateURL: "%s/v1/ip",
		Method:      "DELETE",
	}

	apiEndpointConfiguration = apiEndpoint{
		TemplateURL: "%s/v1/config",
		Method:      "GET",
	}

	apiEndpointStatus = apiEndpoint{
		TemplateURL: "%s/v1/status",
		Method:      "GET",
	}

	apiEndpointRegisterMAC = apiEndpoint{
		TemplateURL: "%s/v1/mac",
		Method:      "POST",
	}

	apiEndpointRemoveMAC = apiEndpoint{
		TemplateURL: "%s/v1/mac",
		Method:      "DELETE",
	}
)
