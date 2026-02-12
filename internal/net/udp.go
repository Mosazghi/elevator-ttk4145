// Package network includes udp socket creation, listenting etc.
package network

import (
	"fmt"
	"net"
	"strings"
)

const (
	BroadCastIP   = "255.255.255.255"
	BroadCastPort = 30000
	ChanBufferLen = 20
)

type UDPMessage struct {
	Data    []byte
	Address *net.UDPAddr
}

type Network struct {
	errChan    chan error
	txChan     chan UDPMessage
	rxChan     chan UDPMessage
	conn       net.PacketConn
	filterEcho bool
}

// NewNetwork creates a new network
func NewNetwork(filterEcho bool) (*Network, error) {
	conn, err := CreateSocket(BroadCastPort)
	if err != nil {
		return nil, err
	}

	return &Network{
		conn:       conn,
		rxChan:     make(chan UDPMessage, ChanBufferLen),
		txChan:     make(chan UDPMessage, ChanBufferLen),
		errChan:    make(chan error, ChanBufferLen),
		filterEcho: filterEcho,
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

func (n *Network) TxChan() chan<- UDPMessage {
	return n.txChan
}

func (n *Network) RxChan() <-chan UDPMessage {
	return n.rxChan
}

func (n *Network) ErrChan() <-chan error {
	return n.errChan
}

// receive reads continuously from socket and passes the data to a channel
func (n *Network) receive() {
	buffer := make([]byte, 2048)

	var localAddrStr string
	if n.filterEcho {
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

		// Extract IP from remote address (format is "IP:Port")
		if n.filterEcho {
			remoteIP := strings.Split(remoteAddress.String(), ":")[0]
			if remoteIP == localAddrStr {
				continue
			}
		}

		n.rxChan <- UDPMessage{
			Data:    data,
			Address: remoteAddress.(*net.UDPAddr),
		}
	}
}

// broadcast sends a message to the broadcast address
func (n *Network) broadcast() {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(BroadCastIP),
		Port: BroadCastPort,
	}

	for msg := range n.txChan {
		_, err := n.conn.WriteTo(msg.Data, addr)
		if err != nil {
			n.errChan <- fmt.Errorf("broadcast error: %w", err)
		}
	}
}
