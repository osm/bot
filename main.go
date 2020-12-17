package main

import (
	"flag"
	"fmt"
	"os"
)

// main starts the bot. The only required flag is the -config flag which
// should point to the configuration file that you want to use.
func main() {
	configPath := flag.String("config", "", "config file path")
	version := flag.Bool("version", false, "display current version")
	schemaOnly := flag.Bool("init-schema-only", false, "init db schema and exit")
	flag.Parse()

	if *version {
		fmt.Fprintf(os.Stdout, "%s\n", VERSION)
		os.Exit(1)
	}

	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "-config is required\n")
		os.Exit(1)
	}

	var bot *bot
	var err error
	if bot, err = newBotFromConfig(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if *schemaOnly == true {
		bot.initDB()
		os.Exit(0)
	}

	if err = bot.start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
