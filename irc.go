package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/osm/irc"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// VERSION will be set to the current git commit id on build.
var VERSION string

// initIRC connects to the IRC server that is defined in the configuration.
// The connection is launched within a goroutine. So this function will not
// block, so we have a wait group that controls the IRC life cycle.
func (b *bot) initIRC() {
	// Make sure that all the below options are set in the configuration.
	if b.IRC.Address == "" {
		b.logger.Fatalf("irc.address is missing in %v", b.configFile)
	}
	if b.IRC.Channel == "" {
		b.logger.Fatalf("irc.channel is missing in %v", b.configFile)
	}
	if b.IRC.Nick == "" {
		b.logger.Fatalf("irc.nick is missing in %v", b.configFile)
	}
	if b.IRC.RealName == "" {
		b.logger.Fatalf("irc.realName is missing in %v", b.configFile)
	}
	if b.IRC.User == "" {
		b.logger.Fatalf("irc.user is missing in %v", b.configFile)
	}
	if b.IRC.Version == "" {
		b.logger.Fatalf("irc.version is missing in %v", b.configFile)
	}

	// We'll set the grace period to 750 ms by default if no value was set
	// in the config.
	if b.IRC.GracePeriod == 0 {
		b.IRC.GracePeriod = 750
	}

	// Prepare an array of irc options.
	opts := []irc.Option{
		irc.WithAddr(b.IRC.Address),
		irc.WithChannel(b.IRC.Channel),
		irc.WithDebug(),
		irc.WithNick(b.IRC.Nick),
		irc.WithRealName(b.IRC.RealName),
		irc.WithUser(b.IRC.User),
		irc.WithVersion(fmt.Sprintf("%s %s", b.IRC.Version, VERSION)),
	}
	b.IRC.client = irc.NewClient(opts...)

	b.IRC.client.Handle("JOIN", b.operatorsHandler)

	// Event handlers that are needed for the names map.
	b.IRC.client.Handle("353", b.handleNamesReply)
	b.IRC.client.Handle("JOIN", b.handleNamesAdd)
	b.IRC.client.Handle("PART", b.handleNamesRemove)
	b.IRC.client.Handle("QUIT", b.handleNamesRemove)
	b.IRC.client.Handle("NICK", b.handleNamesChange)

	if b.IRC.RejoinOnKick {
		b.IRC.client.Handle("KICK", b.kickHandler)
	}

	if b.IRC.EnableFloodProt {
		b.initFloodProt()
		b.initFloodProtDefaults()
		b.IRC.client.Handle("PRIVMSG", b.floodProtHandler)
	}

	if b.IRC.EnableUpdateNotifier {
		b.initUpdateNotifier()
		go b.updateNotifierHandler()
	}

	if b.IRC.EnableSupernytt {
		b.initSupernytt()
		go b.supernyttHandler()
	}

	if b.IRC.EnableQuiz {
		b.initQuizDefaults()
		b.IRC.client.Handle("PRIVMSG", b.quizHandler)
	}

	if b.IRC.EnableCommands {
		b.initCommandDefaults()
		b.IRC.client.Handle("PRIVMSG", b.commandHandler)
	}

	if b.IRC.EnableURLCheck {
		b.initURLCheckDefaults()
		b.IRC.client.Handle("PRIVMSG", b.urlCheckHandler)
	}

	if b.IRC.EnableLogging {
		b.IRC.client.Handle("PRIVMSG", b.loggingHandler)
	}

	if b.IRC.EnableLyssnar {
		b.initLyssnarDefaults()
		b.IRC.client.Handle("PRIVMSG", b.lyssnarHandler)
	}

	if b.IRC.EnableGiphy {
		b.initGiphyDefaults()
		b.IRC.client.Handle("PRIVMSG", b.giphyHandler)
	}

	if b.IRC.EnableTenor {
		b.initTenorDefaults()
		b.IRC.client.Handle("PRIVMSG", b.tenorHandler)
	}

	if b.IRC.EnableChattistik {
		b.initChattistikDefaults()
		b.IRC.client.Handle("PRIVMSG", b.chattistikHandler)
	}

	if b.IRC.EnableFactoid {
		b.initFactoidDefaults()
		b.IRC.client.Handle("PRIVMSG", b.factoidHandler)
	}

	if b.IRC.EnableWeather {
		b.initWeatherDefaults()
		b.IRC.client.Handle("PRIVMSG", b.weatherHandler)
	}

	if b.IRC.EnableCron {
		b.initCronDefaults()
		b.IRC.client.Handle("PRIVMSG", b.cronHandler)
	}

	// This goroutine handles the connection to the IRC server. The IRC
	// library will automatically try to reconnect if the connection dies
	// for some reason.
	go func() {
		if err := b.IRC.client.Connect(); err != nil {
			fmt.Printf("IRC connection died: %v\n", err)
		}

		b.mainWG.Done()
	}()
}

