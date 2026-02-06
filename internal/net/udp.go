package network

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"strings"
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

/* Creates a UDP socket with SO_REUSEADDR, SO_BROADCAST enabled.
Allows multiple programs to bind to the same port. */
func UDPCreateSocket() (*net.UDPConn, error) {

	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("[UDPCreateSocket] Socket creation failed: %v", err)
	}

	err = syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("[UDPCreateSocket] SO_REUSEADDR failed: %v", err)
	}

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

	localAddrStr, _ := LocalIP()
	localAddr := net.ParseIP(localAddrStr)

	for {
		n, remoteAddress, err := connection.ReadFromUDP(buffer)
		if err != nil {
			errorChannel <- fmt.Errorf("[ReadFromUDP] Read error:%v", err)
			continue
		}

		data := make([]byte, n)
		copy(data, buffer[:n])

		if remoteAddress.IP.Equal(localAddr) {
			//fmt.Println("[UDPrx] Echo spotted") // For testing purposes
			continue
		}

		receiveChannel <- UDPMessage{
			Data:    data,
			Address: remoteAddress,
		}
	}
}

/* Initializes & runs the UDP network. */
func UDPRunNetwork() (chan<- UDPMessage, <-chan UDPMessage, <-chan error, error) {


	rxChan := make(chan UDPMessage, 20)
	txChan := make(chan UDPMessage, 20)
	errChan := make(chan error, 10)


	conn, err := UDPCreateSocket()
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
			_, err := conn.WriteToUDP(msg.Data, broadcastAddr)
			if err != nil {
				errChan <- fmt.Errorf("Write error: %v", err)
			}
		}
	}()

	return txChan, rxChan, errChan, nil
}
