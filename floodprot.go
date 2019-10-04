package main

import (
	"strings"
	"sync"
	"time"

	"github.com/osm/irc"
)

// initFloodProtDefaults initializes the flood protection defaults.
func (b *bot) initFloodProtDefaults() {
	if b.IRC.FloodProtTimeThreshold == 0 {
		b.IRC.FloodProtTimeThreshold = 5
	}
	if b.IRC.FloodProtCmdThreshold == 0 {
		b.IRC.FloodProtCmdThreshold = 3
	}
	if b.IRC.FloodProtIgnoreTime == 0 {
		b.IRC.FloodProtIgnoreTime = 60
	}
	if b.IRC.FloodProtMsgIgnore == "" {
		b.IRC.FloodProtMsgIgnore = "<nick> is ignored"
	}
	if b.IRC.FloodProtMsgUnignore == "" {
		b.IRC.FloodProtMsgUnignore = "<nick> is unignored"
	}
}

// floodProtInfo holds information about each user that has written anything
// in the configured channel.
type floodProtInfo struct {
	// isIgnored is a convenient flag that is set as soon as the user is
	// ignored.
	isIgnored bool

	// nick contains the current nick of the user.
	nick string

	// commandCount contains the number of commands the user has issued
	// within the last x seconds.
	commandCount int

	// lastTimestamp is the UNIX timestamp of the last issued command.
	lastTimestamp int
}

// Keep track of the command rate for all users within the channel.
var (
	floodProt   map[string]*floodProtInfo
	floodProtMu sync.Mutex
)

// initFloodProt initializes the flood protection.
func (b *bot) initFloodProt() {
	floodProt = make(map[string]*floodProtInfo)

	// This goroutine is responsible for the unignore phase of the flood
	// protection system.
	// It is executed once every minute and iterates over the map of flood
	// protection info, for every user that has been ignored a check is
	// made to determine if the user should be unignored or not.
	go func(b *bot) {
		for {
			// Get the current timestamp.
			timestamp := newUnixTimestamp()

			// Acquire a lock for the flood prot map.
			floodProtMu.Lock()

			// Iterate over the flood protection info and check
			// if enough time has passed to remove the ignore for
			// the user.
			for k, v := range floodProt {
				if v.isIgnored && timestamp-v.lastTimestamp > b.IRC.FloodProtIgnoreTime {
					v.isIgnored = false

					// Time to remove the ignore from the
					// ignore map, so we need to acquire a
					// lock to prevent a race condition.
					b.IRC.ignoreDynMu.Lock()
					delete(b.IRC.ignoreDyn, k)
					b.IRC.ignoreDynMu.Unlock()

					b.privmsgph(b.IRC.FloodProtMsgUnignore, map[string]string{
						"<nick>": v.nick,
					})
				}
			}

			// Release the lock.
			floodProtMu.Unlock()

			// We'll sleep the goroutine for 60 seconds before we
			// run the check again to see if the users that has
			// been ignored due to flood protection should be
			// unignored.
			time.Sleep(time.Duration(60 * time.Second))
		}
	}(b)
}

func (b *bot) floodProtHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	// We are not intrested in preventing flood for anything but commands,
	// so unless the message starts with a ! we'll just ignore the
	// message.
	if !strings.HasPrefix(a.msg, "!") {
		return
	}

	// Acquire a lock for the flood prot map and release it as soon as we
	// return.
	floodProtMu.Lock()
	defer floodProtMu.Unlock()

	// Check whether or not the host has an entry in the flood prot map.
	// If it doesn't we'll insert a new entry and return.
	info, ok := floodProt[a.host]
	if !ok {
		floodProt[a.host] = &floodProtInfo{
			nick:          a.nick,
			lastTimestamp: newUnixTimestamp(),
		}
		return
	}

	// The user has written something before, it's time to update the
	// info for the given host.
	timestamp := newUnixTimestamp()

	// The current timestamp subtracted with the last timestamp was less
	// than the configured time threshold, so we need to increment the
	// command count by one.
	if timestamp-info.lastTimestamp < b.IRC.FloodProtTimeThreshold {
		// Increment the command count if it was executed prior to the cmd
		// threshold.
		info.commandCount += 1

		// If the user has executed more commands than the defined
		// value in cmd threshold we'll mark the user as ignored so
		// that the shouldIgnore method can ignore the command by the
		// user.
		if !info.isIgnored && info.commandCount >= b.IRC.FloodProtCmdThreshold {
			info.isIgnored = true

			b.IRC.ignoreDynMu.Lock()
			b.IRC.ignoreDyn[a.host] = true
			b.IRC.ignoreDynMu.Unlock()

			b.privmsgph(b.IRC.FloodProtMsgIgnore, map[string]string{
				"<nick>": a.nick,
			})
		}
	} else {
		// The command was not within the time threshhold, so we'll
		// reset the command count.
		info.commandCount = 0
	}

	// And finally, update the nick and set the last timestamp to the
	// current timestamp.
	info.nick = a.nick
	info.lastTimestamp = timestamp
}
