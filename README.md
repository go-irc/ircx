# ircx

[![GoDoc](https://img.shields.io/badge/doc-GoDoc-blue.svg)](https://godoc.org/github.com/go-irc/ircx)
[![Build Status](https://img.shields.io/github/workflow/status/go-irc/ircx/CI.svg)](https://github.com/go-irc/ircx/actions)
[![Coverage Status](https://img.shields.io/coveralls/go-irc/ircx.svg)](https://coveralls.io/github/go-irc/ircx?branch=master)

## Import Paths

All development happens on the `master` branch and when features are
considered stable enough, a new release will be tagged.

## Example

```go
package main

import (
	"log"
	"net"

	"github.com/go-irc/irc.v4"
  "github.com/go-irc/ircx.v0"
)

func main() {
	conn, err := net.Dial("tcp", "chat.freenode.net:6667")
	if err != nil {
		log.Fatalln(err)
	}

	config := ircx.ClientConfig{
		Nick: "i_have_a_nick",
		Pass: "password",
		User: "username",
		Name: "Full Name",
		Handler: ircx.HandlerFunc(func(c *ircx.Client, m *irc.Message) {
			if m.Command == "001" {
				// 001 is a welcome event, so we join channels there
				c.Write("JOIN #bot-test-chan")
			} else if m.Command == "PRIVMSG" && c.FromChannel(m) {
				// Create a handler on all messages.
				c.WriteMessage(&irc.Message{
					Command: "PRIVMSG",
					Params: []string{
						m.Params[0],
						m.Trailing(),
					},
				})
			}
		}),
	}

	// Create the client
	client := ircx.NewClient(conn, config)
	err = client.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
```

## Major Version Changes

### v1

Initial release (In progress)
