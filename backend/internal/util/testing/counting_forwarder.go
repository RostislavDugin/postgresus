package testing

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

// CountingForwarder relays TCP connections to a target address and counts them, so a test can see
// how many connections a client opened instead of inferring it from how long the calls took.
type CountingForwarder struct {
	Host string
	Port int

	listener      net.Listener
	targetAddress string

	mutex         sync.Mutex
	acceptedCount int
	isClosed      bool
	connections   []net.Conn
}

func StartCountingForwarder(t *testing.T, targetAddress string) *CountingForwarder {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for counting forwarder: %v", err)
	}

	listenerAddress, isTCPAddress := listener.Addr().(*net.TCPAddr)
	if !isTCPAddress {
		t.Fatalf("counting forwarder listens on a non-TCP address %s", listener.Addr())
	}

	forwarder := &CountingForwarder{
		Host:          listenerAddress.IP.String(),
		Port:          listenerAddress.Port,
		listener:      listener,
		targetAddress: targetAddress,
	}

	go forwarder.acceptConnections()
	t.Cleanup(forwarder.close)

	return forwarder
}

func (f *CountingForwarder) GetAcceptedCount() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.acceptedCount
}

func (f *CountingForwarder) acceptConnections() {
	for {
		clientConnection, err := f.listener.Accept()
		if err != nil {
			return
		}

		f.mutex.Lock()
		f.acceptedCount++
		f.mutex.Unlock()

		go f.forward(clientConnection)
	}
}

func (f *CountingForwarder) forward(clientConnection net.Conn) {
	targetConnection, err := (&net.Dialer{Timeout: 5 * time.Second}).Dial("tcp", f.targetAddress)
	if err != nil {
		_ = clientConnection.Close()
		return
	}

	// A connection accepted while the forwarder closes would otherwise outlive the test.
	if !f.track(clientConnection, targetConnection) {
		_ = clientConnection.Close()
		_ = targetConnection.Close()

		return
	}

	go pipeAndClose(targetConnection, clientConnection)
	go pipeAndClose(clientConnection, targetConnection)
}

func (f *CountingForwarder) track(connections ...net.Conn) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.isClosed {
		return false
	}

	f.connections = append(f.connections, connections...)

	return true
}

// Clients such as the storage session pools keep idle connections open past the test, so the
// forwarder closes them itself rather than leave its relay goroutines running.
func (f *CountingForwarder) close() {
	_ = f.listener.Close()

	f.mutex.Lock()
	f.isClosed = true
	connections := f.connections
	f.connections = nil
	f.mutex.Unlock()

	for _, connection := range connections {
		_ = connection.Close()
	}
}
