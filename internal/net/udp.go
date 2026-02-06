package network

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

const (
	BROADCAST_IP = "255.255.255.255"
	LISTEN_IP    = "0.0.0.0"
	THE_ONE_PORT = 30000
)

type UDPMessage struct {
	Data    []byte
	Address *net.UDPAddr
}

/* Creates a UDP socket with SO_REUSEADDR enabled */
func UDPCreateSocket() (*net.UDPConn, error) {
	// Create raw socket
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("[UDPCreateSocket] Socket creation failed: %v", err)
	}

	// Enable SO_REUSEADDR - allows multiple programs to bind to same port
	err = syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("[UDPCreateSocket] SO_REUSEADDR failed: %v", err)
	}

	// Enable SO_BROADCAST - allows sending to broadcast address
	err = syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	if err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("[UDPCreateSocket] SO_BROADCAST failed: %v", err)
	}

	// Bind to the the common port. addr.Addr = "0.0.0.0" by default
	var addr syscall.SockaddrInet4
	addr.Port = THE_ONE_PORT

	err = syscall.Bind(s, &addr)
	if err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("[UDPCreateSocket] Bind failed: %v", err)
	}

	// Convert to *net.UDPConn
	f := os.NewFile(uintptr(s), "")
	conn, err := net.FileConn(f)
	f.Close()
	if err != nil {
		return nil, fmt.Errorf("[UDPCreateSocket] FileUDPConn failed: %v", err)
	}

	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		return nil, fmt.Errorf("[UDPCreateSocket] Failed to assert connection as *net.UDPConn")
	}

	return udpConn, nil
}

/* Reads continuously from socket and passes the data to a channel */
func UDPrx(connection *net.UDPConn, receiveChannel chan<- UDPMessage, errorChannel chan<- error) {
	buffer := make([]byte, 1024)

	// Get local address to filter out own messages
	localAddr := connection.LocalAddr().(*net.UDPAddr)

	for {
		n, remoteAddress, err := connection.ReadFromUDP(buffer)
		if err != nil {
			errorChannel <- fmt.Errorf("[ReadFromUDP] Read error:", err)
			continue
		}

		// Copy data since buffer is reused
		data := make([]byte, n)
		copy(data, buffer[:n])

		// Filter out messages from same machine (loopback)
		if remoteAddress.IP.Equal(localAddr.IP) && remoteAddress.Port == localAddr.Port {
			continue
		}

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
func UDPRunNetwork() (chan<- UDPMessage, <-chan UDPMessage, <-chan error, error) {

	// Initialize channels
	rxChan := make(chan UDPMessage, 1024)
	txChan := make(chan UDPMessage, 1024)
	errChan := make(chan error, 1024)

	// Create single socket for both send and receive
	conn, err := UDPCreateSocket()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to create socket: %v", err)
	}

	// Spawn rx goroutine
	go UDPrx(conn, rxChan, errChan)

	// Define broadcast address
	broadcastAddr := &net.UDPAddr{
		IP:   net.ParseIP(BROADCAST_IP),
		Port: THE_ONE_PORT,
	}

	// tx goroutine that broadcasts messages
	go func() {
		for msg := range txChan {
			_, err := conn.WriteToUDP(msg.Data, broadcastAddr)
			if err != nil {
				errChan <- fmt.Errorf("Write error: %v", err)
			}
		}
	}()

	return txChan, rxChan, errChan, nil
}
