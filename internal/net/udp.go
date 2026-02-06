package network

import (
	"encoding/json"
	"fmt"
	_ "log"
	"net"
	"os"
	_ "time"
)

const (
	BROADCAST_IP = "255.255.255.255" // Enables SO_BROADCAST
	LISTEN_IP = "0.0.0.0"
)

type UDPMessage struct {
	Data    []byte
	Address *net.UDPAddr
}

type NodeConfig struct {
	ID			string	`json:"id"`
	ListenPorts	[]int	`json:"listen_ports"`
	BroadcastPort int	`json:"broadcast_port"`
}


/* Creates a UDP socket*/
func UDPCreateSocket(localIP string, localPort int) (*net.UDPConn, error) {
	
	// Default local address
	localAddress := &net.UDPAddr{
		IP:   net.ParseIP(localIP),
		Port: localPort,
	}

	conn, err := net.ListenUDP("udp", localAddress)
	if err != nil {
		return nil, fmt.Errorf("[UDPCreateSocket] Failed to create socket: %v", err)
	}

	return conn, nil
}

/* Reads continously from socket and passes the data to a channel */
func UDPrx(connection *net.UDPConn, receiveChannel chan<- UDPMessage, errorChannel chan<- error) {
	buffer := make([]byte, 1024) // Required buffer size TBD

	for {

		n, remoteAddress, err := connection.ReadFromUDP(buffer)
		if err != nil {
			errorChannel <- fmt.Errorf("Read error: %v", err)
			continue
		}

		// Copy data since buffer is reused
		data := make([]byte, n)
		copy(data, buffer[:n])

		receiveChannel <- UDPMessage{
			Data:    data,
			Address: remoteAddress,
		}
	}
}

// /* UDP transmission */
// func UDPtx(connection *net.UDPConn, transmitChannel <-chan UDPMessage, errorChannel chan<- error) {
// 	for msg := range transmitChannel {
// 		_, err := connection.WriteToUDP(msg.Data, msg.Address)
// 		if err != nil {
// 			errorChannel <- fmt.Errorf("[UDP] Write error: %v", err)
// 		}
// 	}
// }

/* Function to call from main */
func UDPRunNetwork(ID string) (chan<- UDPMessage, <-chan UDPMessage, <-chan error, error) {

	// Read config file
	fileData, err := os.ReadFile("internal/net/network_config.json")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("[UDPRunNetwork] Failed to read config file: %v", err)
	}

	// Parse config file
	var configs map[string]NodeConfig
	err = json.Unmarshal(fileData, &configs)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("[UDPRunNetwork] Failed to parse JSON: %v", err)
	}

	// Get configuration for current node
	nodeConfig, exists := configs[ID]
	if !exists {
		return nil, nil, nil, fmt.Errorf("[UDPRunNetwork] No configuration found for ID: %s", ID)
	}

	// Create broadcast socket 
	broadcastConn, _ := UDPCreateSocket(LISTEN_IP, nodeConfig.BroadcastPort)
	//defer broadcastConn.Close()

	// Initialize channels
	rxChan := make(chan UDPMessage, 1024)
	txChan := make(chan UDPMessage, 1024)
	errChan := make(chan error, 1024)

	// Create listening sockets for each port
	for _, port := range nodeConfig.ListenPorts {
		listenConn, err := UDPCreateSocket(LISTEN_IP, port)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("[UDPRunNetwork] Failed to create tx socket on port %d: %v", port, err)
		}
		//defer listenConn.Close()

		go UDPrx(listenConn, rxChan, errChan)
	}
	
	// Define broadcast address
    broadcastAddr := &net.UDPAddr{
        IP:   net.ParseIP(BROADCAST_IP),
        Port: nodeConfig.BroadcastPort,
    }

	// tx goroutine that wraps messages with broadcast address
	go func() {
		for msg := range txChan {
			msg.Address = broadcastAddr
			_, err := broadcastConn.WriteToUDP(msg.Data, msg.Address)
			if err != nil {
				errChan <- fmt.Errorf("[UDP] write error: %v", err)
			}
		}
	}()

    return txChan, rxChan, errChan, nil
}
