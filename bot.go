package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/osm/irc"
	"github.com/osm/jsonc"
)

// bot is the main structure that binds all bot methods together. It does also
// contain the configuration and objects that needs to be accessible during
// the bots lifetime.
type bot struct {
	// mainWG keeps track of the goroutines needed to keep the bot alive.
	// Currently this is just the IRC main loop, but this wait group
	// should keep track of all long living goroutines that the bot needs
	// to function correctly.
	mainWG sync.WaitGroup

	// Logger for all bot related errors.
	logger *log.Logger

	// configFile keeps a copy of the configuration file path.
	configFile string

	// The cron and cron parser objects are kept here, we might want to
	// extend the cron implementation to run more than just IRC related
	// things in the future.
	cron *cron

	DB struct {
		client *sql.DB

		// Path should point to the location of the SQLite database
		// file.
		Path string `json:"path"`

		// A generic error to display in case of database problems.
		Err string `json:"err"`
	}

	HTTP struct {
		// Logger for all HTTP related things.
		logger *log.Logger

		// Toggle the HTTP server on/off.
		EnableHTTP bool `json:"enableHTTP"`

		// Listen address and port.
		Address string `json:"address"`
		Port    string `json:"port"`

		// EchoRoute defines the echo route, which can be used to post
		// message to the IRC channel.
		EnableEcho bool   `json:"enableEcho"`
		EchoRoute  string `json:"echoRoute"`
		EchoMethod string `json:"echoMethod"`
	}

	IRC struct {
		client *irc.Client

		// This is set to time.Now() each time a message is sent. We
		// will check that time.Now() - lastSentMessage is greater
		// than the value defined in "GracePeriod".
		lastSentMessage time.Time

		// lastSentMessageMu holds the mutex for the last sent
		// message.
		lastSentMessageMu sync.Mutex

		// Grace period is how long we should wait before we are
		// allowed to send a new message to the server. The value is
		// defined in milliseconds.
		GracePeriod int `json:"gracePeriod"`

		// This will be the time.Duration equivalent of the integer
		// GracePeriod. We'll convert this right away when the
		// application is started.
		gracePeriod time.Duration

		// Rejoin the channel if the bot is kicked.
		RejoinOnKick bool `json:"rejoinOnKick"`

		// Connection related settings.
		Address  string `json:"address"`
		Channel  string `json:"channel"`
		Nick     string `json:"nick"`
		RealName string `json:"realName"`
		User     string `json:"user"`
		Version  string `json:"version"`

		// names contains all the nicks in the channel.  The map is
		// updated on JOIN and PART messages.
		names map[string]bool

		// namesMu holds the mutex for the names map.
		namesMu sync.Mutex

		// Update notifier.
		EnableUpdateNotifier bool     `json:"enableUpdateNotifier"`
		UpdateNotifierMsg    string   `json:"updateNotifierMsg"`
		UpdateNotifierNames  []string `json:"updateNotifierNames"`

		// Supernytt - news in Swedish from aftonbladet.
		EnableSupernytt         bool   `json:"enableSupernytt"`
		SupernyttGrammarMessage string `json:"supernyttGrammarMessage"`

		// Quiz.
		EnableQuiz       bool              `json:"enableQuiz"`
		QuizSources      map[string]string `json:"quizSources"`
		quizSourcesCache map[string][]QuizQuestion

		QuizCmd         string `json:"quizCmd"`
		QuizSubCmdStart string `json:"quizSubCmdStart"`
		QuizSubCmdStop  string `json:"quizSubCmdStop"`
		QuizSubCmdStats string `json:"quizSubCmdStats"`

		QuizHintInterval time.Duration `json:"quizHintInterval"`

		QuizMsgNameDoesNotExist string `json:"quizMsgNameDoesNotExist"`
		QuizMsgLoadError        string `json:"quizMsgLoadError"`
		QuizMsgAlreadyStarted   string `json:"quizMsgAlreadyStarted"`
		QuizMsgQuestion         string `json:"quizMsgQuestion"`
		QuizMsgHint             string `json:"quizMsgHint"`
		QuizMsgAnswer           string `json:"quizMsgAnswer"`
		QuizMsgCorrect          string `json:"quizMsgCorrect"`
		QuizMsgQuizEnd          string `json:"quizMsgQuizEnd"`

		quizRound *quizRound

		EnableLyssnar                bool              `json:"enableLyssnar"`
		LyssnarCmd                   string            `json:"lyssnarcmd"`
		Lyssnare                     map[string]string `json:"lyssnare"`
		LyssnarErr                   string            `json:"lyssnarErr"`
		LyssnarErrUserNotConfigured  string            `json:"lyssnarErrUserNotConfigured"`
		LyssnarMsg                   string            `json:"lyssnarMsg"`
		LyssnarMsgUserIsNotListening string            `json:"lyssnarMsgUserIsNotListening"`

		EnableGiphy          bool   `json:"enableGiphy"`
		GiphyCmd             string `json:"giphyCmd"`
		GiphyLang            string `json:"giphyLang"`
		GiphyAPIKey          string `json:"giphyAPIKey"`
		GiphyMsgNothingFound string `json:"giphyMsgNothingFound"`

		EnableTenor          bool   `json:"enableTenor"`
		TenorCmd             string `json:"tenorCmd"`
		TenorLang            string `json:"tenorLang"`
		TenorAPIKey          string `json:"tenorAPIKey"`
		TenorMsgNothingFound string `json:"tenorMsgNothingFound"`

		ChattistikCmd          string `json:"chattistikCmd"`
		ChattistikCmdToday     string `json:"chattistikCmdToday"`
		ChattistikCmdYesterday string `json:"chattistikCmdYesterday"`
		EnableChattistik       bool   `json:"enableChattistik"`
		EnableLogging          bool   `json:"enableLogging"`

		EnableCron bool `json:"enableCron"`

		CronCmd            string `json:"cronCmd"`
		CronSubCmdAdd      string `json:"cronSubCmdAdd"`
		CronSubCmdAddLimit string `json:"cronSubCmdAddLimit"`
		CronSubCmdDelete   string `json:"cronSubCmdDelete"`
		CronSubCmdList     string `json:"cronSubCmdList"`
		CronSubCmdUpdate   string `json:"cronSubCmdUpdate"`

		CronErr       string `json:"cronErr"`
		CronMsgAdd    string `json:"cronMsgAdd"`
		CronMsgDelete string `json:"cronMsgDelete"`
		CronMsgList   string `json:"cronMsgList"`
		CronMsgUpdate string `json:"cronMsgUpdate"`

		CronGrammarMsgExecCount string `json:"cronGrammarMsgExecCount"`
		CronGrammarMsgExecLimit string `json:"cronGrammarMsgExecLimit"`
		CronGrammarMsgIsLimited string `json:"cronGrammarMsgIsLimited"`
		CronGrammarMsgRandomWho string `json:"cronGrammarMsgRandomWho"`
		CronGrammarGiphy        string `json:"cronGrammarGiphy"`
		CronGrammarGiphySearch  string `json:"cronGrammarGiphySearch"`
		CronGrammarTenorSearch  string `json:"cronGrammarTenorSearch"`

		CommandErrExec string            `json:"commandErrExec"`
		Commands       map[string]string `json:"commands"`
		CommandsStatic map[string]string `json:"commandsStatic"`
		EnableCommands bool              `json:"enableCommands"`
		commands       map[string]command

		EnableWeather bool   `json:"enableWeather"`
		WeatherAPIKey string `json:"weatherAPIKey"`
		WeatherCmd    string `json:"weatherCmd"`
		WeatherErr    string `json:"weatherErr"`
		WeatherMsg    string `json:"weatherMsg"`

		EnableURLCheck bool   `json:"enableURLCheck"`
		URLCheckMsg    string `json:"urlCheckMsg"`

		operators []*regexp.Regexp
		Operators []string `json:"operators"`

		ignoreDyn   map[string]bool
		ignoreDynMu sync.Mutex
		ignorePerm  []*regexp.Regexp
		Ignore      []string `json:"ignore"`

		EnableFloodProt        bool   `json:"enableFloodProt"`
		FloodProtTimeThreshold int    `json:"floodProtTimeThreshold"`
		FloodProtCmdThreshold  int    `json:"floodProtCmdThreshold"`
		FloodProtIgnoreTime    int    `json:"floodProtIgnoreTime"`
		FloodProtMsgIgnore     string `json:"floodProtMsgIgnore"`
		FloodProtMsgUnignore   string `json:"floodProtMsgUnignore"`

		EnableFactoid bool `json:"enableFactoid"`
		FactoidRate   int  `json:"factoidRate"`

		FactoidCmd                string `json:"factoidCmd"`
		FactoidSubCmdAdd          string `json:"factoidSubCmdAdd"`
		FactoidSubCmdAddDelimiter string `json:"factoidSubCmdAddDelimiter"`
		FactoidSubCmdDelete       string `json:"factoidSubCmdDelete"`
		FactoidSubCmdSnoop        string `json:"factoidSubCmdSnoop"`
		FactoidSubCmdCount        string `json:"factoidSubCmdCount"`

		FactoidGrammarAction      string `json:"factoidGrammarAction"`
		FactoidGrammarRandomWho   string `json:"factoidGrammarRandomWho"`
		FactoidGrammarRandomWord  string `json:"factoidGrammarRandomWord"`
		FactoidGrammarReply       string `json:"factoidGrammarReply"`
		FactoidGrammarWho         string `json:"factoidGrammarWho"`
		FactoidGrammarGiphy       string `json:"factoidGrammarGiphy"`
		FactoidGrammarGiphySearch string `json:"factoidGrammarGiphySearch"`
		FactoidGrammarTenorSearch string `json:"factoidGrammarTenorSearch"`

		FactoidMsgAdd    string `json:"factoidMsgAdd"`
		FactoidMsgDelete string `json:"factoidMsgDelete"`
		FactoidMsgIs     string `json:"factoidMsgIs"`
		FactoidMsgSnoop  string `json:"factoidMsgSnoop"`
		FactoidMsgCount  string `json:"factoidMsgCount"`

		PastebinAPIKey string `json:"pastebinApiKey"`
	}
}

