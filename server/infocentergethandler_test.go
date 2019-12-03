package server

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

type testGetResponseWriterExpectations struct {
	writeHeader            http.Header
	writeHeaderInvocations int
	writeFlushInvocations  int
	writeInvocations       [][]byte
}

type testGetResponseWriter struct {
	t                  *testing.T
	expectedStatusCode int
	e                  *testGetResponseWriterExpectations
	wroteBytes         func()
}

func (w testGetResponseWriter) Header() http.Header {
	return w.e.writeHeader
}

func (w testGetResponseWriter) Write(bytes []byte) (int, error) {
	w.e.writeInvocations = append(w.e.writeInvocations, bytes)
	if w.wroteBytes != nil {
		w.wroteBytes()
	}
	return 0, nil
}

func (w testGetResponseWriter) WriteHeader(statusCode int) {
	if statusCode != w.expectedStatusCode {
		w.t.Fatalf("Unexpected status code %d", statusCode)
	}
	w.e.writeHeaderInvocations++
}

func (w testGetResponseWriter) Flush() {
	w.e.writeFlushInvocations++
}

func TestNoMessageCancelInfocenterGetHandler_ServeHTTP(t *testing.T) {
	infocenterGetHandler, writer, requestCancel, request := mockGetRequestHandler(t, "get-topic")
	requestCancel()

	infocenterGetHandler.ServeHTTP(writer, request)

	assertResponseHeaders(t, writer)
	if writer.e.writeFlushInvocations != 1 {
		t.Fatalf("Unexpected write flush invocation count %d", writer.e.writeFlushInvocations)
	}
	if len(writer.e.writeInvocations) != 0 {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestSameTopicCancelInfocenterGetHandler_ServeHTTP(t *testing.T) {
	infocenterGetHandler, writer, requestCancel, request := mockGetRequestHandler(t, "get-topic")
	publishTestEvent(&infocenterGetHandler, nil)
	writer.wroteBytes = func() {
		requestCancel()
	}

	infocenterGetHandler.ServeHTTP(writer, request)

	assertResponseHeaders(t, writer)
	if writer.e.writeFlushInvocations != 2 {
		t.Fatalf("Unexpected write flush invocation count %d", writer.e.writeFlushInvocations)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations,
		bytesOfBytes("id: 1\n", "event: msg\n", "data: message text\n", "\n")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestDifferentTopicTimeoutInfocenterGetHandler_ServeHTTP(t *testing.T) {
	const eventStreamTimeoutSeconds = 2
	const eventStreamTimeoutData = "data: 2s\n"
	savedEventStreamTimeoutSeconds := EventStreamTimeoutSeconds
	EventStreamTimeoutSeconds = eventStreamTimeoutSeconds
	defer func() {
		EventStreamTimeoutSeconds = savedEventStreamTimeoutSeconds
	}()
	infocenterGetHandler, writer, _, request := mockGetRequestHandler(t, "different topic")
	publishTestEvent(&infocenterGetHandler, nil)

	infocenterGetHandler.ServeHTTP(writer, request)

	assertResponseHeaders(t, writer)
	if writer.e.writeFlushInvocations != 2 {
		t.Fatalf("Unexpected write flush invocation count %d", writer.e.writeFlushInvocations)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations,
		bytesOfBytes("id: 1\n", "event: timeout\n", eventStreamTimeoutData, "\n")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestTwoMessageTimeoutInfocenterGetHandler_ServeHTTP(t *testing.T) {
	const eventStreamTimeoutSeconds = 2
	const eventStreamTimeoutData = "data: 2s\n"
	savedEventStreamTimeoutSeconds := EventStreamTimeoutSeconds
	EventStreamTimeoutSeconds = eventStreamTimeoutSeconds
	defer func() {
		EventStreamTimeoutSeconds = savedEventStreamTimeoutSeconds
	}()
	infocenterGetHandler, writer, requestCancel, request := mockGetRequestHandler(t, "get-topic")
	stopPublishingCh := make(chan struct{})
	publishTestEvent(&infocenterGetHandler, func(defaultPublishFunc func()) {
		defaultPublishFunc()
		for i := 0; i < eventStreamTimeoutSeconds*10; i++ {
			select {
			case <-time.After(time.Second):
				infocenterGetHandler.eventStreamBroker.Publish(topicAndMessage{"get-topic", "message text"})
			case <-stopPublishingCh:
				return
			}
		}
		requestCancel()
		t.Fatal("Publish loop never stopped")
	})

	infocenterGetHandler.ServeHTTP(writer, request)
	close(stopPublishingCh)

	assertResponseHeaders(t, writer)
	if writer.e.writeFlushInvocations < 2 {
		t.Fatalf("Unexpected write flush invocation count %d", writer.e.writeFlushInvocations)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations[len(writer.e.writeInvocations)-3:],
		bytesOfBytes("event: timeout\n", eventStreamTimeoutData, "\n")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}
