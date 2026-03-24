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
	errChan chan error
	txChan  chan DataPacket
	rxChan  chan DataPacket
	conn    net.PacketConn
}

// NewNetwork creates a new network
func NewNetwork() (*Network, error) {
	conn, err := CreateSocket(BroadcastPort)
	if err != nil {
		return nil, err
	}

	return &Network{
		conn:    conn,
		rxChan:  make(chan DataPacket, NetChanBufferLen),
		txChan:  make(chan DataPacket, NetChanBufferLen),
		errChan: make(chan error, NetChanBufferLen),
	}, nil
}

// Close closes the TX channel and the network connection.
func (n *Network) Close() error {
	close(n.txChan)
	if n.conn != nil {
		return n.conn.Close()
	}

	return nil
}

// Start launches receive and broadcast goroutines.
func (n *Network) Start() {
	go n.receive()
	go n.broadcast()
}

// TxChan returns the transmit channel.
func (n *Network) TxChan() chan<- DataPacket {
	return n.txChan
}

// RxChan returns the receive channel.
func (n *Network) RxChan() <-chan DataPacket {
	return n.rxChan
}

// ErrChan returns the error channel.
func (n *Network) ErrChan() <-chan error {
	return n.errChan
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
		bytesRead, remoteAddress, err := n.conn.ReadFrom(buffer[0:])
		if err != nil {
			n.errChan <- fmt.Errorf("receive error: %w", err)
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

		n.rxChan <- data
	}
}

// broadcast transmits packets from txChan to the broadcast address.
func (n *Network) broadcast() {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(BroadcastIP),
		Port: BroadcastPort,
	}

	for data := range n.txChan {
		_, err := n.conn.WriteTo(data, addr)
		if err != nil {
			n.errChan <- fmt.Errorf("broadcast error: %w", err)
		}
	}
}
