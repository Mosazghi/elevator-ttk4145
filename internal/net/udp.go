package network

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
)

const (
	BROADCAST_IP = "255.255.255.255"
	THE_ONE_PORT = 30000
)

type UDPMessage struct {
	Data    []byte
	Address *net.UDPAddr
}

var localIP string

func LocalIP() (string, error) {
	if localIP == "" {
		conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: []byte{8, 8, 8, 8}, Port: 53})
		if err != nil {
			return "", err
		}
		defer conn.Close()
		localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
	}
	return localIP, nil
}

/*
	Creates a UDP socket with SO_REUSEADDR, SO_BROADCAST enabled.

Allows multiple programs to bind to the same port.
*/
func UDPCreateSocket(port int) (net.PacketConn, error) {
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
	conn, err := net.FilePacketConn(f)
	if err != nil {
		return nil, fmt.Errorf("FilePacketConn error: %w", err)
	}
	f.Close()

	return conn, nil
} /* Reads continuously from socket and passes the data to a channel */
func UDPrx(connection net.PacketConn, receiveChannel chan<- UDPMessage, errorChannel chan<- error) {
	buffer := make([]byte, 2048)

	// localAddrStr, _ := LocalIP()

	for {
		n, remoteAddress, err := connection.ReadFrom(buffer[0:])
		if err != nil {
			errorChannel <- fmt.Errorf("[ReadFromUDP] Read error:%v", err)
			continue
		}

		data := make([]byte, n)
		copy(data, buffer[:n])

		// Extract IP from remote address (format is "IP:Port")
		// remoteIP := strings.Split(remoteAddress.String(), ":")[0]
		// if remoteIP == localAddrStr {
		// 	// fmt.Println("[UDPrx] Echo spotted") // For testing purposes
		// 	continue
		// }

		receiveChannel <- UDPMessage{
			Data:    data,
			Address: remoteAddress.(*net.UDPAddr),
		}
	}
}

/* Initializes & runs the UDP network. */
func UDPRunNetwork() (chan<- UDPMessage, <-chan UDPMessage, <-chan error, error) {
	rxChan := make(chan UDPMessage, 20)
	txChan := make(chan UDPMessage, 20)
	errChan := make(chan error, 10)

	conn, err := UDPCreateSocket(THE_ONE_PORT)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to create socket: %v", err)
	}

	go UDPrx(conn, rxChan, errChan)

	broadcastAddr := &net.UDPAddr{
		IP:   net.ParseIP(BROADCAST_IP),
		Port: THE_ONE_PORT,
	}

	// Broadcast
	go func() {
		for msg := range txChan {
			_, err := conn.WriteTo(msg.Data, broadcastAddr)
			if err != nil {
				errChan <- fmt.Errorf("Write error: %v", err)
			}
		}
	}()

	return txChan, rxChan, errChan, nil
}
