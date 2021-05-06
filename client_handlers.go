package ircx

import (
	"fmt"
	"strings"

	"github.com/go-irc/irc/v4"
)

type clientFilter func(*Client, *irc.Message)

// clientFilters are pre-processing which happens for certain message
// types. These were moved from below to keep the complexity of each
// component down.
var clientFilters = map[string]clientFilter{
	"001":  handle001,
	"433":  handle433,
	"437":  handle437,
	"PING": handlePing,
	"PONG": handlePong,
	"NICK": handleNick,
	"CAP":  handleCap,
}

// From rfc2812 section 5.1 (Command responses)
//
// 001    RPL_WELCOME
//        "Welcome to the Internet Relay Network
//        <nick>!<user>@<host>"
func handle001(c *Client, m *irc.Message) {
	c.currentNick = m.Params[0]
	c.connected = true

	// If the user provided a NickServ password, try to identify.
	if c.config.NickServPass != "" {
		c.Writef("PRIVMSG %s :IDENTIFY %s", c.config.NickServNick, c.config.NickServPass)
	}

}

// From rfc2812 section 5.2 (Error Replies)
//
// 433    ERR_NICKNAMEINUSE
//        "<nick> :Nickname is already in use"
//
// - Returned when a NICK message is processed that results
//   in an attempt to change to a currently existing
//   nickname.
func handle433(c *Client, m *irc.Message) {
	// We only want to try and handle nick collisions during the initial
	// handshake.
	if c.connected {
		return
	}
	c.currentNick += "_"
	c.Writef("NICK :%s", c.currentNick)
}

// From rfc2812 section 5.2 (Error Replies)
//
// 437    ERR_UNAVAILRESOURCE
//        "<nick/channel> :Nick/channel is temporarily unavailable"
//
// - Returned by a server to a user trying to join a channel
//   currently blocked by the channel delay mechanism.
//
// - Returned by a server to a user trying to change nickname
//   when the desired nickname is blocked by the nick delay
//   mechanism.
func handle437(c *Client, m *irc.Message) {
	// We only want to try and handle nick collisions during the initial
	// handshake.
	if c.connected {
		return
	}
	c.currentNick += "_"
	c.Writef("NICK :%s", c.currentNick)
}

func handlePing(c *Client, m *irc.Message) {
	reply := m.Copy()
	reply.Command = "PONG"
	c.WriteMessage(reply)
}

func handlePong(c *Client, m *irc.Message) {
	if c.incomingPongChan != nil {
		select {
		case c.incomingPongChan <- m.Trailing():
		default:
			// Note that this return isn't really needed, but it helps some code
			// coverage tools actually see this line.
			return
		}
	}
}

func handleNick(c *Client, m *irc.Message) {
	if m.Prefix.Name == c.currentNick && len(m.Params) > 0 {
		c.currentNick = m.Params[0]
	}
}

var capFilters = map[string]clientFilter{
	"LS":  handleCapLs,
	"ACK": handleCapAck,
	"NAK": handleCapNak,
}

func handleCap(c *Client, m *irc.Message) {
	if c.remainingCapResponses <= 0 || len(m.Params) <= 2 {
		return
	}

	if filter, ok := capFilters[m.Params[1]]; ok {
		filter(c, m)
	}

	if c.remainingCapResponses <= 0 {
		for key, cap := range c.caps {
			if cap.Required && !cap.Enabled {
				c.sendError(fmt.Errorf("CAP %s requested but not accepted", key))
				return
			}
		}

		c.Write("CAP END")
	}
}

func handleCapLs(c *Client, m *irc.Message) {
	for _, key := range strings.Split(m.Trailing(), " ") {
		cap := c.caps[key]
		cap.Available = true
		c.caps[key] = cap
	}
	c.remainingCapResponses--
}

func handleCapAck(c *Client, m *irc.Message) {
	for _, key := range strings.Split(m.Trailing(), " ") {
		cap := c.caps[key]
		cap.Enabled = true
		c.caps[key] = cap
	}
	c.remainingCapResponses--
}

func handleCapNak(c *Client, m *irc.Message) {
	// If we got a NAK and this REQ was required, we need to bail
	// with an error.
	for _, key := range strings.Split(m.Trailing(), " ") {
		if c.caps[key].Required {
			c.sendError(fmt.Errorf("CAP %s requested but was rejected", key))
			return
		}
	}
	c.remainingCapResponses--
}
