package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/osm/irc"
)

// initQuizDefaults sets default values for the quiz commands and messages.
func (b *bot) initQuizDefaults() {
	// Commands.
	if b.IRC.QuizCmd == "" {
		b.IRC.QuizCmd = "!quiz"
	}
	if b.IRC.QuizSubCmdStart == "" {
		b.IRC.QuizCmd = "start"
	}
	if b.IRC.QuizSubCmdStop == "" {
		b.IRC.QuizCmd = "stop"
	}
	if b.IRC.QuizSubCmdStats == "" {
		b.IRC.QuizCmd = "stats"
	}

	// Hint interval.
	if b.IRC.QuizHintInterval == 0 {
		b.IRC.QuizHintInterval = 15
	}

	// Messages.
	if b.IRC.QuizMsgNameDoesNotExist == "" {
		b.IRC.QuizMsgNameDoesNotExist = "<name> does not exist"
	}
	if b.IRC.QuizMsgLoadError == "" {
		b.IRC.QuizMsgLoadError = "unable to load <name>"
	}
	if b.IRC.QuizMsgAlreadyStarted == "" {
		b.IRC.QuizMsgAlreadyStarted = "a quiz is already started"
	}
	if b.IRC.QuizMsgQuestion == "" {
		b.IRC.QuizMsgQuestion = "<category>: <question>"
	}
	if b.IRC.QuizMsgHint == "" {
		b.IRC.QuizMsgHint = "hint: <text>"
	}
	if b.IRC.QuizMsgAnswer == "" {
		b.IRC.QuizMsgHint = "answer: <text>"
	}
	if b.IRC.QuizMsgCorrect == "" {
		b.IRC.QuizMsgCorrect = "correct! one point to <nick>"
	}
	if b.IRC.QuizMsgQuizEnd == "" {
		b.IRC.QuizMsgQuizEnd = "the quiz is over"
	}
}

// quizHandler handles all IRC related communication with the quiz bot.
func (b *bot) quizHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}

	if b.shouldIgnore(m) {
		return
	}

	// Handle the quiz IRC commands.
	if len(a.args) == 2 &&
		a.cmd == b.IRC.QuizCmd &&
		a.args[0] == b.IRC.QuizSubCmdStart {
		b.quizStart(a.args[1])
		return
	} else if len(a.args) == 1 &&
		a.cmd == b.IRC.QuizCmd &&
		a.args[0] == b.IRC.QuizSubCmdStop &&
		b.IRC.quizRound != nil {
		b.IRC.quizRound.stop()
		return
	}

	// No active quiz round, return immediately.
	if b.IRC.quizRound == nil {
		return
	}

	// Check if the given message was the correct answer for the question.
	b.IRC.quizRound.answer(a.nick, a.msg)
}

// quizStart starts a quiz with the given name.
func (b *bot) quizStart(name string) {
	// Don't allow a new quiz to be started if there are one running
	// already.
	if b.IRC.quizRound != nil {
		b.privmsg(b.IRC.QuizMsgAlreadyStarted)
		return
	}

	// Make sure that the name of the quiz exists in our database.
	_, exists := b.IRC.QuizSources[name]
	if !exists {
		b.privmsgph(b.IRC.QuizMsgNameDoesNotExist, map[string]string{
			"<name>": name,
		})
		return
	}

	// Initialize a new quiz round and pop the first question.
	b.IRC.quizRound = newQuizRound(b, name, 10)
	if b.IRC.quizRound != nil {
		b.IRC.quizRound.getQuestion()
	}
}

