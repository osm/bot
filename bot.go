package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

	DB struct {
		client *sql.DB

		// Path should point to the location of the SQLite database
		// file.
		Path string `json:"path"`

		// A generic error to display in case of database problems.
		Err string `json:"err"`
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

		EnableLyssnar                bool              `json:"enableLyssnar"`
		LyssnarCmd                   string            `json:"lyssnarcmd"`
		Lyssnare                     map[string]string `json:"lyssnare"`
		LyssnarErr                   string            `json:"lyssnarErr"`
		LyssnarErrUserNotConfigured  string            `json:"lyssnarErrUserNotConfigured"`
		LyssnarMsg                   string            `json:"lyssnarMsg"`
		LyssnarMsgUserIsNotListening string            `json:"lyssnarMsgUserIsNotListening"`

		ChattistikCmd          string `json:"chattistikCmd"`
		ChattistikCmdToday     string `json:"chattistikCmdToday"`
		ChattistikCmdYesterday string `json:"chattistikCmdYesterday"`
		EnableChattistik       bool   `json:"enableChattistik"`
		EnableLogging          bool   `json:"enableLogging"`

		CommandErrExec string            `json:"commandErrExec"`
		Commands       map[string]string `json:"commands"`
		EnableCommands bool              `json:"enableCommands"`

		EnableWeather bool   `json:"enableWeather"`
		WeatherAPIKey string `json:"weatherAPIKey"`
		WeatherCmd    string `json:"weatherCmd"`
		WeatherErr    string `json:"weatherErr"`
		WeatherMsg    string `json:"weatherMsg"`

		EnableURLCheck bool   `json:"enableURLCheck"`
		URLCheckMsg    string `json:"urlCheckMsg"`

		operators map[string]bool
		Operators []string `json:"operators"`

		ignore map[string]bool
		Ignore []string `json:"ignore"`

		EnableFactoid bool `json:"enableFactoid"`
		FactoidRate   int  `json:"factoidRate"`

		FactoidCmdAdd          string `json:"factoidCmdAdd"`
		FactoidCmdAddDelimiter string `json:"factoidCmdAddDelimiter"`
		FactoidCmdDelete       string `json:"factoidCmdDelete"`
		FactoidCmdSnoop        string `json:"factoidCmdSnoop"`

		FactoidGrammarAction    string `json:"factoidGrammarAction"`
		FactoidGrammarRandomWho string `json:"factoidGrammarRandomWho"`
		FactoidGrammarReply     string `json:"factoidGrammarReply"`
		FactoidGrammarWho       string `json:"factoidGrammarWho"`

		FactoidMsgAdd    string `json:"factoidMsgAdd"`
		FactoidMsgDelete string `json:"factoidMsgDelete"`
		FactoidMsgIs     string `json:"factoidMsgIs"`
		FactoidMsgSnoop  string `json:"factoidMsgSnoop"`

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
		bot.IRC.operators = make(map[string]bool)
		for _, o := range bot.IRC.Operators {
			if _, ok := bot.IRC.operators[o]; !ok {
				bot.IRC.operators[o] = true
			}
		}
	}

	// Convert the Ignore array into a map to make lookups more
	// efficient.
	if len(bot.IRC.Ignore) > 0 {
		bot.IRC.ignore = make(map[string]bool)
		for _, o := range bot.IRC.Ignore {
			if _, ok := bot.IRC.ignore[o]; !ok {
				bot.IRC.ignore[o] = true
			}
		}
	}

	// Set the lastSentMessage to time.Now().
	bot.IRC.lastSentMessageMu.Lock()
	defer bot.IRC.lastSentMessageMu.Unlock()
	bot.IRC.lastSentMessage = time.Now()

	// Convert the integer grace period to time.Duration.
	bot.IRC.gracePeriod = time.Duration(bot.IRC.GracePeriod) * time.Millisecond

	// Initialize the logger.
	bot.logger = log.New(os.Stdout, "BOT: ", log.LstdFlags)

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

	// The IRC main loop runs in a goroutine So we'll add one to our wait
	// group and wait until it completes.
	b.mainWG.Add(1)
	b.initIRC()
	b.mainWG.Wait()

	return nil
}
