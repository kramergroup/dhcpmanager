package dhcpmanager

import (
	"errors"
	"log"
	"net"
	"time"

	dhclient "github.com/digineo/go-dhclient"
	"github.com/vishvananda/netlink"
)

// DHCPController manages the DHCP clients
type DHCPController struct {
	iface            string
	timeout          time.Duration
	clients          map[string]*dhclient.Client
	manageInterfaces bool
	assignInterfaces bool
}

// NewDHCPController creates a new DHCPController for an interface
func NewDHCPController(iface string, timeout time.Duration, manageInterfaces, assignInterfaces bool) *DHCPController {
	c := DHCPController{
		timeout:          timeout,
		clients:          make(map[string]*dhclient.Client),
		manageInterfaces: manageInterfaces,
		iface:            iface,
		assignInterfaces: assignInterfaces,
	}
	return &c
}

// BindAllocationToInterface create a new DHCP client with the interface and bind to allocation
func (c *DHCPController) BindAllocationToInterface(allocation *Allocation, iface *net.Interface, onRenew func(*net.Interface, *dhclient.Lease)) (*dhclient.Lease, error) {

	boundCh := make(chan *dhclient.Lease)
	client := dhclient.Client{
		Iface:    iface,
		Hostname: allocation.Hostname,

		OnBound: func(lease *dhclient.Lease) {
			// Non-blocking send  because we only have a receiver for the first call
			// But the OnBound callback is also executed for renewals, which we use
			// to update state
			select {
			case boundCh <- lease:
			default:
				onRenew(iface, lease)
			}
		},
	}
	client.Start()
	select {
	case lease := <-boundCh:
		// First check if a client is already handling this IP and stop
		if _, ok := c.clients[lease.FixedAddress.String()]; ok {
			client.Stop()
			return nil, errors.New("IP address already managed")
		}
		if c.assignInterfaces {
			c.associateLeasewithDevice(lease, iface)
		}
		c.clients[lease.FixedAddress.String()] = &client
		return lease, nil
	case <-time.After(c.timeout):
		log.Printf("Timeout binding to interface [%s] for %s", c.iface, allocation.Hostname)
		client.Stop()
		return nil, errors.New("Timeout binding to interface")
	}

}

// Interface returns the parent interface for the DHCP clients
func (c *DHCPController) Interface() (*net.Interface, error) {
	return net.InterfaceByName(c.iface)
}

// Stop stops the DHCP client keeping ip alive
func (c *DHCPController) Stop(ip *net.IP) {

	if client, ok := c.clients[ip.String()]; ok {
		delete(c.clients, ip.String())
		client.Stop()
		log.Printf("Stopped managing IP %s for %s", ip.String(), client.Hostname)
	} else {
		log.Printf("Cannot stop DHCP client for IP %s - No known client", ip.String())
	}

}

func (c *DHCPController) associateLeasewithDevice(lease *dhclient.Lease, iface *net.Interface) {

	link, _ := netlink.LinkByName(iface.Name)

	cidr := net.IPNet{
		IP:   lease.FixedAddress,
		Mask: lease.Netmask,
	}
	addr, _ := netlink.ParseAddr(cidr.String())
	log.Printf("Adding %s to link %s", cidr.String(), iface.Name)

	netlink.AddrAdd(link, addr)
}

// CreateDevice creates a new network interface and bridges it to iface
func (c *DHCPController) CreateDevice(ifName string, mac *net.HardwareAddr) (*net.Interface, error) {

	parent, err := netlink.LinkByName(c.iface)
	if err != nil {
		return nil, err
	}
	la := netlink.LinkAttrs{
		Name:        ifName,
		ParentIndex: parent.Attrs().Index,
	}

	log.Printf("MAC address: %s", mac)
	if mac != nil {
		la.HardwareAddr = *mac
	}

	mybridge := &netlink.Macvlan{
		LinkAttrs: la,
		Mode:      netlink.MACVLAN_MODE_BRIDGE,
	}
	err = netlink.LinkAdd(mybridge)
	if err != nil {
		log.Printf("could not add interface %s: %v\n", la.Name, err)
		return nil, err
	}

	ifn, err := net.InterfaceByName(la.Name)
	netlink.LinkSetUp(mybridge)
	return ifn, err
}

// RemoveDevice removes virtual NICs
func RemoveDevice(iface *net.Interface) error {
	link, err := netlink.LinkByName(iface.Name)
	if err != nil {
		return err
	}
	return netlink.LinkDel(link)
}
