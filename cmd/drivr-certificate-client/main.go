package main

import (
	"os"

	"github.com/urfave/cli/v2"

	log "github.com/sirupsen/logrus"
)

var (
	version      = "dev"
	logLevelFlag = &cli.StringFlag{
		Name:    "log-level",
		Usage:   "Minimum level of log events which should be displayed.",
		Value:   "INFO",
		EnvVars: []string{"DRIVR_LOG_LEVEL"},
	}
)

func initLogging(ctx *cli.Context) {
	log.SetOutput(os.Stderr)
	level, err := log.ParseLevel(ctx.String(logLevelFlag.Name))
	if err != nil {
		level = log.DebugLevel
		log.Warnf("Invalid log level '%s', defaulting to '%s'", ctx.String(logLevelFlag.Name), level)
	}
	log.SetLevel(level)
}

func main() {
	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "drivr-certificate-client",
		Description:          "drivr-certificate-client is a command line tool for creating certificates",
		Flags: []cli.Flag{
			logLevelFlag,
		},
		Before: func(ctx *cli.Context) error {
			initLogging(ctx)
			return nil
		},
		Commands: []*cli.Command{
			createCommand(),
			fetchCommand(),
			completionCommand(),
			dumpCommand(),
			validateCommand(),
		},
		Version: version,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
