package server

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
	"testing"
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
		w.t.Errorf("Unexpected status code %d", statusCode)
		panic(w)
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

	if !reflect.DeepEqual(writer.e.writeHeader,
		http.Header{"Cache-Control": {"no-cache"}, "Content-Type": {"text/event-stream"}}) {
		t.Fatalf("Unexpected write headers %q", writer.e.writeHeader)
	}
	if writer.e.writeHeaderInvocations != 1 {
		t.Fatalf("Unexpected write header invocation count %d", writer.e.writeHeaderInvocations)
	}
	if writer.e.writeFlushInvocations != 1 {
		t.Fatalf("Unexpected write flush invocation count %d", writer.e.writeFlushInvocations)
	}
	if len(writer.e.writeInvocations) != 0 {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestSameTopicCancelInfocenterGetHandler_ServeHTTP(t *testing.T) {
	infocenterGetHandler, writer, requestCancel, request := mockGetRequestHandler(t, "get-topic")
	publishTestEvent(&infocenterGetHandler)
	writer.wroteBytes = func() {
		requestCancel()
	}

	infocenterGetHandler.ServeHTTP(writer, request)

	if !reflect.DeepEqual(writer.e.writeHeader,
		http.Header{"Cache-Control": {"no-cache"}, "Content-Type": {"text/event-stream"}}) {
		t.Fatalf("Unexpected write headers %q", writer.e.writeHeader)
	}
	if writer.e.writeHeaderInvocations != 1 {
		t.Fatalf("Unexpected write header invocation count %d", writer.e.writeHeaderInvocations)
	}
	if writer.e.writeFlushInvocations != 2 {
		t.Fatalf("Unexpected write flush invocation count %d", writer.e.writeFlushInvocations)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations,
		bytesOfBytes("event: msg\n", "data: message text\n", "\n")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestDifferentTopicTimeoutInfocenterGetHandler_ServeHTTP(t *testing.T) {
	savedEventStreamTimeoutSeconds := EventStreamTimeoutSeconds
	EventStreamTimeoutSeconds = 2
	defer func() {
		EventStreamTimeoutSeconds = savedEventStreamTimeoutSeconds
	}()
	infocenterGetHandler, writer, _, request := mockGetRequestHandler(t, "different topic")
	publishTestEvent(&infocenterGetHandler)

	infocenterGetHandler.ServeHTTP(writer, request)

	if !reflect.DeepEqual(writer.e.writeHeader,
		http.Header{"Cache-Control": {"no-cache"}, "Content-Type": {"text/event-stream"}}) {
		t.Fatalf("Unexpected write headers %q", writer.e.writeHeader)
	}
	if writer.e.writeHeaderInvocations != 1 {
		t.Fatalf("Unexpected write header invocation count %d", writer.e.writeHeaderInvocations)
	}
	if writer.e.writeFlushInvocations != 2 {
		t.Fatalf("Unexpected write flush invocation count %d", writer.e.writeFlushInvocations)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations,
		bytesOfBytes("event: timeout\n", "data: 2s\n", "\n")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
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