// quizQuestion defines the data structure that holds information about a
// question in the quiz.
type QuizQuestion struct {
	Category string `json:"category"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// quizRound defines the structure that holds all the data that is required
// for the quiz run.
type quizRound struct {
	// id is a random UUID that should be unique for each quiz round. It
	// is used only for the purpose of making the database data easier to
	// query.
	id string

	// name is the name of the quiz source that is defined in the
	// configuration file.
	name string

	// ch is a channel that is used to stop the hint goroutines early if
	// the correct answer has been given by a user.
	ch chan bool

	// mu is a mutex that will be used to make sure that we don't get a
	// race condition when answering questions.
	mu sync.Mutex

	// bot is just a pointer to the bot.
	bot *bot

	// stats holds information about the current quiz round.
	stats map[string]int

	// question is the current question that.
	question QuizQuestion

	// questions holds all the questions for the current quiz round.
	questions []QuizQuestion
}

// quizLoadFromFile reads the given file path into memory and returns a slice
// of quiz questions.
func quizLoadFromFile(filePath string) ([]QuizQuestion, error) {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("quizLoadFromFile: cant open quiz file %s, %w", filePath, err)
	}

	var questions []QuizQuestion
	err = json.Unmarshal(file, &questions)
	if err != nil {
		return nil, fmt.Errorf("quizLoadFromFile: cant decode quiz file, %w", err)
	}

	return questions, nil
}

// quizLoadFromHttp loads quiz questions from the given url.
func quizLoadFromHttp(url string) ([]QuizQuestion, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("quizLoadFromHttp: cant open url %s, %w", url, err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("quizLoadFromHttp: cant read body from %s, %w", url, err)
	}

	var questions []QuizQuestion
	err = json.Unmarshal(body, &questions)
	if err != nil {
		return nil, fmt.Errorf("quizLoadFromHttp: cant decode quiz from %s, %s, %w", url, body, err)
	}

	return questions, nil
}

// newQuizRound returns a new quizRound data structure.
func newQuizRound(bot *bot, name string, nQuestions int) *quizRound {
	var allQuestions []QuizQuestion
	var err error
	path := bot.IRC.QuizSources[name]

	// Load questions.
	if strings.HasPrefix(path, "http") {
		allQuestions, err = quizLoadFromHttp(path)
	} else {
		allQuestions, err = quizLoadFromFile(path)
	}
	if err != nil {
		bot.logger.Printf("%w", err)
		bot.privmsgph(bot.IRC.QuizMsgLoadError, map[string]string{
			"<name>": name,
		})
		return nil
	}

	// Make sure that we don't pick too many questions.
	quizLen := len(allQuestions)
	l := nQuestions
	if l > quizLen {
		l = quizLen
	}

	// We pick nQuestions number of questions from the source and store
	// them inside the questions slice.
	var questions []QuizQuestion
	for i := 0; i < l; i++ {
		questions = append(questions, allQuestions[rand.Int()%quizLen])
	}

	// Return a new quiz round with the randomly picked questions.
	return &quizRound{
		id:   newUUID(),
		name: name,

		bot:   bot,
		ch:    make(chan bool),
		stats: make(map[string]int),

		questions: questions,
	}
}

// answer checks whether or not the given answer is the correct answer for the
// current question.
func (qr *quizRound) answer(n, a string) {
	// Acquire a lock before we check whether or not the answer is
	// correct.
	qr.mu.Lock()
	defer qr.mu.Unlock()

	// The quiz round might have been completed by another goroutine, so
	// let's check that before we proceed.
	if qr.bot.IRC.quizRound == nil {
		return
	}

	// Incorrect answer, return early.
	if strings.ToLower(a) != strings.ToLower(qr.question.Answer) {
		return
	}

	// Closing the channel will stop all running goroutines, which we want
	// at this point. So, let's close it and create a new channel for the
	// next question.
	close(qr.ch)
	qr.ch = make(chan bool)

	// Print the correct message to the channel.
	qr.bot.privmsgph(qr.bot.IRC.QuizMsgCorrect, map[string]string{
		"<nick>": n,
		"<text>": a,
	})

	// Increment the quiz round stats
	if _, ok := qr.stats[n]; !ok {
		qr.stats[n] = 0
	}
	qr.stats[n]++

	// Also, add stats to the database.
	stmt, err := qr.bot.prepare("INSERT INTO quiz_stat (id, nick, quiz_round_id, quiz_name, category, question, answer, inserted_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		qr.bot.logger.Printf("quizAnswer: %w", err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		newUUID(),
		n,
		qr.id,
		qr.name,
		qr.question.Category,
		qr.question.Question,
		qr.question.Answer,
		newTimestamp(),
	)
	if err != nil {
		qr.bot.logger.Printf("quizAnswer: %w", err)
	}

	// Pop a new qusetion.
	qr.getQuestion()
}

// hint sleeps for t number of seconds before it writes the given hint back to
// the channel. If newQuestion is set to true a new question will be popped
// when the timeout occurs.
func (qr *quizRound) hint(t time.Duration, ph, text string, newQuestion bool) {
	select {
	case <-time.After(t * time.Second):
		qr.bot.privmsgph(ph, map[string]string{
			"<text>": text,
		})

		if newQuestion {
			qr.getQuestion()
		}
	case <-qr.ch:
	}
}

// getQuestion takes a question from the slice of questions and returns it to
// the channel.
func (qr *quizRound) getQuestion() {
	if len(qr.questions) > 0 {
		// Pop one question from the array.
		qr.question, qr.questions = qr.questions[0], qr.questions[1:]

		// Write the question to the channel.
		qr.bot.privmsgph(qr.bot.IRC.QuizMsgQuestion, map[string]string{
			"<category>": qr.question.Category,
			"<question>": qr.question.Question,
		})

		// Start the hint and correct answer goroutines for the
		// question.
		go qr.hint(
			qr.bot.IRC.QuizHintInterval,
			qr.bot.IRC.QuizMsgHint,
			maskText(qr.question.Answer),
			false,
		)
		go qr.hint(
			qr.bot.IRC.QuizHintInterval*2,
			qr.bot.IRC.QuizMsgHint,
			hintText(qr.question.Answer),
			false,
		)
		go qr.hint(
			qr.bot.IRC.QuizHintInterval*3,
			qr.bot.IRC.QuizMsgAnswer,
			qr.question.Answer,
			true,
		)

	} else {
		// No more questions left, stop the quiz.
		qr.bot.IRC.quizRound.stop()
	}
}

// stops the current quiz round early.
func (qr *quizRound) stop() {
	// Quiz is over, present the results.
	qr.bot.privmsg(qr.bot.IRC.QuizMsgQuizEnd)

	// Construct a map of the stats but where the key is the
	// number of points instead of the nick.
	count := make(map[int]string)
	for k, v := range qr.stats {
		if _, ok := count[v]; !ok {
			count[v] = k
		} else {
			count[v] = fmt.Sprintf("%s, %s", count[v], k)
		}
	}

	// Sort the count map so that we can output the stats in a nicer way.
	sortedKeys := make([]int, 0, len(count))
	for k := range count {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedKeys)))

	// Output the results to the channel.
	for _, k := range sortedKeys {
		qr.bot.privmsg(fmt.Sprintf("%d: %s", k, count[k]))
	}

	close(qr.ch)
	qr.bot.IRC.quizRound = nil
}

// maskText replaces all characters of the string with an asterisk unless it's
// a space.
func maskText(s string) string {
	ret := ""

	for _, r := range s {
		if r == ' ' {
			ret += " "
		} else {
			ret += "*"
		}
	}

	return ret
}

// hintText replaces half of the text with *, unless it's a space, spaces are
// left untouched.
func hintText(s string) string {
	ret := ""
	c := 0
	l := len(s)

	for i, r := range s {
		if r == ' ' {
			ret += " "
		} else if c <= l/2 && i%l >= rand.Intn(l) {
			c += 1
			ret += string(r)
		} else {
			ret += "*"
		}
	}

	return ret
}
