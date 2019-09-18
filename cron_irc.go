package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/osm/irc"
	"github.com/osm/pastebin"
)

// cronAddLimitRegexp contains the regular expression used to add cron jobs
// with an execution limit.
var cronAddLimitRegexp *regexp.Regexp

// initCronDefaults sets default values for all settings.
func (b *bot) initCronDefaults() {
	// Command and sub commands.
	if b.IRC.CronCmd == "" {
		b.IRC.CronCmd = "!cron"
	}
	if b.IRC.CronSubCmdAdd == "" {
		b.IRC.CronSubCmdAdd = "add"
	}
	if b.IRC.CronSubCmdAddLimit == "" {
		b.IRC.CronSubCmdAddLimit = "limit"
	}
	cronAddLimitRegexp = regexp.MustCompile("^" + b.IRC.CronSubCmdAddLimit + ":([0-9]+) ")
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
		b.IRC.CronMsgList = "id: <id> expression: <expression> message: <message> limited: <is_limited> count: <exec_count>/<exec_limit>"
	}
	if b.IRC.CronMsgUpdate == "" {
		b.IRC.CronMsgUpdate = "cron job updated"
	}

	// Grammar
	if b.IRC.CronGrammarMsgExecCount == "" {
		b.IRC.CronGrammarMsgExecCount = "<exec_count>"
	}
	if b.IRC.CronGrammarMsgExecLimit == "" {
		b.IRC.CronGrammarMsgExecLimit = "<exec_limit>"
	}
	if b.IRC.CronGrammarMsgIsLimited == "" {
		b.IRC.CronGrammarMsgIsLimited = "<is_limited>"
	}
	if b.IRC.CronGrammarMsgRandomWho == "" {
		b.IRC.CronGrammarMsgRandomWho = "<random_who>"
	}
}

// initCron initializes the cron jobs.
func (b *bot) initCron() {
	// Fetch all active cron jobs from the database.
	rows, err := b.query("SELECT id, expression, message, is_limited, exec_count, exec_limit FROM cron WHERE is_deleted = 0")
	if err != nil {
		b.logger.Printf("initCron: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer rows.Close()

	// Iterate over the results and add new cron jobs for each job.
	for rows.Next() {
		var id, expression, message string
		var isLimited bool
		var execCount, execLimit int
		rows.Scan(&id, &expression, &message, &isLimited, &execCount, &execLimit)

		// Add a new cron job for the given expression, the message
		// that is defined will be sent back to the channel when the
		// cron job is triggered. We don't add limited jobs where the
		// execution count has reached its limit.
		if isLimited == false || (isLimited == true && execCount < execLimit) {
			err = b.cron.add(id, expression, message, execCount, execLimit, isLimited, b)
			if err != nil {
				b.logger.Printf("initCron: %w", err)
			}
		}
	}

	// Start the cron job runner.
	b.cron.cron.Start()
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
	schedule, err := b.cron.parser.Parse(expression)
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.IRC.CronErr)
	}

	// Determine whether the message also holds an execution limit or not,
	// if it does we'll extract the limit and use it when we insert the
	// new job.
	var execLimit int
	var isLimited bool
	if matches := cronAddLimitRegexp.FindStringSubmatch(message); len(matches) > 1 {
		// Since the value is controlled by the regexp we can safely
		// assume that  the value is going to be converted correctly.
		el, _ := strconv.Atoi(matches[1])
		if el <= 0 {
			execLimit = 1
		} else {
			execLimit = el
		}
		isLimited = true
		message = strings.Replace(message, matches[0], "", 1)
	}

	// Prepare the INSERT statement.
	stmt, err := b.prepare("INSERT INTO cron (id, expression, message, is_limited, exec_limit, is_deleted, inserted_at) VALUES(?, ?, ?, ?, ?, 0, ?)")
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}
	defer stmt.Close()

	// Execute it.
	id := newUUID()
	_, err = stmt.Exec(id, expression, message, isLimited, execLimit, newTimestamp())
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.DB.Err)
		return
	}

	// Add the job
	err = b.cron.add(id, expression, message, 0, execLimit, isLimited, b)
	if err != nil {
		b.logger.Printf("cronAdd: %w", err)
		b.privmsgf(b.IRC.CronErr)
	}

	// ... and send a notice that the cron job has been stored.
	b.privmsgph(b.IRC.CronMsgAdd, map[string]string{
		"<next_execution>": schedule.Next(time.Now()).Format("2006-01-02 15:04"),
	})
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
	rows, err := b.query("SELECT id, expression, message, is_limited, exec_count, exec_limit FROM cron WHERE is_deleted = 0")
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
		isLimited  bool
		execCount  int
		execLimit  int
	}

	var cronjobs []cronjob
	for rows.Next() {
		var id, expression, message string
		var isLimited bool
		var execCount, execLimit int
		rows.Scan(&id, &expression, &message, &isLimited, &execCount, &execLimit)

		if isLimited == false || (isLimited == true && execCount < execLimit) {
			cronjobs = append(cronjobs, cronjob{
				id,
				expression,
				message,
				isLimited,
				execCount,
				execLimit,
			})
		}
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
		var nextExecution string
		schedule, err := b.cron.parser.Parse(c.expression)
		if err == nil {
			nextExecution = schedule.Next(time.Now()).Format("2006-01-02 15:04")
		}

		data := map[string]string{
			"<id>":             c.id,
			"<expression>":     c.expression,
			"<message>":        c.message,
			"<is_limited>":     strconv.FormatBool(c.isLimited),
			"<exec_count>":     strconv.FormatInt(int64(c.execCount), 10),
			"<exec_limit>":     strconv.FormatInt(int64(c.execLimit), 10),
			"<next_execution>": nextExecution,
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
	err = b.cron.add(id, expression, message, 0, 0, false, b)
	if err != nil {
		b.logger.Printf("cronUpdate: %w", err)
		b.privmsgf(b.IRC.CronErr)
	}

	// Send a notice that the cron job was updated.
	b.privmsgf(b.IRC.CronMsgUpdate)
}
