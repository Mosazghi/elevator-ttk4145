package network

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
)

var localIP string

// GetLocalIP returns the machine's local IP address, cached after first lookup.
func GetLocalIP() (string, error) {
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

// CreateSocket creates a UDP socket with SO_REUSEADDR and SO_BROADCAST,
// allowing multiple programs to bind to the same port.
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