// preventSpam checks when the last message was sent and makes sure that the
// value defined in "GracePeriod" has been passed before we we proceed with
// the execution. This is used so that we don't spam the IRC server with
// messages too fast.
func (b *bot) preventSpam() {
	// Acquire the lock and close it when we are done.
	b.IRC.lastSentMessageMu.Lock()
	defer b.IRC.lastSentMessageMu.Unlock()

	// Get the current time and subtract it with when the last message was
	// sent.
	last := time.Now().Sub(b.IRC.lastSentMessage)

	// If the last message was sent prior to the grace period we'll sleep
	// for the remaining time.
	if last < b.IRC.gracePeriod {
		time.Sleep(b.IRC.gracePeriod - last)
	}

	// Set the last sent message to now.
	b.IRC.lastSentMessage = time.Now()
}

// privmsg sends the given message back to the channel set from the
// configuration.
func (b *bot) privmsg(msg string) {
	b.preventSpam()
	b.IRC.client.Privmsg(b.IRC.Channel, msg)
}

// privmsgph replaces the keys of the phs map with the values and sends the
// message to the configured channel.
func (b *bot) privmsgph(msg string, phs map[string]string) {
	b.preventSpam()

	for k, v := range phs {
		msg = strings.ReplaceAll(msg, k, v)
	}

	b.IRC.client.Privmsg(b.IRC.Channel, msg)
}

// privmsgpht replaces the keys of the phs map with the values and sends the
// message to the specified target.
func (b *bot) privmsgpht(msg, target string, phs map[string]string) {
	b.preventSpam()

	for k, v := range phs {
		msg = strings.ReplaceAll(msg, k, v)
	}

	b.IRC.client.Privmsg(target, msg)
}

// action sends the given message back to the channel set from the
// configuration as an ACTION message.
func (b *bot) action(msg string) {
	b.preventSpam()
	b.IRC.client.Privmsg(b.IRC.Channel, "\u0001ACTION "+msg+"\u0001")
}

// rndName returns a random name from the names map.
func (b *bot) rndName() string {
	i := 0
	stop := rand.Intn(len(b.IRC.names))

	for n := range b.IRC.names {
		if i == stop {
			return n
		}
		i++
	}

	return b.IRC.Nick
}

// shouldIgnore determines if the given message should be ignored by the bot
// or not.
func (b *bot) shouldIgnore(m *irc.Message) bool {
	h := parseHost(m)

	// Check if the message comes from a permanently ignored user.
	for _, r := range b.IRC.ignorePerm {
		if r.Match([]byte(h)) {
			return true
		}
	}

	if b.IRC.EnableFloodProt {
		// Acquire the ignore mutex lock before we check if the dynamic ignore
		// map contains the host.
		b.IRC.ignoreDynMu.Lock()
		defer b.IRC.ignoreDynMu.Unlock()
		if _, ignored := b.IRC.ignoreDyn[h]; ignored {
			return true
		}
	}

	return false
}
