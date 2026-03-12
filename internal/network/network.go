// Package network includes udp socket creation, listenting etc.
package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/Mosazghi/elevator-ttk4145/internal/config"
)

type DataPacket []byte

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

// Close closes the transmitt channel and possibly closes the net. connection
func (n *Network) Close() error {
	close(n.txChan)
	if n.conn != nil {
		return n.conn.Close()
	}

	return nil
}

func (n *Network) Start() {
	go n.receive()
	go n.broadcast()
}

func (n *Network) TxChan() chan<- DataPacket {
	return n.txChan
}

func (n *Network) RxChan() <-chan DataPacket {
	return n.rxChan
}

func (n *Network) ErrChan() <-chan error {
	return n.errChan
}

// receive reads continuously from socket and passes the data to a channel
func (n *Network) receive() {
	buffer := make([]byte, 2048)

	filterEcho := config.ProdMode

	var localAddrStr string
	if filterEcho {
		localAddrStr, _ = LocalIP()
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

// broadcast sends a message to the broadcast address
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
