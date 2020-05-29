package ircx

import "github.com/go-irc/irc/v4"

// Handler is a simple interface meant for dispatching a message from
// a Client connection.
type Handler interface {
	Handle(*Client, *irc.Message)
}

// HandlerFunc is a simple wrapper around a function which allows it
// to be used as a Handler.
type HandlerFunc func(*Client, *irc.Message)

// Handle calls f(c, m).
func (f HandlerFunc) Handle(c *Client, m *irc.Message) {
	f(c, m)
}
