package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/osm/irc"
	"github.com/osm/pastebin"
)

// factoidRandom initializes the random source.
var factoidRandom = rand.New(rand.NewSource(time.Now().UnixNano()))

// initFactoidDefaults sets default values for all settings.
func (b *bot) initFactoidDefaults() {
	// Commands
	if b.IRC.FactoidCmd == "" {
		b.IRC.FactoidCmd = "!factoid"
	}
	if b.IRC.FactoidSubCmdAdd == "" {
		b.IRC.FactoidSubCmdAdd = "add"
	}
	if b.IRC.FactoidSubCmdAddDelimiter == "" {
		b.IRC.FactoidSubCmdAddDelimiter = " _is_ "
	}
	if b.IRC.FactoidSubCmdDelete == "" {
		b.IRC.FactoidSubCmdDelete = "forget"
	}
	if b.IRC.FactoidSubCmdSnoop == "" {
		b.IRC.FactoidSubCmdSnoop = "snoop"
	}
	if b.IRC.FactoidSubCmdCount == "" {
		b.IRC.FactoidSubCmdCount = "count"
	}

	// Messages
	if b.IRC.FactoidMsgAdd == "" {
		b.IRC.FactoidMsgAdd = "noted"
	}
	if b.IRC.FactoidMsgDelete == "" {
		b.IRC.FactoidMsgDelete = "*removed*"
	}
	if b.IRC.FactoidMsgSnoop == "" {
		b.IRC.FactoidMsgSnoop = "<id>: <author> taught me that <trigger> is <reply> <timestamp>"
	}
	if b.IRC.FactoidMsgCount == "" {
		b.IRC.FactoidMsgCount = "<trigger> has <count> occurrences"
	}
	if b.IRC.FactoidMsgIs == "" {
		b.IRC.FactoidMsgIs = "is"
	}

	// Grammar
	if b.IRC.FactoidGrammarAction == "" {
		b.IRC.FactoidGrammarAction = "<action>"
	}
	if b.IRC.FactoidGrammarRandomWho == "" {
		b.IRC.FactoidGrammarRandomWho = "<randomwho>"
	}
	if b.IRC.FactoidGrammarReply == "" {
		b.IRC.FactoidGrammarReply = "<reply>"
	}
	if b.IRC.FactoidGrammarWho == "" {
		b.IRC.FactoidGrammarWho = "<who>"
	}
}

// factoidHandler is the main entry point for all factoid related commands.
func (b *bot) factoidHandler(m *irc.Message) {
	if b.shouldIgnore(m) {
		return
	}

	// Parse the action
	a := b.parseAction(m).(*privmsgAction)
	if !a.validChannel {
		return
	}

	// Determine which action to take.
	subCmd := ""
	if len(a.args) > 0 {
		subCmd = a.args[0]
	}

	// First we'll prioritize commands, if the message isn't a command
	// we'll check if it's a factoid and if we should send a reply to the
	// channel.
	if a.cmd == b.IRC.FactoidCmd && subCmd == b.IRC.FactoidSubCmdAdd && len(a.args) >= 4 {
		// Remove the factoid cmd and sub cmd from the message.
		msg := strings.Replace(
			a.msg,
			fmt.Sprintf("%s %s ", b.IRC.FactoidCmd, b.IRC.FactoidSubCmdAdd),
			"",
			1,
		)

		// Make sure that the delimiter word is present
		dpos := strings.Index(msg, b.IRC.FactoidSubCmdAddDelimiter)
		if dpos == -1 {
			return
		}

		// Insert the factoid.
		b.factoidHandleInsertFact(
			a.nick,
			msg[0:dpos],
			msg[dpos+len(b.IRC.FactoidSubCmdAddDelimiter):],
		)
	} else if a.cmd == b.IRC.FactoidCmd && subCmd == b.IRC.FactoidSubCmdDelete && len(a.args) == 2 {
		b.factoidHandleDelete(a.args[1])
	} else if a.cmd == b.IRC.FactoidCmd && subCmd == b.IRC.FactoidSubCmdSnoop && len(a.args) >= 2 {
		b.factoidHandleSnoop(
			strings.Replace(
				a.msg,
				fmt.Sprintf("%s %s ", b.IRC.FactoidCmd, b.IRC.FactoidSubCmdSnoop),
				"",
				1,
			),
		)
	} else if a.cmd == b.IRC.FactoidCmd && subCmd == b.IRC.FactoidSubCmdCount && len(a.args) >= 2 {
		b.factoidHandleCount(
			strings.Replace(
				a.msg,
				fmt.Sprintf("%s %s ", b.IRC.FactoidCmd, b.IRC.FactoidSubCmdCount),
				"",
				1,
			),
		)
	} else {
		b.factoidHandleFact(a)
	}
}

