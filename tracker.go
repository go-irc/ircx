package ircx

// TODO: store all nicks by uuid and map them in outgoing seabird events rather
// than passing the nicks around directly

// TODO: track the currentNick in the tracker so it can be independent of the
// Client.

import (
	"errors"
	"strings"
	"sync"

	"github.com/go-irc/irc/v4"
)

type Tracker struct {
	sync.RWMutex

	channels map[string]*ChannelState
}

func NewTracker() *Tracker {
	return &Tracker{
		channels: make(map[string]*ChannelState),
	}
}

type ChannelState struct {
	Name  string
	Topic string
	Users map[string]struct{}
}

func (t *Tracker) ListChannels() []string {
	t.RLock()
	defer t.RUnlock()

	var ret []string
	for channel := range t.channels {
		ret = append(ret, channel)
	}

	return ret
}

func (t *Tracker) GetChannel(name string) *ChannelState {
	t.RLock()
	defer t.RUnlock()

	return t.channels[name]
}
func (t *Tracker) Handle(client *Client, msg *irc.Message) error {
	switch msg.Command {
	case "332":
		return t.handleRplTopic(client, msg)
	case "353":
		return t.handleRplNamReply(client, msg)
	case "JOIN":
		return t.handleJoin(client, msg)
	case "TOPIC":
		return t.handleTopic(client, msg)
	case "PART":
		return t.handlePart(client, msg)
	case "KICK":
		return t.handleKick(client, msg)
	case "QUIT":
		return t.handleQuit(client, msg)
	case "NICK":
		return t.handleNick(client, msg)
	}

	return nil
}

func (t *Tracker) handleTopic(client *Client, msg *irc.Message) error {
	if len(msg.Params) != 2 {
		return errors.New("malformed TOPIC message")
	}

	channel := msg.Params[0]
	topic := msg.Trailing()

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received TOPIC message for unknown channel")
	}

	t.channels[channel].Topic = topic

	return nil
}

func (t *Tracker) handleRplTopic(client *Client, msg *irc.Message) error {
	if len(msg.Params) != 3 {
		return errors.New("malformed RPL_TOPIC message")
	}

	// client set channel topic to topic

	// client := msg.Params[0]
	channel := msg.Params[1]
	topic := msg.Trailing()

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received RPL_TOPIC for unknown channel")
	}

	t.channels[channel].Topic = topic

	return nil
}

func (t *Tracker) handleJoin(client *Client, msg *irc.Message) error {
	if len(msg.Params) != 1 {
		return errors.New("malformed JOIN message")
	}

	// user joined channel
	user := msg.Prefix.Name
	channel := msg.Trailing()

	t.Lock()
	defer t.Unlock()

	_, ok := t.channels[channel]

	if !ok {
		if user != client.CurrentNick() {
			return errors.New("received JOIN message for unknown channel")
		}

		t.channels[channel] = &ChannelState{Name: channel, Users: make(map[string]struct{})}
	}

	state := t.channels[channel]
	state.Users[user] = struct{}{}

	return nil
}

func (t *Tracker) handlePart(client *Client, msg *irc.Message) error {
	if len(msg.Params) < 1 {
		return errors.New("malformed PART message")
	}

	// user joined channel

	user := msg.Prefix.Name
	channel := msg.Params[0]

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received PART message for unknown channel")
	}

	// If we left the channel, we can drop the whole thing, otherwise just drop
	// this user from the channel.
	if user == client.CurrentNick() {
		delete(t.channels, channel)
	} else {
		state := t.channels[channel]
		delete(state.Users, user)
	}

	return nil
}

func (t *Tracker) handleKick(client *Client, msg *irc.Message) error {
	if len(msg.Params) != 3 {
		return errors.New("malformed KICK message")
	}

	// user was kicked from channel by actor

	//actor := msg.Prefix.Name
	user := msg.Params[1]
	channel := msg.Params[0]

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received KICK message for unknown channel")
	}

	// If we left the channel, we can drop the whole thing, otherwise just drop
	// this user from the channel.
	if user == client.CurrentNick() {
		delete(t.channels, channel)
	} else {
		state := t.channels[channel]
		delete(state.Users, user)
	}

	return nil
}

func (t *Tracker) handleQuit(client *Client, msg *irc.Message) error {
	if len(msg.Params) != 1 {
		return errors.New("malformed QUIT message")
	}

	// user quit

	user := msg.Prefix.Name

	t.Lock()
	defer t.Unlock()

	for _, state := range t.channels {
		delete(state.Users, user)
	}

	return nil
}

func (t *Tracker) handleNick(client *Client, msg *irc.Message) error {
	if len(msg.Params) != 1 {
		return errors.New("malformed NICK message")
	}

	// oldUser renamed to newUser

	oldUser := msg.Prefix.Name
	newUser := msg.Params[0]

	t.Lock()
	defer t.Unlock()

	for _, state := range t.channels {
		if _, ok := state.Users[oldUser]; ok {
			delete(state.Users, oldUser)
			state.Users[newUser] = struct{}{}
		}
	}

	return nil
}

func (t *Tracker) handleRplNamReply(client *Client, msg *irc.Message) error {
	if len(msg.Params) != 4 {
		return errors.New("malformed RPL_NAMREPLY message")
	}

	channel := msg.Params[2]
	users := strings.Split(strings.TrimSpace(msg.Trailing()), " ")

	prefixes, ok := client.ISupport.GetPrefixMap()
	if !ok {
		return errors.New("ISupport missing prefix map")
	}

	t.Lock()
	defer t.Unlock()

	if _, ok := t.channels[channel]; !ok {
		return errors.New("received RPL_NAMREPLY message for untracked channel")
	}

	for _, user := range users {
		i := strings.IndexFunc(user, func(r rune) bool {
			_, ok := prefixes[r]
			return !ok
		})

		if i != -1 {
			user = user[i:]
		}

		// The bot user should be added via JOIN
		if user == client.CurrentNick() {
			continue
		}

		state := t.channels[channel]
		state.Users[user] = struct{}{}
	}

	return nil
}