// newBotFromConfig reads the configuration file from the given path and
// initializes a new bot structure by the values that exists in the
// configuration. If something goes wrong here it means that the config has
// some incorrect value and the caller of this function should fail loudly.
func newBotFromConfig(c string) (*bot, error) {
	file, err := ioutil.ReadFile(c)
	if err != nil {
		return nil, fmt.Errorf("error: can't open config, %v", err)
	}

	bot := bot{}
	decoder := json.NewDecoder(strings.NewReader(jsonc.ToJSON(string(file))))
	if err = decoder.Decode(&bot); err != nil {
		return nil, fmt.Errorf("error: can't decode config, %v", err)
	}

	// Make sure there's a DB error present.
	if bot.DB.Err == "" {
		bot.DB.Err = "a database error occured, contact admin"
	}

	bot.configFile = c

	// Initialize the names map.
	bot.IRC.names = make(map[string]bool)

	// Convert the Operators array into a map so lookups will be
	// efficient.
	if len(bot.IRC.Operators) > 0 {
		for _, o := range bot.IRC.Operators {
			bot.IRC.operators = append(bot.IRC.operators, regexp.MustCompile(o))
		}
	}

	// Create a map for the dynamic ignores and if there are permanent
	// ignores we'll add them as regexps to the ignorePerm array.
	if bot.IRC.EnableFloodProt {
		bot.IRC.ignoreDyn = make(map[string]bool)
	}
	if len(bot.IRC.Ignore) > 0 {
		for _, r := range bot.IRC.Ignore {
			bot.IRC.ignorePerm = append(bot.IRC.ignorePerm, regexp.MustCompile(r))
		}
	}

	// Convert Commands and CommandsStatic to the internal command
	// structure.
	bot.IRC.commands = make(map[string]command)
	if len(bot.IRC.Commands) > 0 {
		for k, v := range bot.IRC.Commands {
			bot.IRC.commands[k] = parseCommand(true, v)
		}
	}
	if len(bot.IRC.CommandsStatic) > 0 {
		for k, v := range bot.IRC.CommandsStatic {
			bot.IRC.commands[k] = parseCommand(false, v)
		}
	}

	// Set the lastSentMessage to time.Now().
	bot.IRC.lastSentMessageMu.Lock()
	defer bot.IRC.lastSentMessageMu.Unlock()
	bot.IRC.lastSentMessage = time.Now()

	// Convert the integer grace period to time.Duration.
	bot.IRC.gracePeriod = time.Duration(bot.IRC.GracePeriod) * time.Millisecond

	// Initialize the loggers.
	bot.logger = log.New(os.Stdout, "BOT: ", log.LstdFlags)
	bot.HTTP.logger = log.New(os.Stdout, "HTTP: ", log.LstdFlags)

	return &bot, nil
}

// start fires up a new instance of the bot, it connects to the IRC server and
// starts handling requests immediately. This function is blocking so we won't
// get any errors returned back to us unless there's a problem with a database
// migration.
func (b *bot) start() error {
	// Set up the database connection. The initDB method will also run
	// migrations in the database.  So any change that you make to the
	// repository in db.go will be executed when the bot starts. If
	// there's an error in a migration we'll return an error. When this
	// occur we should terminate the execution of the bot immediately so
	// that we can address the migration issue before we launch the bot.
	if err := b.initDB(); err != nil {
		return err
	}

	// Initialize all the cron related objects and start processing of the
	// existing cron jobs.
	b.cron = newCron()
	b.initCron()

	// Calculate how many wait groups to wait for.
	wgs := 1
	if b.HTTP.EnableHTTP {
		wgs = wgs + 1
	}

	// The HTTP and IRC main loops runs in a goroutine So we'll add gw to
	// our wait group and wait until it completes.
	b.mainWG.Add(wgs)
	b.initIRC()

	// Start the HTTP server.
	if b.HTTP.EnableHTTP {
		b.initHTTP()
	}

	b.mainWG.Wait()

	return nil
}
