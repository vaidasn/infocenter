package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestPost(t *testing.T) {
	l, server, doneServing := listenAndServe(t)
	postUrl := fmt.Sprintf("http://%s/infocenter/test", l.Addr().String())
	response, err := http.DefaultClient.Post(postUrl, "text/plain", bytes.NewBufferString("test message"))
	if err != nil {
		t.Fatal("POST failed")
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("Response code was %d but expected %d", response.StatusCode, http.StatusNoContent)
	}
	stopServing(t, server, doneServing)
}

func TestGetTimeout(t *testing.T) {
	savedEventStreamTimeoutSeconds := EventStreamTimeoutSeconds
	EventStreamTimeoutSeconds = 2
	defer func() {
		EventStreamTimeoutSeconds = savedEventStreamTimeoutSeconds
	}()
	l, server, doneServing := listenAndServe(t)
	getUrl := fmt.Sprintf("http://%s/infocenter/test", l.Addr().String())
	response, err := http.DefaultClient.Get(getUrl)
	if err != nil {
		t.Fatal("GET failed")
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Response code was %d but expected %d", response.StatusCode, http.StatusOK)
	}
	bodyBuffer := bytes.Buffer{}
	if _, err := bodyBuffer.ReadFrom(response.Body); err != nil {
		t.Fatalf("Read body failed: %q", err)
	}
	responseContent := bodyBuffer.String()
	if responseContent != "event: timeout\ndata: 2s\n\n" {
		t.Fatalf("Unrecognized response content %q", responseContent)
	}

	stopServing(t, server, doneServing)
}

type testWriteEventWriterExpectations struct {
	writeHeaderInvocations int
	writeInvocations       [][]byte
	writeInvocationNum     int
}

type testWriteEventWriter struct {
	t           *testing.T
	failWriteOn int
	e           *testWriteEventWriterExpectations
}

func (w testWriteEventWriter) Write(bytes []byte) (int, error) {
	w.e.writeInvocations = append(w.e.writeInvocations, bytes)
	w.e.writeInvocationNum++
	if w.e.writeInvocationNum == w.failWriteOn {
		return 0, errors.New("testing failed writer error")
	} else {
		return 0, nil
	}
}

func TestWriteEventWithEventName(t *testing.T) {
	writer := testWriteEventWriter{t: t, e: &testWriteEventWriterExpectations{}}
	if err := writeEvent(writer, "message", "data1"); err != nil {
		t.Fatalf("Failed to write %q", err)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations,
		bytesOfBytes("event: message\n", "data: data1\n", "\n")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestWriteEventWithoutEventName(t *testing.T) {
	writer := testWriteEventWriter{t: t, e: &testWriteEventWriterExpectations{}}
	if err := writeEvent(writer, "", "data2"); err != nil {
		t.Fatalf("Failed to write %q", err)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations, bytesOfBytes("data: data2\n", "\n")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestWriteEventFailedWrite(t *testing.T) {
	writeEventFailedWrite(t, 1, bytesOfBytes("event: message\n"))
	writeEventFailedWrite(t, 2, bytesOfBytes("event: message\n", "data: data3\n"))
	writeEventFailedWrite(t, 3, bytesOfBytes("event: message\n", "data: data3\n", "\n"))
}

func writeEventFailedWrite(t *testing.T, failedWriteOn int, expectedWriteInvocation [][]byte) {
	writer := testWriteEventWriter{t: t, failWriteOn: failedWriteOn, e: &testWriteEventWriterExpectations{}}
	if err := writeEvent(writer, "message", "data3"); err == nil ||
		err.Error() != "testing failed writer error" {
		t.Fatalf("Unexpected write event error \"%v\"", err)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations, expectedWriteInvocation) {
		t.Errorf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestWriteEventWithMultilineEvent(t *testing.T) {
	writer := testWriteEventWriter{t: t, e: &testWriteEventWriterExpectations{}}
	if err := writeEvent(writer, "message\n", "data4"); err == nil ||
		err.Error() != "invalid event name" {
		t.Fatalf("Unexpected write event error \"%v\"", err)
	}
	if len(writer.e.writeInvocations) != 0 {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}

func TestWriteEventWithMultilineData(t *testing.T) {
	writer := testWriteEventWriter{t: t, e: &testWriteEventWriterExpectations{}}
	if err := writeEvent(writer, "message", "data5\rdata6"); err == nil ||
		err.Error() != "invalid event data" {
		t.Fatalf("Unexpected write event error \"%v\"", err)
	}
	if !reflect.DeepEqual(writer.e.writeInvocations,
		bytesOfBytes("event: message\n", "data: <invalid data>\n", "\n")) {
		t.Fatalf("Unexpected write invocations %q", writer.e.writeInvocations)
	}
}
