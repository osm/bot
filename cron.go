package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/osm/irc"
	"github.com/osm/pastebin"
	_cron "github.com/robfig/cron/v3"
)

// cron holds the internal data structures needed to run the cron lib.
type cron struct {
	cron   *_cron.Cron
	jobs   map[string]_cron.EntryID
	mu     sync.Mutex
	parser _cron.Parser
}

// add adds a new cron job to the cron lib.
func (c *cron) add(id, expression string, cmd func()) error {
	// Acquire a lock and release it when we return.
	c.mu.Lock()
	defer c.mu.Unlock()

	// The id is already in the jobs map, reaturn early.
	if _, ok := c.jobs[id]; ok {
		return nil
	}

	// Add it with the corresponding ID received by the cron lob.
	entryID, err := c.cron.AddFunc(expression, cmd)
	if err != nil {
		return err
	}

	// Add it to the map and return.
	c.jobs[id] = entryID
	return nil
}

// delete deletes the given cron job.
func (c *cron) delete(id string) {
	// Acquire a lock and release it at the end of the method.
	c.mu.Lock()
	defer c.mu.Unlock()

	// If there's an id, stop the job and remove the entry from the map.
	if entryID, ok := c.jobs[id]; ok {
		c.cron.Remove(entryID)
		delete(c.jobs, id)
	}
}

// newCron returns a new cron structure.
func newCron() *cron {
	return &cron{
		cron:   _cron.New(),
		jobs:   make(map[string]_cron.EntryID),
		parser: _cron.NewParser(_cron.Minute | _cron.Hour | _cron.Dom | _cron.Month | _cron.Dow),
	}
}

// initCron initializes the cron jobs.
func (b *bot) initCron() {
	// Fetch all active cron jobs from the database.
	rows, err := b.query("SELECT id, expression, message FROM cron WHERE is_deleted = 0")
	if err != nil {
		b.logger.Printf("initCron: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer rows.Close()

	// Iterate over the results and add new cron jobs for each job.
	for rows.Next() {
		var id, expression, message string
		rows.Scan(&id, &expression, &message)

		// Add a new cron job for the given expression, the message
		// that is defined will be sent back to the channel when the
		// cron job is triggered.
		err = b.cron.add(id, expression, b.cronNewFunc(message))
		if err != nil {
			b.logger.Printf("initCron: %w", err)
		}
	}

	// Start the cron job runner.
	b.cron.cron.Start()
}

// initCronDefaults sets default values for all settings.
func (b *bot) initCronDefaults() {
	// Command and sub commands.
	if b.IRC.CronCmd == "" {
		b.IRC.CronCmd = "!cron"
	}
	if b.IRC.CronSubCmdAdd == "" {
		b.IRC.CronSubCmdAdd = "add"
	}
	if b.IRC.CronSubCmdDelete == "" {
		b.IRC.CronSubCmdDelete = "delete"
	}
	if b.IRC.CronSubCmdList == "" {
		b.IRC.CronSubCmdList = "list"
	}
	if b.IRC.CronSubCmdUpdate == "" {
		b.IRC.CronSubCmdUpdate = "update"
	}

	// Messages
	if b.IRC.CronErr == "" {
		b.IRC.CronErr = "check your syntax"
	}
	if b.IRC.CronMsgAdd == "" {
		b.IRC.CronMsgAdd = "cron job added"
	}
	if b.IRC.CronMsgDelete == "" {
		b.IRC.CronMsgDelete = "cron job deleted"
	}
	if b.IRC.CronMsgList == "" {
		b.IRC.CronMsgList = "<id>: <expression> <message>"
	}
	if b.IRC.CronMsgUpdate == "" {
		b.IRC.CronMsgUpdate = "cron job updated"
	}
}

// cronNewFunc returns a new cron func with the given message.
func (b *bot) cronNewFunc(message string) func() {
	return func() {
		b.privmsgf(message)
	}
}

// cronHandler handles the IRC integration of the cron job scheduler.
func (b *bot) cronHandler(m *irc.Message) {
	if b.shouldIgnore(m) {
		return
	}

	// Parse the action
	a := b.parseAction(m).(*privmsgAction)
	if !a.validChannel {
		return
	}

	// Not a valid cron cmd, return early.
	if a.cmd != b.IRC.CronCmd || len(a.args) < 1 {
		return
	}

	subCmd := a.args[0]
	if subCmd == b.IRC.CronSubCmdAdd && len(a.args) >= 7 {
		// The cron expression should be within position 1 to 6 in the
		// args slice, the rest of the args is the message to send
		// when the cron expression is evaluated and hit.
		b.cronAdd(strings.Join(a.args[1:6], " "), strings.Join(a.args[6:], " "))
	} else if subCmd == b.IRC.CronSubCmdDelete && len(a.args) == 2 {
		b.cronDelete(a.args[1])
	} else if subCmd == b.IRC.CronSubCmdList && len(a.args) == 1 {
		b.cronList()
	} else if subCmd == b.IRC.CronSubCmdUpdate && len(a.args) >= 8 {
		// The first argument should be the id of the cron job to
		// update. The cron expression should be within position 2 to
		// 7 in the args slice, the rest of the args is the message to
		// send when the cron expression is evaluated and hit.
		b.cronUpdate(a.args[1], strings.Join(a.args[2:7], " "), strings.Join(a.args[7:], " "))
	}

}

// cronAdd adds the given expression and message to the database.
func (b *bot) cronAdd(expression, message string) {
	// Make sure that the expression is valid.
	_, err := b.cron.parser.Parse(expression)
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.IRC.CronErr)
	}

	// Prepare the INSERT statement.
	stmt, err := b.prepare("INSERT INTO cron (id, expression, message, is_deleted, inserted_at) VALUES(?, ?, ?, 0, ?)")
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer stmt.Close()

	// Execute it.
	id := newUUID()
	_, err = stmt.Exec(id, expression, message, newTimestamp())
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}

	// Add the job
	err = b.cron.add(id, expression, b.cronNewFunc(message))
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.IRC.CronErr)
	}

	// ... and send a notice that the cron job has been stored.
	b.privmsgf(b.IRC.CronMsgAdd)
}

