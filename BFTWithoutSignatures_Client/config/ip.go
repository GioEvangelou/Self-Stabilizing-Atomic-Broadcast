package config

import (
	"BFTWithoutSignatures_Client/variables"
	"strconv"
)

// servers' IP adress
var address0 = []string{}
var changingClientsTest = true

var address1 = []string{}

var (
	// ServerAddressesIP - Initialize the address of IP Server sockets
	ServerAddressesIP map[int]string

	// ResponseAddressesIP - Initialize the address of IP Response sockets
	ResponseAddressesIP map[int]string
)

// InitializeIP - Initializes system with ips.
func InitializeIP(clients int) {
	ServerAddressesIP = make(map[int]string, variables.N)
	ResponseAddressesIP = make(map[int]string, variables.N)

	address := address1
	if changingClientsTest {
		address = address0
	}

	for i := 0; i < variables.N; i++ {
		ad := i
		if i >= len(address) {
			ad = (i % 2)
		}

		ServerAddressesIP[i] = "tcp://" + address[ad] + ":" + strconv.Itoa(27250+variables.ID+(i*clients))
		ResponseAddressesIP[i] = "tcp://" + address[ad] + ":" + strconv.Itoa(27625+variables.ID+(i*clients))
	}
}

// GetServerAddress - Returns the IP Server address of the server with that id
func GetServerAddress(id int) string {
	return ServerAddressesIP[id]
}

// GetResponseAddress - Returns the IP Response address of the server with that id
func GetResponseAddress(id int) string {
	return ResponseAddressesIP[id]
}
