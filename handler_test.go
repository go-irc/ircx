package ircx

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-irc/irc/v4"
)

func TestHandlerFunc(t *testing.T) {
	t.Parallel()

	hit := false
	var f HandlerFunc = func(c *Client, m *irc.Message) {
		hit = true
	}

	f.Handle(nil, nil)
	assert.True(t, hit, "HandlerFunc doesn't work correctly as Handler")
}