// cronDelete deletes the cron job.
func (b *bot) cronDelete(id string) {
	// We expect a valid UUID to be sent.
	if !isUUID(id) {
		return
	}

	stmt, err := b.prepare("UPDATE cron SET is_deleted = 1 WHERE id = ?")
	if err != nil {
		b.logger.Printf("cronDelete: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer stmt.Close()

	// Execute the UPDATE statement.
	_, err = stmt.Exec(id)
	if err != nil {
		b.logger.Printf("cronDelete: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}

	// Delete the job from the runner.
	b.cron.delete(id)

	// Send a notice that the cron job was removed.
	b.privmsgf(b.IRC.CronMsgDelete)
}

// cronList lists all the cron jobs.
func (b *bot) cronList() {
	rows, err := b.query("SELECT id, expression, message FROM cron WHERE is_deleted = 0")
	if err != nil {
		b.logger.Printf("cronList: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer rows.Close()

	type cronjob struct {
		id         string
		expression string
		message    string
	}

	var cronjobs []cronjob
	for rows.Next() {
		var i, e, m string
		rows.Scan(&i, &e, &m)
		cronjobs = append(cronjobs, cronjob{i, e, m})
	}

	// Determine the target of the information.
	var target string
	if len(cronjobs) > 5 {
		target = "pastebin"
	} else {
		target = b.IRC.Channel
	}

	// Send the information back to the given target.
	var pastebinCode string

	for _, c := range cronjobs {
		data := map[string]string{
			"<id>":         c.id,
			"<expression>": c.expression,
			"<message>":    c.message,
		}

		if target == "pastebin" {
			code := b.IRC.CronMsgList
			for k, v := range data {
				code = strings.ReplaceAll(code, k, v)
			}

			if pastebinCode == "" {
				pastebinCode = fmt.Sprintf("%s", code)
			} else {
				pastebinCode = fmt.Sprintf("%s\n%s", pastebinCode, code)
			}

		} else {
			b.privmsgph(b.IRC.CronMsgList, data)
		}
	}

	if target == "pastebin" {
		if b.IRC.PastebinAPIKey == "" {
			b.logger.Printf("cronList: you need to set a pastebin api key\n")
			return
		}
		pb := pastebin.New(b.IRC.PastebinAPIKey)

		var url string
		url, err = pb.NewPaste(pastebinCode, "cron job", pastebin.Unlisted, pastebin.TenMinutes)
		if err != nil {
			b.logger.Printf("cronList: pastebin err: %v\n", err)
			return
		}

		b.privmsgf(url)
	}
}

// cronUpdate updates the cron job.
func (b *bot) cronUpdate(id, expression, message string) {
	// We expect a valid UUID to be sent.
	if !isUUID(id) {
		return
	}

	// Make sure that the expression is valid.
	_, err := b.cron.parser.Parse(expression)
	if err != nil {
		b.logger.Printf("cronUpdate: %w", err)
		b.privmsgf(b.IRC.CronErr)
	}

	// Prepare the update query.
	stmt, err := b.prepare("UPDATE cron SET expression = ?, message = ?, updated_at = ? WHERE id = ? AND is_deleted = 0")
	if err != nil {
		b.logger.Printf("cronUpdate: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer stmt.Close()

	// Execute the UPDATE statement.
	_, err = stmt.Exec(expression, message, newTimestamp(), id)
	if err != nil {
		b.logger.Printf("cronUpdate: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}

	// Delete the old job and re-add it as a new.
	b.cron.delete(id)
	err = b.cron.add(id, expression, b.cronNewFunc(message))
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.IRC.CronErr)
	}

	// Send a notice that the cron job was updated.
	b.privmsgf(b.IRC.CronMsgUpdate)

}
