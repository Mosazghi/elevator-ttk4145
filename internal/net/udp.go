// Package network includes udp socket creation, listenting etc.
package network

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
)

const (
	BroadCastIP   = "255.255.255.255"
	BroadCastPort = 30000
)

type UDPMessage struct {
	Data    []byte
	Address *net.UDPAddr
}

// CreateSocket creates a UDP socket with SO_REUSEADDR, SO_BROADCAST enabled.
// Allows multiple programs to bind to the same port.
func CreateSocket(port int) (net.PacketConn, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("socket error: %w", err)
	}
	err = syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return nil, fmt.Errorf("setsockopt REUSEADDR error: %w", err)
	}
	err = syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	if err != nil {
		return nil, fmt.Errorf("setsockopt BROADCAST error: %w", err)
	}
	err = syscall.Bind(s, &syscall.SockaddrInet4{Port: port})
	if err != nil {
		return nil, fmt.Errorf("bind error: %w", err)
	}

	f := os.NewFile(uintptr(s), "")
	defer f.Close()

	conn, err := net.FilePacketConn(f)
	if err != nil {
		return nil, fmt.Errorf("FilePacketConn error: %w", err)
	}

	return conn, nil
}

// receive reads continuously from socket and passes the data to a channel
// filterEcho: if true, filters out messages from the local IP (production mode)
func receive(connection net.PacketConn, receiveChannel chan<- UDPMessage, errorChannel chan<- error, filterEcho bool) {
	buffer := make([]byte, 2048)

	var localAddrStr string
	if filterEcho {
		localAddrStr, _ = LocalIP()
	}

	for {
		n, remoteAddress, err := connection.ReadFrom(buffer[0:])
		if err != nil {
			errorChannel <- err
			continue
		}

		data := make([]byte, n)
		copy(data, buffer[:n])

		// Extract IP from remote address (format is "IP:Port")
		if filterEcho {
			remoteIP := strings.Split(remoteAddress.String(), ":")[0]
			if remoteIP == localAddrStr {
				continue
			}
		}

		receiveChannel <- UDPMessage{
			Data:    data,
			Address: remoteAddress.(*net.UDPAddr),
		}
	}
}

// broadcast sends a message to the broadcast address
func broadcast(txChan <-chan UDPMessage, errChan chan<- error, conn net.PacketConn, addr *net.UDPAddr) {
	for msg := range txChan {
		_, err := conn.WriteTo(msg.Data, addr)
		if err != nil {
			errChan <- fmt.Errorf("Broadcast error: %w", err)
		}
	}
}

// Start initializes & runs the UDP network.
// prodMode: if true, filters echo messages (messages from local IP)
func Start(prodMode bool) (chan<- UDPMessage, <-chan UDPMessage, <-chan error, error) {
	rxChan := make(chan UDPMessage, 20)
	txChan := make(chan UDPMessage, 20)
	errChan := make(chan error, 10)

	conn, err := CreateSocket(BroadCastPort)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create socket: %w", err)
	}

	broadcastAddr := &net.UDPAddr{
		IP:   net.ParseIP(BroadCastIP),
		Port: BroadCastPort,
	}

	go receive(conn, rxChan, errChan, prodMode)

	go broadcast(txChan, errChan, conn, broadcastAddr)

	return txChan, rxChan, errChan, nil
}
