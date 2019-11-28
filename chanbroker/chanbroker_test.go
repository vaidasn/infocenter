package chanbroker

import (
	"testing"
)

func TestBroker_Publish(t *testing.T) {
	b := NewBroker()
	go b.Start()

	const clientCount = 3
	clientDone := make(chan [clientCount]int)
	// Create and subscribe some clients:
	clientFunc := func(id int, subscriberCh chan chan interface{}) {
		msgCh := b.Subscribe()
		subscriberCh <- msgCh
		msgSum := 0
		msgCount := 0
		for msg := range msgCh {
			msgSum += msg.(int)
			msgCount++
		}
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
		for i := 0; i < clientCount; i++ {
			b.Unsubscribe(subscribers[i])
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
	b.Stop()
}
