package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kramergroup/dhcpmanager"
	"github.com/spf13/viper"
)

/*
	Backend provides server-side services for the dhcpmanager-ui.

	This is includes:

	- /ws/allocations - A websocket broadcasting changes to the allocations
	- /ws/macs - A websocket broadcasting changes to the MAC table

	- /api/allocations - CRUD endpoint for allocation manipulation
	- /api/macs - CRUD endpoint for MAC table manipulation
*/

type Configuration struct {
	EtcdEndpoints  []string      `mapstructure:"etcd"`
	Port           int           `mapstructure:"ui-port"`
	RequestTimeout time.Duration `mapstructure:"request-timeout"`
	DialTimeout    time.Duration `mapstructure:"dial-timeout"`
}

type Response struct {
	Status string `json:"status"`
	Info   string `json:"info"`
}

type AllocationsUpdate struct {
	Response
	Data []*dhcpmanager.Allocation
}

type MACPoolUpdate struct {
	Response
	NumAvailable int      `json:"available"`
	NumBound     int      `json:"bound"`
	Macs         []string `json:"macs"`
}

type AllocationRequest struct {
	Hostname string
}

type AddMACRequest struct {
	Macs []string
}

var config Configuration
var sm dhcpmanager.StateManager
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

func main() {

	// Process configuration
	processConfiguration()

	// Backing infrastructure
	var err error
	sm, err = dhcpmanager.NewStateManager(config.EtcdEndpoints, config.DialTimeout, config.RequestTimeout)
	if err != nil {
		log.Fatalf("Could not access etcd at %s", config.EtcdEndpoints)
	}
	//sm = NewInMemoryStateManager()

	// Routing
	router := mux.NewRouter()
	router.HandleFunc("/ws/allocations", pushAllocationChange)
	router.HandleFunc("/ws/macpool", pushMACPoolChange)
	router.HandleFunc("/api/allocations", addAllocation).Methods("POST")
	router.HandleFunc("/api/macs", addMAC).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("/static/")))

	// Start http server
	log.Printf("Start listening on port %d", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), router))

}

// Handler functions

func pushAllocationChange(w http.ResponseWriter, r *http.Request) {

	// Upgrate to websocket connection
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	// Observe system signals for graceful shutdown
	syssig := make(chan os.Signal, 1)
	signal.Notify(syssig, syscall.SIGINT, syscall.SIGTERM)

	watchersig := make(chan bool, 1)
	trigger := func(a *dhcpmanager.Allocation) {
		watchersig <- true
	}
	watcher := dhcpmanager.AllocationWatcher{
		OnDelete: trigger,
		OnModify: trigger,
		OnCreate: trigger,
	}

	stopWatcher := sm.Watch(&watcher)
	defer stopWatcher()

	// The write function that serialises the list of allocations and sends
	// across the socket
	serialise := func() {
		var update AllocationsUpdate
		allocs, err := sm.Allocations()
		if err != nil {
			log.Printf("Error obtaining allocations [%s]", err.Error())
			update = AllocationsUpdate{
				Data: nil,
				Response: Response{
					Status: "error",
					Info:   "Internal error while obtaining allocations",
				},
			}
		} else {
			update = AllocationsUpdate{
				Data:     allocs,
				Response: Response{Status: "success", Info: ""},
			}
		}
		c.WriteJSON(update)
	}

	// Send current state to new client
	serialise()

	// Watch etcd changes and push new allocation state
	for {
		select {
		case <-syssig:
			// System signal - terminate gracefully
			return
		case <-watchersig:
			// Allocation state changed - push new list of allocations
			serialise()
		}
	}
}

func pushMACPoolChange(w http.ResponseWriter, r *http.Request) {

	// Upgrate to websocket connection
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	// Observe system signals for graceful shutdown
	syssig := make(chan os.Signal, 1)
	signal.Notify(syssig, syscall.SIGINT, syscall.SIGTERM)

	// Watch the MAC pool
	watchersig := make(chan bool, 1)
	trigger := func(a net.HardwareAddr) {
		watchersig <- true
	}
	watcher := dhcpmanager.MACPoolWatcher{
		OnPop:  trigger,
		OnPush: trigger,
	}

	// Also watch the Allocations to update the number of
	// bound MAC addresses on changes to Allocation state
	aTrigger := func(a *dhcpmanager.Allocation) {
		watchersig <- true
	}
	aWatcher := dhcpmanager.AllocationWatcher{
		OnCreate: aTrigger,
		OnModify: aTrigger,
		OnDelete: aTrigger,
	}

	serialise := func() {

		response := MACPoolUpdate{
			NumAvailable: 0,
			NumBound:     0,
		}

		// The the number of bound interfaces
		allocs, errA := sm.Allocations()
		if errA != nil {
			response.Status = "error"
			response.Info = err.Error()
			c.WriteJSON(response)
			return
		}

		for _, al := range allocs {
			if al.State == dhcpmanager.Bound {
				response.NumBound = response.NumBound + 1
			}
		}
		macs, errB := sm.MACPool()
		if errB != nil {
			response.Status = "error"
			response.Info = err.Error()
			c.WriteJSON(response)
			return
		}

		response.NumAvailable = len(macs)
		response.Macs = macs
		response.Status = "success"
		c.WriteJSON(response)

	}

	stopWatcher := sm.WatchMACPool(&watcher)
	defer stopWatcher()

	aStopWatcher := sm.Watch(&aWatcher)
	defer aStopWatcher()

	// Send current state to new client
	serialise()

	// Watch etcd changes and push new allocation state
	for {
		select {
		case <-syssig:
			// System signal - terminate gracefully
			return
		case <-watchersig:
			// Allocation state changed - push new list of allocations
			serialise()
		}
	}

}

func addAllocation(w http.ResponseWriter, r *http.Request) {
	data := AllocationRequest{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		res := Response{Status: "error", Info: err.Error()}
		json.NewEncoder(w).Encode(res)
		return
	}

	alloc := dhcpmanager.NewAllocation(data.Hostname)
	sm.Put(alloc)
	json.NewEncoder(w).Encode(alloc)
}

func addMAC(w http.ResponseWriter, r *http.Request) {
	data := AddMACRequest{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		res := Response{Status: "error", Info: err.Error()}
		json.NewEncoder(w).Encode(res)
		return
	}

	var mac net.HardwareAddr
	for _, v := range data.Macs {
		mac, err = net.ParseMAC(v)
		if err != nil {
			res := Response{Status: "error", Info: err.Error()}
			json.NewEncoder(w).Encode(res)
			return
		}
		err = sm.PutMAC(mac)
		if err != nil {
			res := Response{Status: "error", Info: err.Error()}
			json.NewEncoder(w).Encode(res)
			return
		}
	}
	res := Response{Status: "success"}
	json.NewEncoder(w).Encode(res)
	return
}

func processConfiguration() {

	viper.AddConfigPath("/etc/dhcpmanager")
	viper.SetConfigFile("dhcpmanager")

	viper.SetEnvPrefix("DHCP")
	viper.AutomaticEnv()

	viper.SetDefault("etcd", []string{"etcd:2379"})
	viper.SetDefault("ui-port", 8080)
	viper.SetDefault("request-timeout", "10s")

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Configuration error: %s", err.Error())
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Configuration error: %s", err.Error())
	}

}
