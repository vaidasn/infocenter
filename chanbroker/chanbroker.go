// Modified version of broker given at https://stackoverflow.com/a/49877632
//
// The Broker uses single chan for subscribe, unsubscribe and publish events.
// That ensures proper serialization during event processing what
// simplifies test predictability.
package chanbroker

type eventType int

const (
	eventSubscribe eventType = iota
	eventUnsubscribe
	eventPublish
)

type event struct {
	eventType eventType
	content   interface{}
}

type Broker struct {
	stopCh  chan struct{}
	eventCh chan event
}

func NewBroker() *Broker {
	return &Broker{
		stopCh:  make(chan struct{}),
		eventCh: make(chan event, 1),
	}
}

func (b *Broker) Start() {
	subs := map[chan interface{}]struct{}{}
	for {
		select {
		case <-b.stopCh:
			return
		case event := <-b.eventCh:
			switch event.eventType {
			case eventSubscribe:
				msgCh := event.content.(chan interface{})
				subs[msgCh] = struct{}{}
			case eventUnsubscribe:
				msgCh := event.content.(chan interface{})
				if _, ok := subs[msgCh]; ok {
					delete(subs, msgCh)
					close(msgCh)
				}
			case eventPublish:
				msg := event.content
				for msgCh := range subs {
					msgCh <- msg
				}
			}
		}
	}
}

func (b *Broker) Stop() {
	close(b.stopCh)
}

func (b *Broker) Subscribe() chan interface{} {
	msgCh := make(chan interface{}, 1)
	b.eventCh <- event{
		eventType: eventSubscribe,
		content:   msgCh,
	}
	return msgCh
}

func (b *Broker) Unsubscribe(msgCh chan interface{}) {
	go func() {
		b.eventCh <- event{
			eventType: eventUnsubscribe,
			content:   msgCh,
		}
	}()
	for range msgCh {
	}
}

func (b *Broker) Publish(msg interface{}) {
	b.eventCh <- event{
		eventType: eventPublish,
		content:   msg,
	}
}
