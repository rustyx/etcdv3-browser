package main

import "log"

// Broker is a helper class to distribute updates to connected clients.
type Broker struct {
	stopCh    chan struct{}
	publishCh chan interface{}
	subCh     chan chan interface{}
	unsubCh   chan chan interface{}
}

// NewBroker is self-explanatory.
func NewBroker() *Broker {
	return &Broker{
		stopCh:    make(chan struct{}),
		publishCh: make(chan interface{}, 64),
		subCh:     make(chan chan interface{}),
		unsubCh:   make(chan chan interface{}),
	}
}

// Start is self-explanatory.
func (b *Broker) Start() {
	subs := map[chan interface{}]struct{}{}
	for {
		select {
		case <-b.stopCh:
			for msgCh := range subs {
				close(msgCh)
			}
			return
		case msgCh := <-b.subCh:
			subs[msgCh] = struct{}{}
		case msgCh := <-b.unsubCh:
			delete(subs, msgCh)
		case msg := <-b.publishCh:
			for msgCh := range subs {
				// msgCh is buffered, use non-blocking send to protect the broker:
				select {
				case msgCh <- msg:
				default:
					log.Print("Client is stuck - message ignored")
				}
			}
		}
	}
}

// Stop stops the broker.
func (b *Broker) Stop() {
	close(b.stopCh)
}

// Subscribe returns a new channel for a receiver.
func (b *Broker) Subscribe() chan interface{} {
	msgCh := make(chan interface{}, 64)
	b.subCh <- msgCh
	return msgCh
}

// Unsubscribe unbinds a receiver.
func (b *Broker) Unsubscribe(msgCh chan interface{}) {
	b.unsubCh <- msgCh
	close(msgCh)
}

// Publish publishes a new message to all subscribers.
func (b *Broker) Publish(msg interface{}) {
	b.publishCh <- msg
}
