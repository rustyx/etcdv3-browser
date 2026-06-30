package main

import "log"

// Broker is a helper class to distribute updates to connected clients.
type Broker struct {
	stopCh    chan struct{}
	resetCh   chan struct{}
	publishCh chan any
	subCh     chan chan any
	unsubCh   chan chan any
}

// NewBroker is self-explanatory.
func NewBroker() *Broker {
	return &Broker{
		stopCh:    make(chan struct{}),
		resetCh:   make(chan struct{}),
		publishCh: make(chan any, 64),
		subCh:     make(chan chan any),
		unsubCh:   make(chan chan any),
	}
}

// Start is self-explanatory.
func (b *Broker) Start() {
	subs := map[chan any]struct{}{}
	for {
		select {
		case <-b.stopCh:
			for msgCh := range subs {
				close(msgCh)
			}
			return
		case <-b.resetCh:
			for msgCh := range subs {
				close(msgCh)
			}
			subs = map[chan any]struct{}{}
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

// Reset closes all subscriber channels (disconnecting clients) but keeps the
// broker running.
func (b *Broker) Reset() {
	b.resetCh <- struct{}{}
}

// Subscribe returns a new channel for a receiver.
func (b *Broker) Subscribe() chan any {
	msgCh := make(chan any, 64)
	b.subCh <- msgCh
	return msgCh
}

// Unsubscribe unbinds a receiver. The broker owns channel lifetime; do not
// close msgCh after this.
func (b *Broker) Unsubscribe(msgCh chan any) {
	b.unsubCh <- msgCh
}

// Publish publishes a new message to all subscribers.
func (b *Broker) Publish(msg any) {
	b.publishCh <- msg
}
