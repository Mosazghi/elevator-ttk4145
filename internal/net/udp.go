package net

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type UDPMessage struct {
	Data    []byte
	Address *net.UDPAddr
}

/* Creates an unconnected UDP socket*/
func CreateUDPSocket(localIP string, localPort int) (*net.UDPConn, error) {
	// Default local address
	localAddress := &net.UDPAddr{
		IP:   net.ParseIP(localIP),
		Port: localPort,
	}

	conn, err := net.ListenUDP("udp", localAddress)
	if err != nil {
		return nil, fmt.Errorf("Failed to create UDP socket: %v", err)
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

/* UDP transmission */
func UDPtx(connection *net.UDPConn, transmitChannel <-chan UDPMessage, errorChannel chan<- error) {
	for msg := range transmitChannel {
		_, err := connection.WriteToUDP(msg.Data, msg.Address)
		if err != nil {
			errorChannel <- fmt.Errorf("Write error: %v", err)
		}
	}
}

/* Creates channels, sockets and Goroutines to initiate the UDP network */
func InitUDPNetwork(localIP string, localPort int) (
	transmitChannel chan<- UDPMessage,
	receiveChannel <-chan UDPMessage,
	errorChannel <-chan error,
	cleanup func(),
	err error,
) {

	// Create socket
	conn, err := CreateUDPSocket(localIP, localPort)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Create channels (Buffered, to prevent blocking)
	send := make(chan UDPMessage, 100)
	receive := make(chan UDPMessage, 100)
	errors := make(chan error, 10)

	// Start Goroutines
	go UDPrx(conn, receive, errors)
	go UDPtx(conn, send, errors)

	// Cleanup function to properly close channels
	CleanUp := func() {
		close(send)
		conn.Close()
		close(receive)
		close(errors)
	}
	return send, receive, errors, CleanUp, nil
}


/* Copy pasta test with some edits
TODO: Make operational
*/
func UDP3WayTest() {

	// Parse command line arguments
	localPort := flag.Int("port", 8080, "Local UDP port to listen on")
	localIP := flag.String("ip", "0.0.0.0", "Local IP address to bind to")
	peer1IP := flag.String("peer1", "", "First peer IP address (required)")
	peer2IP := flag.String("peer2", "", "Second peer IP address (required)")
	peerPort := flag.Int("peerport", 8080, "Port that peers are listening on")
	duration := flag.Int("duration", 30, "Test duration in seconds")

	flag.Parse()

	// Validate required arguments
	if *peer1IP == "" || *peer2IP == "" {
		log.Fatal("Error: Both -peer1 and -peer2 IP addresses are required")
	}

	log.Printf("Starting UDP test on %s:%d", *localIP, *localPort)
	log.Printf("Peers: %s:%d and %s:%d", *peer1IP, *peerPort, *peer2IP, *peerPort)

	// Start UDP communication
	sendChan, receiveChan, errorChan, cleanup, err := InitUDPNetwork(*localIP, *localPort)
	if err != nil {
		log.Fatalf("Failed to start UDP: %v", err)
	}
	defer cleanup()

	// Define peer addresses
	peer1Addr := &net.UDPAddr{
		IP:   net.ParseIP(*peer1IP),
		Port: *peerPort,
	}
	peer2Addr := &net.UDPAddr{
		IP:   net.ParseIP(*peer2IP),
		Port: *peerPort,
	}

	log.Println("UDP socket ready, listening for messages...")

	// Handle incoming messages
	go func() {
		for msg := range receiveChan {
			log.Printf("RECEIVED from %s: %s", msg.Address, string(msg.Data))

			// ILLEGAL: Auto-reply to sender
			reply := fmt.Sprintf("ACK from %s:%d", *localIP, *localPort)
			sendChan <- UDPMessage{
				Data:    []byte(reply),
				Address: msg.Address,
			}
		}
	}()

	// Handle errors
	go func() {
		for err := range errorChan {
			log.Printf("ERROR: %v", err)
		}
	}()

	// Get local hostname for identification
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Send periodic messages to both peers
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial announcement
	announcement := fmt.Sprintf("Hello from %s (%s:%d)", hostname, *localIP, *localPort)
	sendChan <- UDPMessage{
		Data:    []byte(announcement),
		Address: peer1Addr,
	}
	sendChan <- UDPMessage{
		Data:    []byte(announcement),
		Address: peer2Addr,
	}

	log.Println("Sent initial announcements to both peers")

	// Message counter
	messageCount := 0

	// Run test for specified duration
	timeout := time.After(time.Duration(*duration) * time.Second)

	for {
		select {
		case <-ticker.C:
			messageCount++
			msg := fmt.Sprintf("Message #%d from %s", messageCount, hostname)

			// Transmit to both peers
			log.Printf("SENDING to peer1 (%s): %s", peer1Addr, msg)
			sendChan <- UDPMessage{
				Data:    []byte(msg),
				Address: peer1Addr,
			}
			log.Printf("SENDING to peer2 (%s): %s", peer2Addr, msg)
			sendChan <- UDPMessage{
				Data:    []byte(msg),
				Address: peer2Addr,
			}

		case <-timeout:
			log.Printf("\nTest completed after %d seconds", *duration)
			log.Printf("Sent %d messages total", messageCount)
			return
		}
	}
}