// factoidHandleDelete deletes the given factoid if the id exists. If the id
// doesn't exist it will silently ignore the message.
func (b *bot) factoidHandleDelete(id string) {
	// We expect a valid UUID to be sent.
	if !isUUID(id) {
		return
	}

	// Prepare the UPDATE statement. We are not actually deleting the
	// factoid, we'll just hide it so that it can be restored if we have
	// someone deleting things we want to keep.
	stmt, err := b.prepare("UPDATE factoid SET is_deleted = 1 WHERE id = ?")
	if err != nil {
		b.logger.Printf("factoidHandleDelete: %v", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer stmt.Close()

	// Execute the UPDATE statement.
	_, err = stmt.Exec(id)
	if err != nil {
		b.logger.Printf("factoidHandleDelete: %v", err)
		b.privmsgf(b.DB.Err)
		return
	}

	// Send a notice that the factoid was removed.
	b.privmsgf(b.IRC.FactoidMsgDelete)
}

// factoidHandleSnoop finds information about the given factoid. If there are
// more than five factoids found for the given trigger it'll send the message
// as a private message instead so we don't flood the channel.
func (b *bot) factoidHandleSnoop(trigger string) {
	// Get all the relevant factoid information
	rows, err := b.query("SELECT id, author, timestamp, reply FROM factoid WHERE trigger = ? AND is_deleted = 0", trigger)
	if err != nil {
		b.logger.Printf("factoidHandleSnoop: %v", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer rows.Close()

	// We'll setup a struct so that we more easily can iterate over all
	// the factoids.
	type fact struct {
		id        string
		author    string
		timestamp string
		reply     string
	}

	// Fetch the facts.
	var facts []fact
	for rows.Next() {
		var i, a, t, r string
		rows.Scan(&i, &a, &t, &r)
		facts = append(facts, fact{i, a, t, r})
	}

	// Determine the target of the information.
	var target string
	if len(facts) > 5 {
		target = "pastebin"
	} else {
		target = b.IRC.Channel
	}

	// Send the information back to the given target.
	var pastebinCode string

	for _, f := range facts {
		data := map[string]string{
			"<id>":        f.id,
			"<author>":    f.author,
			"<trigger>":   trigger,
			"<reply>":     f.reply,
			"<timestamp>": f.timestamp,
		}

		if target == "pastebin" {
			code := b.IRC.FactoidMsgSnoop
			for k, v := range data {
				code = strings.ReplaceAll(code, k, v)
			}

			if pastebinCode == "" {
				pastebinCode = fmt.Sprintf("%s", code)
			} else {
				pastebinCode = fmt.Sprintf("%s\n%s", pastebinCode, code)
			}

		} else {
			b.privmsgph(b.IRC.FactoidMsgSnoop, data)
		}
	}

	if target == "pastebin" {
		if b.IRC.PastebinAPIKey == "" {
			b.logger.Printf("factoidHandleSnoop: you need to set a pastebin api key\n")
			return
		}
		pb := pastebin.New(b.IRC.PastebinAPIKey)

		var url string
		url, err = pb.NewPaste(pastebinCode, trigger, pastebin.Unlisted, pastebin.TenMinutes)
		if err != nil {
			b.logger.Printf("factoidHandleSnoop: pastebin err: %v\n", err)
			return
		}

		b.privmsgf(url)
	}
}

// factoidHandleCount returns the number of occurrences the given trigger has.
func (b *bot) factoidHandleCount(trigger string) {
	// Get the count for the given trigger.
	var count int
	err := b.queryRow("SELECT COUNT(*) FROM factoid WHERE trigger = ? AND is_deleted = 0", trigger).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		b.logger.Printf("factoidHandleCount: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}

	// Return it to the channel.
	b.privmsgph(b.IRC.FactoidMsgCount, map[string]string{
		"<trigger>": trigger,
		"<count>":   fmt.Sprintf("%d", count),
	})
}

// factoidHandleInsertFact inserts a new factoid into the database.
func (b *bot) factoidHandleInsertFact(author, trigger, reply string) {
	// Prepare the INSERT statement.
	stmt, err := b.prepare("INSERT INTO factoid (id, timestamp, author, trigger, reply, is_deleted) VALUES(?, ?, ?, ?, ?, 0)")
	if err != nil {
		b.logger.Printf("factoidHandleInsertFact: %v", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer stmt.Close()

	// Execute it.
	_, err = stmt.Exec(newUUID(), newTimestamp(), author, trigger, reply)
	if err != nil {
		b.logger.Printf("factoidHandleInsertFact: %v", err)
		b.privmsgf(b.DB.Err)
		return
	}

	// ... and send a notice that the fact has been stored.
	b.privmsgf(b.IRC.FactoidMsgAdd)
}

// factoidHandleFact checks whether the message in the action is a known
// factoid. If it is, we'll parse the factoid and send the results back to the
// channel.
func (b *bot) factoidHandleFact(a *privmsgAction) {
	// Let's check whether the message is a known trigger.
	rows, err := b.query("SELECT reply FROM factoid WHERE trigger = ? AND is_deleted = 0", a.msg)
	if err != nil {
		b.logger.Printf("factoidHandleFact: %v", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer rows.Close()

	// There can be more than one factoid for a trigger. So let's store
	// all of them.
	var factoids []string
	for rows.Next() {
		var f string
		rows.Scan(&f)
		factoids = append(factoids, f)
	}

	// No factoids, return early.
	if len(factoids) < 1 {
		return
	}

	// If factoid rate is set, we'll only reply with the found factoid if
	// the random number is greater than the defined value in the config.
	if b.IRC.FactoidRate > 0 {
		if b.IRC.FactoidRate >= factoidRandom.Intn(100) {
			return
		}
	}

	// Get a random factoid from the slice.
	factoid := factoids[rand.Intn(len(factoids))]

	// Replace all occurences of <who> with the senders nick.
	i := strings.Index(factoid, b.IRC.FactoidGrammarWho)
	for i != -1 {
		factoid = factoid[0:i] + a.nick + factoid[i+len(b.IRC.FactoidGrammarWho):]
		i = strings.Index(factoid, b.IRC.FactoidGrammarWho)
	}

	// Replace all occurences of <randomwho> with a random nick from the
	// names map.
	i = strings.Index(factoid, b.IRC.FactoidGrammarRandomWho)
	for i != -1 {
		factoid = factoid[0:i] + b.rndName() + factoid[i+len(b.IRC.FactoidGrammarRandomWho):]
		i = strings.Index(factoid, b.IRC.FactoidGrammarRandomWho)
	}

	// Handle replies.
	if strings.HasPrefix(factoid, b.IRC.FactoidGrammarReply) {
		b.privmsgf(factoid[len(b.IRC.FactoidGrammarReply)+1:])
	} else if strings.HasPrefix(factoid, b.IRC.FactoidGrammarAction) {
		b.actionf(factoid[len(b.IRC.FactoidGrammarAction)+1:])
	} else {
		b.privmsgf("%s %s %s", a.msg, b.IRC.FactoidMsgIs, factoid)
	}
}
