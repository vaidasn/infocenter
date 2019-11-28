package server

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/vaidasn/infocenter/chanbroker"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

func ListenAndServe(port uint16) {
	server := NewServer()
	server.Addr = fmt.Sprintf(":%d", port)
	log.Fatal(server.ListenAndServe())
}

func NewServer() *http.Server {
	eventStreamBroker := newEventStreamBroker()
	r := configRoutes(eventStreamBroker)
	server := &http.Server{Handler: r}
	server.RegisterOnShutdown(func() {
		eventStreamBroker.Stop()
	})
	return server
}

func newEventStreamBroker() *chanbroker.Broker {
	eventStreamBroker := chanbroker.NewBroker()
	go eventStreamBroker.Start()
	return eventStreamBroker
}

func configRoutes(eventStreamBroker *chanbroker.Broker) *mux.Router {
	r := mux.NewRouter()
	r.Handle("/infocenter/{topic}", newInfocenterPostHandler(eventStreamBroker)).Methods(http.MethodPost)
	r.Handle("/infocenter/{topic}", newInfocenterGetHandler(eventStreamBroker)).Methods(http.MethodGet)
	return r
}

var EventStreamTimeoutSeconds = 30

type topicAndMessage struct {
	topic   string
	message string
}

type infocenterPostHandler struct {
	eventStreamBroker *chanbroker.Broker
}

func newInfocenterPostHandler(eventStreamBroker *chanbroker.Broker) *infocenterPostHandler {
	return &infocenterPostHandler{eventStreamBroker: eventStreamBroker}
}

func (handler infocenterPostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	bodyBuffer := bytes.Buffer{}
	if _, err := bodyBuffer.ReadFrom(request.Body); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		if _, err = writer.Write([]byte(err.Error())); err != nil {
			log.Println("Writing response failed: ", err)
		}
		return
	}
	message := legalMessage(bodyBuffer.String())
	topic, ok := requestTopic(request, writer)
	if !ok {
		return
	}
	handler.eventStreamBroker.Publish(topicAndMessage{topic, message})
	writer.WriteHeader(http.StatusNoContent)
}

func legalMessage(body string) (message string) {
	message = strings.ReplaceAll(body, "\r", "")
	message = strings.ReplaceAll(message, "\n", "")
	return
}

type infocenterGetHandler struct {
	eventStreamBroker          *chanbroker.Broker
	idCounter                  uint64
	aboutToEnterSelectLoopFunc func()
}

func newInfocenterGetHandler(eventStreamBroker *chanbroker.Broker) *infocenterGetHandler {
	return &infocenterGetHandler{eventStreamBroker: eventStreamBroker}
}

func (handler *infocenterGetHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	topic, ok := requestTopic(request, writer)
	if !ok {
		return
	}
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.WriteHeader(http.StatusOK)
	if writerFlusher, ok := writer.(http.Flusher); ok {
		writerFlusher.Flush()
	}
	messageChannel := handler.eventStreamBroker.Subscribe()
	context := request.Context()
	if handler.aboutToEnterSelectLoopFunc != nil {
		handler.aboutToEnterSelectLoopFunc()
	}
loop:
	for {
		select {
		case m := <-messageChannel:
			topicAndMessage := m.(topicAndMessage)
			if topicAndMessage.topic != topic {
				break
			}
			if err := writeEvent(&handler.idCounter, writer, "msg", topicAndMessage.message); err != nil {
				log.Println("Writing response failed: ", err)
				break loop
			}
		case <-time.After(time.Duration(EventStreamTimeoutSeconds) * time.Second):
			handler.eventStreamBroker.Unsubscribe(messageChannel)
			timeoutMessage := fmt.Sprintf("%ds", EventStreamTimeoutSeconds)
			if err := writeEvent(&handler.idCounter, writer, "timeout", timeoutMessage); err != nil {
				log.Println("Writing response failed: ", err)
			}
			break loop
		case <-context.Done():
			handler.eventStreamBroker.Unsubscribe(messageChannel)
			break loop
		}
	}
}

func requestTopic(request *http.Request, writer http.ResponseWriter) (topic string, ok bool) {
	requestParameters := mux.Vars(request)
	topic, ok = requestParameters["topic"]
	if !ok {
		writer.WriteHeader(http.StatusInternalServerError)
		if _, err := writer.Write([]byte("Topic could not be recognized")); err != nil {
			log.Println("Writing response failed: ", err)
		}
	}
	return topic, ok
}

func writeEvent(idCounter *uint64, w io.Writer, event string, data string) error {
	if writerFlusher, ok := w.(http.Flusher); ok {
		defer writerFlusher.Flush()
	}
	if !validEventAnyChar(data) {
		return errors.New("invalid event data")
	}
	if event != "" {
		if !validEventAnyChar(event) {
			return errors.New("invalid event name")
		}
	}
	if _, err := w.Write([]byte(fmt.Sprintln("id:", atomic.AddUint64(idCounter, 1)))); err != nil {
		return err
	}
	if event != "" {
		if _, err := w.Write([]byte(fmt.Sprintln("event:", event))); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte(fmt.Sprintln("data:", data))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func validEventAnyChar(value string) bool {
	if strings.ContainsAny(value, "\r\n") {
		return false
	}
	return true
}
