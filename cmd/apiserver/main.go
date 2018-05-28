package main

import (
	"fmt"
	"log"

	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kramergroup/dhcpmanager"
	"github.com/spf13/viper"
)

// Configuration holds the global apiserver configuration
type Configuration struct {
	Port           int
	Cidrs          []string
	Etcd           []string
	RequestTimeout time.Duration `mapstructure:"request-timeout"`
	DialTimeout    time.Duration `mapstructure:"dial-timeout"`
}

var configuration Configuration
var sm *dhcpmanager.StateManager

// our main function
func main() {

	processConfiguration()

	var err error
	sm, err = dhcpmanager.NewStateManager(configuration.Etcd, configuration.DialTimeout, configuration.RequestTimeout)
	if err == nil {
		ListenAndServe()
	} else {
		log.Fatalf("Error starting: %s", err.Error())
	}
}

// ListenAndServe starts the HTTP server and listens for requests
func ListenAndServe() {
	router := mux.NewRouter()

	router.HandleFunc(
		fmt.Sprintf(apiEndpointObtainIP.TemplateURL, ""),
		obtainIP).Methods(apiEndpointObtainIP.Method)

	router.HandleFunc(
		fmt.Sprintf(apiEndpointReturnIP.TemplateURL, ""),
		returnIP).Methods(apiEndpointReturnIP.Method)

	router.HandleFunc(
		fmt.Sprintf(apiEndpointRegisterMAC.TemplateURL, ""),
		registerMACs).Methods(apiEndpointRegisterMAC.Method)

	router.HandleFunc(
		fmt.Sprintf(apiEndpointRemoveMAC.TemplateURL, ""),
		removeMACs).Methods(apiEndpointRemoveMAC.Method)

	router.HandleFunc(
		fmt.Sprintf(apiEndpointStatus.TemplateURL, ""),
		returnStatus).Methods(apiEndpointStatus.Method)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", configuration.Port), router))
}

func processConfiguration() {

	viper.SetConfigName("dhcpmanager")
	viper.AddConfigPath("/etc/dhcpmanager")
	viper.SetEnvPrefix("DHCP")
	viper.AutomaticEnv()

	viper.SetDefault("etcd", []string{"etcd:2379"})
	viper.SetDefault("port", 8000)
	viper.SetDefault("dial-timeout", "5s")
	viper.SetDefault("request-timeout", "10s")
	viper.SetDefault("cidrs", []string{"192.168.0.0/16"})

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Configuration error: %s", err.Error())
	}

	if err := viper.Unmarshal(&configuration); err != nil {
		log.Fatalf("Configuration error: %s", err.Error())
	}

	log.Printf("[config]            port: %d", configuration.Port)
	log.Printf("[config] request-timeout: %s", configuration.RequestTimeout)
	log.Printf("[config]    dial-timeout: %s", configuration.DialTimeout)
	log.Printf("[config]            etcd: %s", configuration.Etcd)
	log.Printf("[config]           cidrs: %s", configuration.Cidrs)
}
