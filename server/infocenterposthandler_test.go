package server

import (
	"bytes"
	"context"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
	"testing"
)

type testPostResponseWriterExpectations struct {
	writeHeaderInvocations int
	writeInvocations       [][]byte
}

type testPostResponseWriter struct {
	t                  *testing.T
	expectedStatusCode int
	e                  *testPostResponseWriterExpectations
}

func (w testPostResponseWriter) Header() http.Header {
	w.t.Fatal("Unexpected Header invocation")
	return nil
}

func (w testPostResponseWriter) Write(bytes []byte) (int, error) {
	w.e.writeInvocations = append(w.e.writeInvocations, bytes)
	return 0, nil
}

func (w testPostResponseWriter) WriteHeader(statusCode int) {
	if statusCode != w.expectedStatusCode {
		w.t.Errorf("Unexpected status code %d", statusCode)
		panic(w)
	}
	w.e.writeHeaderInvocations++
}

func (w testPostResponseWriter) Flush() {
	w.t.Fatal("Unexpected Flush invocation")
}

type failingReader struct {
	readInvocations int
}

func (r failingReader) Read(p []byte) (n int, err error) {
	r.readInvocations++
	return 0, errors.New("expected read error")
}

func TestMultipleSuccessfulInfocenterPostHandler_ServeHTTP(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	const postRequestCount = 10000
	for i := 0; i < postRequestCount; i++ {
		TestSuccessfulInfocenterPostHandler_ServeHTTP(t)
	}
}

func TestSuccessfulInfocenterPostHandler_ServeHTTP(t *testing.T) {
	infocenterPostHandler := infocenterPostHandler{eventStreamBroker: newEventStreamBroker()}
	defer infocenterPostHandler.eventStreamBroker.Stop()
	writer := testPostResponseWriter{
		t:                  t,
		expectedStatusCode: http.StatusNoContent,
		e:                  &testPostResponseWriterExpectations{},
	}
	request, err := http.NewRequestWithContext(context.Background(), "POST",
		"http://localhost/infocenter/test-topic", bytes.NewBufferString("message text"))
	request = mux.SetURLVars(request, map[string]string{"topic": "test-topic"})
	if err != nil {
		t.Fatalf("Got error while creating new request: %q", err)
	}
	quitCh := subscribeTestEvent(t, &infocenterPostHandler, false)

	infocenterPostHandler.ServeHTTP(writer, request)

	publishInvocationCount := <-quitCh
	if publishInvocationCount != 1 {
		t.Fatalf("Unexpected publish invocation count %d", publishInvocationCount)
	}
	if writer.e.writeHeaderInvocations != 1 {
		t.Fatalf("Unexpected write header invocation count %d", writer.e.writeHeaderInvocations)
	}
	if len(writer.e.writeInvocations) != 0 {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestFailedBodyReadInfocenterPostHandler_ServeHTTP(t *testing.T) {
	infocenterPostHandler := infocenterPostHandler{eventStreamBroker: newEventStreamBroker()}
	writer := testPostResponseWriter{
		t:                  t,
		expectedStatusCode: http.StatusInternalServerError,
		e:                  &testPostResponseWriterExpectations{},
	}
	request, err := http.NewRequestWithContext(context.Background(), "POST",
		"http://localhost/infocenter/test-topic", failingReader{})
	request = mux.SetURLVars(request, map[string]string{"topic": "test-topic"})
	if err != nil {
		t.Fatalf("Got error while creating new request: %q", err)
	}
	quitCh := subscribeTestEvent(t, &infocenterPostHandler, true)

	infocenterPostHandler.ServeHTTP(writer, request)

	publishInvocationCount := <-quitCh
	if publishInvocationCount != 0 {
		t.Fatalf("Unexpected publish invocation count %d", publishInvocationCount)
	}
	if writer.e.writeHeaderInvocations != 1 {
		t.Fatalf("Unexpected write header invocation count %d", writer.e.writeHeaderInvocations)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations, bytesOfBytes("expected read error")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestFailedTopicReadInfocenterPostHandler_ServeHTTP(t *testing.T) {
	infocenterPostHandler := infocenterPostHandler{eventStreamBroker: newEventStreamBroker()}
	writer := testPostResponseWriter{
		t:                  t,
		expectedStatusCode: http.StatusInternalServerError,
		e:                  &testPostResponseWriterExpectations{},
	}
	request, err := http.NewRequestWithContext(context.Background(), "POST",
		"http://localhost/infocenter/test-topic", bytes.NewBufferString("message text"))
	request = mux.SetURLVars(request, map[string]string{})
	if err != nil {
		t.Fatalf("Got error while creating new request: %q", err)
	}
	quitCh := subscribeTestEvent(t, &infocenterPostHandler, true)

	infocenterPostHandler.ServeHTTP(writer, request)

	publishInvocationCount := <-quitCh
	if publishInvocationCount != 0 {
		t.Fatalf("Unexpected publish invocation count %d", publishInvocationCount)
	}
	if writer.e.writeHeaderInvocations != 1 {
		t.Fatalf("Unexpected write header invocation count %d", writer.e.writeHeaderInvocations)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations, bytesOfBytes("Topic could not be recognized")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestMultilineInfocenterPostHandler_ServeHTTP(t *testing.T) {
	infocenterPostHandler := infocenterPostHandler{eventStreamBroker: newEventStreamBroker()}
	writer := testPostResponseWriter{
		t:                  t,
		expectedStatusCode: http.StatusNoContent,
		e:                  &testPostResponseWriterExpectations{},
	}
	request, err := http.NewRequestWithContext(context.Background(), "POST",
		"http://localhost/infocenter/test-topic", bytes.NewBufferString("mess\rage \ntext\r\n"))
	request = mux.SetURLVars(request, map[string]string{"topic": "test-topic"})
	if err != nil {
		t.Fatalf("Got error while creating new request: %q", err)
	}
	quitCh := subscribeTestEvent(t, &infocenterPostHandler, false)

	infocenterPostHandler.ServeHTTP(writer, request)

	publishInvocationCount := <-quitCh
	if publishInvocationCount != 1 {
		t.Fatalf("Unexpected publish invocation count %d", publishInvocationCount)
	}
	if writer.e.writeHeaderInvocations != 1 {
		t.Fatalf("Unexpected write header invocation count %d", writer.e.writeHeaderInvocations)
	}
	if len(writer.e.writeInvocations) != 0 {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}
