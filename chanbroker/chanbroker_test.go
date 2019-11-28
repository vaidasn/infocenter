package chanbroker

import (
	"testing"
)

func TestBroker_Publish(t *testing.T) {
	b := NewBroker()
	go b.Start()

	clientDone := make(chan [3]int)
	// Create and subscribe 3 clients:
	clientFunc := func(id int, subscriberCh chan chan interface{}) {
		msgCh := b.Subscribe()
		subscriberCh <- msgCh
		msgSum := 0
		msgCount := 0
		for msg := range msgCh {
			msgSum += msg.(int)
			msgCount++
		}
		clientDone <- [3]int{id, msgSum, msgCount}
	}
	var subscribers [3]chan interface{}
	for i := 0; i < 3; i++ {
		subscriberCh := make(chan chan interface{})
		go clientFunc(i, subscriberCh)
		subscribers[i] = <-subscriberCh
	}

	// Start publishing messages:
	go func() {
		for msgId := 0; msgId < 10; msgId++ {
			b.Publish(msgId)
		}
		for i := 0; i < 3; i++ {
			b.Unsubscribe(subscribers[i])
		}
	}()
	accumulatedIds := 0
	for i := 0; i < 3; i++ {
		idMsgSumAndCount := <-clientDone
		accumulatedIds += idMsgSumAndCount[0]
		if idMsgSumAndCount[1] != 45 {
			t.Errorf("Wrong sum from cient id %d: %d", idMsgSumAndCount[0], idMsgSumAndCount[1])
		}
		if idMsgSumAndCount[2] != 10 {
			t.Errorf("Wrong count from cient id %d: %d", idMsgSumAndCount[0], idMsgSumAndCount[2])
		}
	}
	b.Stop()
}
