package server

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func listenAndServe(t *testing.T) (net.Listener, *http.Server, chan error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("net.Listen failed")
	}
	server := NewServer()
	doneServing := make(chan error)
	go func() {
		doneServing <- server.Serve(l)
	}()
	return l, server, doneServing
}

func subscribeTestEvent(t *testing.T, infocenterPostHandler *infocenterPostHandler, enableTimeout bool) (quitCh chan int) {
	quitCh = make(chan int)
	go func() {
		publishInvocationCount := 0
		subscribeCh := infocenterPostHandler.eventStreamBroker.Subscribe()
		quitCh <- -1
	loop:
		for {
			select {
			case chVal := <-subscribeCh:
				expectedMessage := topicAndMessage{"test-topic", "message text"}
				if chVal != expectedMessage {
					t.Errorf("Channel value %q", chVal)
				}
				publishInvocationCount++
				if publishInvocationCount >= 1 {
					break loop
				}
			case <-time.After(2 * time.Second):
				if enableTimeout {
					break loop
				}
			}
		}
		infocenterPostHandler.eventStreamBroker.Unsubscribe(subscribeCh)
		quitCh <- publishInvocationCount
	}()
	<-quitCh
	return
}

func publishTestEvent(infocenterGetHandler *infocenterGetHandler) {
	infocenterGetHandler.aboutToEnterSelectLoopFunc = func() {
		started := make(chan struct{})
		go func() {
			infocenterGetHandler.eventStreamBroker.Publish(topicAndMessage{"get-topic", "message text"})
			close(started)
		}()
		<-started
	}
}

func stopServing(t *testing.T, server *http.Server, doneServing chan error) {
	_ = server.Shutdown(context.Background())
	serveError := <-doneServing
	if serveError != http.ErrServerClosed {
		t.Fatalf("Server failed with error %q", serveError)
	}
}

func bytesOfBytes(strings ...string) [][]byte {
	bytesOfBytes := make([][]byte, len(strings))
	for i, s := range strings {
		bytesOfBytes[i] = []byte(s)
	}
	return bytesOfBytes
}
