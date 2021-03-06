package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kramergroup/dhcpmanager"

	"github.com/spf13/viper"
)

var controller *Controller

// Configuration structure for the application
type Configuration struct {

	// Array of etcd endpoints
	//
	// Default: etcd:2379
	Etcd []string

	// The name of the interface used to send DHCP packages
	// If the managed interface feature is enabled, virtual interfaces
	// will be connected in bridge-mode to this interface as well
	//
	// Default: eth0
	Interface string

	// Timeout to reach etcd in seconds
	//
	// Default: 5 sec
	DialTimeout time.Duration `mapstructure:"dial-timeout"`

	// Timeout for requests in seconds
	//
	// Default: 10 sec
	RequestTimeout time.Duration `mapstructure:"request-timeout"`

	// Timeout for DHCP client operations in seconds
	//
	// Default: 5 sec
	ClientTimeout time.Duration `mapstructure:"client-timeout"`

	// The managed interface feature is turned on if true
	// Many DHCP servers do not issue more than one IP per MAC address
	// If this is the case, managedInterfaces will create virtual interfaces
	// and manage them. IP addresses are obtained for these interfaces, but not
	// associated with the interface. No network packages will, therefore, be
	// picked up by the virtual interfaces. If an IP expires or is returned,
	// the associated virtual interface is returned.
	//
	// Default: true
	ManageInterfaces bool `mapstructure:"manage-interfaces"`

	// If true, the controller adds the received IP to the interface. If used
	// in conjunction with a load-balancer, this is usually not what is required
	// as it is the load-balancers job to setup routing.
	//
	// Default: false
	AssignInterfaces bool `mapstructure:"assign-interfaces"`

	// The MAC addresses that will be used to obtain unique IPs from the DHCP
	// server if ManageInterfaces = true
	// If manage-interfaces is set to true, the list of MACs defines the total
	// size of the IP pool available unless dynamic-interfaces is true
	Macs []string

	// Dynamic interfaces allows to generate randomn hardware MAC addresses as
	// needed. This has the advantage that the pool of IP is essentially infinite.
	// Some environments use the MAC to enforce security rules. In these circumstances,
	// randomn MACs should not be used.
	//
	// Default: false
	DynamicInterfaces bool `mapstructure:"dynamic-interfaces"`
}

func main() {

	// Process configuration
	config := processConfiguration()

	// Start Controller and Manager
	dhcp := dhcpmanager.NewDHCPController(config.Interface,
		config.ClientTimeout, config.ManageInterfaces, config.AssignInterfaces)
	sm, err := dhcpmanager.NewStateManager(config.Etcd, config.DialTimeout, config.RequestTimeout)
	if err == nil {

		// Register the MAC addresses
		for _, mac := range config.Macs {
			mmac, errB := net.ParseMAC(mac)
			if errB != nil {
				log.Printf("Invalid MAC address [%s]", mac)
				continue
			}

			switch sm.PutMAC(mmac) {
			case nil:
				log.Printf("Registered MAC [%s] with pool", mac)
			default:
				log.Printf("Error registering MAC [%s] with pool", mac)
			}
		}

		// Start the main controller syncing state with DHCP clients
		controller = NewController(sm, dhcp, config.ManageInterfaces, config.DynamicInterfaces)
		log.Print("Controller: starting")
		controller.Start()

		// Start a watcher to maintain indicies
		sm.MaintainIndices()

		// Wait for system signals to shutdown
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs

		controller.Stop()
		log.Print("Controller: stopped")
	} else {
		log.Fatalf("Controller: %s", err.Error())
	}
}

func processConfiguration() *Configuration {

	viper.SetConfigName("dhcpmanager")
	viper.AddConfigPath("/etc/dhcpmanager")
	viper.SetEnvPrefix("DHCP")
	viper.AutomaticEnv()

	viper.SetDefault("etcd", []string{"etcd:2379"})
	viper.SetDefault("interface", "eth0")
	viper.SetDefault("dial-timeout", "5s")
	viper.SetDefault("request-timeout", "10s")
	viper.SetDefault("client-timeout", "5s")
	viper.SetDefault("manage-interfaces", true)
	viper.SetDefault("assign-interfaces", false)
	viper.SetDefault("dynamic-interfaces", false)

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Configuration error: %s", err.Error())
	}

	config := Configuration{}
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Configuration error: %s", err.Error())
	}

	log.Printf("[config]          interface: %s", config.Interface)
	log.Printf("[config]  manage-interfaces: %t", config.ManageInterfaces)
	log.Printf("[config]  assign-interfaces: %t", config.AssignInterfaces)
	log.Printf("[config] dynamic-interfaces: %t", config.DynamicInterfaces)
	log.Printf("[config]      MAC pool size: %d", len(config.Macs))
	log.Printf("[config]               etcd: %s", config.Etcd)
	log.Printf("[config]     client-timeout: %s", config.ClientTimeout)
	log.Printf("[config]    request-timeout: %s", config.RequestTimeout)
	log.Printf("[config]       dial-timeout: %s", config.DialTimeout)

	return &config
}
