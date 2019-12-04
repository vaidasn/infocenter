package chanbroker

import (
	"testing"
)

func TestBroker_Publish(t *testing.T) {
	b := NewBroker()
	go b.Start()
	defer b.Stop()

	const clientCount = 3
	const stopClientMessage = -1
	clientDone := make(chan [clientCount]int)
	// Create and subscribe some clients:
	clientFunc := func(id int, subscriberCh chan chan interface{}) {
		msgCh := b.Subscribe()
		subscriberCh <- msgCh
		msgSum := 0
		msgCount := 0
		for msg := range msgCh {
			msgInt := msg.(int)
			if msgInt == stopClientMessage {
				break
			}
			msgSum += msgInt
			msgCount++
		}
		b.Unsubscribe(msgCh)
		clientDone <- [clientCount]int{id, msgSum, msgCount}
	}
	var subscribers [clientCount]chan interface{}
	for i := 0; i < clientCount; i++ {
		subscriberCh := make(chan chan interface{})
		go clientFunc(i, subscriberCh)
		subscribers[i] = <-subscriberCh
	}

	// Start publishing messages:
	const messageCount = 10
	const messageSum = 45
	go func() {
		for msgId := 0; msgId < messageCount; msgId++ {
			b.Publish(msgId)
		}
		for msgId := 0; msgId < messageCount; msgId++ {
			b.Publish(stopClientMessage)
		}
	}()
	accumulatedIds := 0
	for i := 0; i < clientCount; i++ {
		idMsgSumAndCount := <-clientDone
		accumulatedIds += idMsgSumAndCount[0]
		if idMsgSumAndCount[1] != messageSum {
			t.Errorf("Wrong sum from cient id %d: %d", idMsgSumAndCount[0], idMsgSumAndCount[1])
		}
		if idMsgSumAndCount[2] != messageCount {
			t.Errorf("Wrong count from cient id %d: %d", idMsgSumAndCount[0], idMsgSumAndCount[2])
		}
	}
}

func TestConcurrentPublish_Unsubscribe(t *testing.T) {
	b := NewBroker()
	go b.Start()
	defer b.Stop()
	msgCh := b.Subscribe()
	b.Publish("received message")
	<-msgCh
	b.Publish("buffered message")
	b.Publish("pending message")
	b.Unsubscribe(msgCh)
	b.Unsubscribe(msgCh) // Allow unsubscribe already unsubscribed channel
	b.Publish("dummy message")
}
