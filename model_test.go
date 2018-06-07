package dhcpmanager

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/google/uuid"
)

func TestMashallingInterface(t *testing.T) {

	macString := "aa:bb:cc:dd:ee:ff"
	mac, _ := net.ParseMAC(macString)

	iface := net.Interface{
		Name:         "test",
		MTU:          1,
		Index:        2,
		HardwareAddr: mac,
		Flags:        0,
	}

	data, err := json.Marshal((InterfaceAlias)(iface))
	if err != nil {
		t.Error(err)
	} else {
		t.Log("Marshal returned without error")
	}

	t.Log(string(data))

	alias := InterfaceAlias{}
	err = json.Unmarshal(data, &alias)
	if err != nil {
		t.Error(err)
	} else {
		t.Log("Unmarshal returned without error")
	}
	t.Log(alias)

	if mac.String() != alias.HardwareAddr.String() {
		t.Errorf("Expected [%s] got [%s]", mac.String(), alias.HardwareAddr.String())
	}

}

func TestMashallingAllocation(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	alloc := &Allocation{
		ID: uuid.New(),
		Interface: net.Interface{
			Index:        0,
			MTU:          1,
			Flags:        4,
			Name:         "test",
			HardwareAddr: mac,
		},
	}

	data, err := json.Marshal(alloc)
	if err != nil {
		t.Error(err)
	} else {
		t.Log("Marshal returned without error")
	}
	t.Log(string(data))

	alloc2 := &Allocation{}
	err = json.Unmarshal(data, alloc2)
	if err != nil {
		t.Error(err)
	} else {
		t.Log("Unmarshal returned without error")
	}

	t.Log(alloc2)

	if alloc.Interface.Index != alloc2.Interface.Index {
		t.Errorf("Field mismatch [Index]: %d / %d", alloc.Interface.Index, alloc2.Interface.Index)
	}

	if alloc.Interface.MTU != alloc2.Interface.MTU {
		t.Errorf("Field mismatch [MTU]: %d / %d", alloc.Interface.MTU, alloc2.Interface.MTU)
	}

	if alloc.Interface.Flags != alloc2.Interface.Flags {
		t.Errorf("Field mismatch [Flags]: %d / %d", alloc.Interface.Flags, alloc2.Interface.Flags)
	}

	if alloc.Interface.Name != alloc2.Interface.Name {
		t.Errorf("Field mismatch [Name]: %s / %s", alloc.Interface.Name, alloc2.Interface.Name)
	}

	if alloc.Interface.HardwareAddr.String() != alloc2.Interface.HardwareAddr.String() {
		t.Errorf("Field mismatch [HardwareAddr]: %s / %s", alloc.Interface.HardwareAddr.String(), alloc2.Interface.HardwareAddr.String())
	}

}
