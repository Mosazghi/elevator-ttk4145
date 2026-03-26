// Package network provides UDP broadcast communication for elevator coordination.
package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/Mosazghi/elevator-ttk4145/internal/config"
)

// DataPacket represents network message data.
type DataPacket []byte

// Network manages UDP socket communication with separate TX/RX/error channels.
type Network struct {
	errorChan    chan error
	transmitChan chan DataPacket
	receiveChan  chan DataPacket
	socket       net.PacketConn
}

// NewNetwork creates a new network
func NewNetwork() (*Network, error) {
	socket, err := CreateSocket(broadcastPort)
	if err != nil {
		return nil, err
	}

	return &Network{
		socket:       socket,
		receiveChan:  make(chan DataPacket, networkChannelBufferLength),
		transmitChan: make(chan DataPacket, networkChannelBufferLength),
		errorChan:    make(chan error, networkChannelBufferLength),
	}, nil
}

// Close closes the transmit channel and terminates the network connection.
func (n *Network) Close() error {
	close(n.transmitChan)
	if n.socket != nil {
		return n.socket.Close()
	}

	return nil
}

// Start launches receive and broadcast goroutines.
func (n *Network) Start() {
	go n.receive()
	go n.broadcast()
}

func (n *Network) GetTransmitChannel() chan<- DataPacket {
	return n.transmitChan
}

func (n *Network) GetReceiveChannel() <-chan DataPacket {
	return n.receiveChan
}

func (n *Network) GetErrorChannel() <-chan error {
	return n.errorChan
}

// receive reads from socket and forwards packets to rxChan, filtering echoes in production.
func (n *Network) receive() {
	buffer := make([]byte, 2048)

	filterEcho := config.IsProdMode()

	var localAddrStr string
	if filterEcho {
		localAddrStr, _ = GetLocalIP()
	}

	for {
		bytesRead, remoteAddress, err := n.socket.ReadFrom(buffer[0:])
		if err != nil {
			n.errorChan <- fmt.Errorf("receive error: %w", err)
			continue
		}

		data := make([]byte, bytesRead)
		copy(data, buffer[:bytesRead])

		if filterEcho {
			remoteIP := strings.Split(remoteAddress.String(), ":")[0]
			if remoteIP == localAddrStr {
				continue
			}
		}

		n.receiveChan <- data
	}
}

// broadcast transmits packets from txChan to the broadcast address.
func (n *Network) broadcast() {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(broadcastIP),
		Port: broadcastPort,
	}

	for data := range n.transmitChan {
		_, err := n.socket.WriteTo(data, addr)
		if err != nil {
			n.errorChan <- fmt.Errorf("broadcast error: %w", err)
		}
	}
}
