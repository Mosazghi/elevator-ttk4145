package network

import (
	"testing"
	"net"
)


func TestUDNetworkRun(t *testing.T){
	response := false
	if response == false {

		t.Error("No response")
		
	}
}

func TestUDPNetworkInit(t *testing.T) {
	send, receive, _, CleanUp, _ := UDPNetworkInit("10.0.0.20", 8080)
	defer CleanUp()

	// Test by sending a valid UDPMessage 
	testMsg := UDPMessage{
		Data:    []byte("hello, UDP!"),
		Address: &net.UDPAddr{IP: net.ParseIP("10.0.0.20"), Port: 8080},
	}	
	send <- testMsg
	receivedMsg := <-receive

	// Add an assertion to check the received message?
	if receivedMsg.Data != nil {
		return
	} else {
		t.Errorf("Expected %v, got %v", testMsg, receivedMsg)}
}