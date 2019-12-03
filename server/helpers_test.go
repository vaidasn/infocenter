package server

import (
	"context"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"reflect"
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
	const subscribeTimeout = 2 * time.Second
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
			case <-time.After(subscribeTimeout):
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

func publishTestEvent(infocenterGetHandler *infocenterGetHandler, customPublishFunc func(defaultPublishFunc func())) {
	infocenterGetHandler.aboutToEnterSelectLoopFunc = func() {
		started := make(chan struct{})
		publishFunc := func() {
			infocenterGetHandler.eventStreamBroker.Publish(topicAndMessage{"get-topic", "message text"})
			close(started)
		}
		if customPublishFunc != nil {
			go customPublishFunc(publishFunc)
		} else {
			go publishFunc()
		}
		<-started
	}
}

func mockGetRequestHandler(t *testing.T, topic string) (infocenterGetHandler, testGetResponseWriter,
	context.CancelFunc, *http.Request) {
	infocenterGetHandler := infocenterGetHandler{eventStreamBroker: newEventStreamBroker()}
	writer := testGetResponseWriter{
		t:                  t,
		expectedStatusCode: http.StatusOK,
		e:                  &testGetResponseWriterExpectations{writeHeader: http.Header{}},
	}
	requestContext, requestCancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(requestContext, "GET",
		"http://localhost/infocenter/test-topic", http.NoBody)
	if err != nil {
		t.Fatalf("Got error while creating new request: %q", err)
	}
	request = mux.SetURLVars(request, map[string]string{"topic": topic})
	return infocenterGetHandler, writer, requestCancel, request
}

func assertResponseHeaders(t *testing.T, writer testGetResponseWriter) {
	if !reflect.DeepEqual(writer.e.writeHeader,
		http.Header{"Cache-Control": {"no-cache"}, "Content-Type": {"text/event-stream"}}) {
		t.Fatalf("Unexpected write headers %q", writer.e.writeHeader)
	}
	if writer.e.writeHeaderInvocations != 1 {
		t.Fatalf("Unexpected write header invocation count %d", writer.e.writeHeaderInvocations)
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
