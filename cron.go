package main

import (
	"strconv"
	"strings"
	"sync"

	_cron "github.com/robfig/cron/v3"
)

// newCron returns a new cron structure.
func newCron() *cron {
	return &cron{
		cron:   _cron.New(),
		jobs:   make(map[string]_cron.EntryID),
		parser: _cron.NewParser(_cron.Minute | _cron.Hour | _cron.Dom | _cron.Month | _cron.Dow),
	}
}

// cron holds the internal data structures needed to run the cron lib.
type cron struct {
	// cron holds a reference to the cron object.
	cron *_cron.Cron

	// jobs is a map of all the existing jobs, the key is the database id
	// of the job and the value is the cron entry id.
	jobs map[string]_cron.EntryID

	// mu is a mutex that will be used to make jobs insert/delete safe.
	mu sync.Mutex

	// parser holds the reference to the parsing instance.
	parser _cron.Parser
}

// add adds a new entry to the cron runner.
func (c *cron) add(id, expression, message string, execCount, execLimit int, isLimited bool, bot *bot) error {
	// Acquire a lock and release it when we return.
	c.mu.Lock()
	defer c.mu.Unlock()

	// The id is already in the jobs map, reaturn early.
	if _, ok := c.jobs[id]; ok {
		return nil
	}

	// Add it with the corresponding ID received by the cron lob.
	cronJob := newCronJob(bot, id, message, execCount, execLimit, isLimited)
	entryID, err := c.cron.AddJob(expression, cronJob)
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

// newCronJob returns a new cron job.
func newCronJob(bot *bot, id, message string, execCount, execLimit int, isLimited bool) *cronJob {
	return &cronJob{
		bot:       bot,
		id:        id,
		message:   message,
		execCount: execCount,
		execLimit: execLimit,
		isLimited: isLimited,
	}
}

// cronJob holds the data that each cron job is required to provide.
type cronJob struct {
	// bot holds a reference to the bot.
	bot *bot

	// id is the internal id of the bot (the database id).
	id string

	// message holds the message that will be sent back to the channel
	// when the cron job is executed.
	message string

	// isLimited is set to true if there's a limit on how many times the
	// job can be executed.
	isLimited bool

	// execCount keeps track of how many times cron job has been executed.
	execCount int

	// mu will be used to safely increment the exec count.
	mu sync.Mutex

	// execLimit defines how many times the job can be executed before
	// it will be stopped, if set to zero it will be keep running forever,
	// which is the default state.
	execLimit int
}

// Run implements the Job interface.
func (cj *cronJob) Run() {
	// Acquire a lock, increment the execution count and release it.
	cj.mu.Lock()
	cj.execCount = cj.execCount + 1
	cj.mu.Unlock()

	// Increment the exec_count in the database as well.
	stmt, err := cj.bot.prepare("UPDATE cron SET exec_count = $1 WHERE id = $2")
	if err != nil {
		cj.bot.logger.Printf("cronJobRun: %w", err)
		return
	}
	defer stmt.Close()

	// Execute the UPDATE statement.
	_, err = stmt.Exec(cj.execCount, cj.id)
	if err != nil {
		cj.bot.logger.Printf("cronJobRun: %w", err)
		return
	}

	message := cj.message
	i := strings.Index(message, cj.bot.IRC.CronGrammarMsgRandomWho)
	for i != -1 {
		message = message[0:i] + cj.bot.rndName() + message[i+len(cj.bot.IRC.CronGrammarMsgRandomWho):]
		i = strings.Index(message, cj.bot.IRC.CronGrammarMsgRandomWho)
	}

	// Replace all occurences of <giphy> with a gif from giphy.
	i = strings.Index(message, cj.bot.IRC.CronGrammarGiphy)
	for i != -1 {
		if url, _ := cj.bot.giphyRandom(); url != "" {
			message = message[0:i] + url + message[i+len(cj.bot.IRC.CronGrammarGiphy):]
		}
		i = strings.Index(message, cj.bot.IRC.CronGrammarGiphy)
	}

	// Replace all <giphy search="<query>"> with replies from the giphy API.
	for _, matches := range cronGrammarGiphySearchRegexp.FindAllStringSubmatch(message, -1) {
		if url, _ := cj.bot.giphySearch(matches[1]); url != "" {
			message = strings.Replace(message, matches[0], url, 1)
		} else {
			message = strings.Replace(message, matches[0], "", 1)
		}
	}

	// Replace all <tenor search="<query>"> with replies from the Tenor API.
	for _, matches := range cronGrammarTenorSearchRegexp.FindAllStringSubmatch(message, -1) {
		if url, _ := cj.bot.tenorSearch(matches[1]); url != "" {
			message = strings.Replace(message, matches[0], url, 1)
		} else {
			message = strings.Replace(message, matches[0], "", 1)
		}
	}

	// Send message to the channel and replace the placeholders with the
	// actual values.
	cj.bot.privmsgph(message, map[string]string{
		cj.bot.IRC.CronGrammarMsgIsLimited: strconv.FormatBool(cj.isLimited),
		cj.bot.IRC.CronGrammarMsgExecCount: strconv.FormatInt(int64(cj.execCount), 10),
		cj.bot.IRC.CronGrammarMsgExecLimit: strconv.FormatInt(int64(cj.execLimit), 10),
	})

	// Execution count has reached the limit, terminate the job.
	if cj.isLimited && cj.execCount >= cj.execLimit {
		cj.bot.cron.delete(cj.id)
	}
}
