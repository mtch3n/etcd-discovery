package main

import (
	"github.com/mtch3n/etcd-discovery/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var opt = &Options{}

// todo: add endpoint back to global

type Options struct {
	Debug    bool
	LogLevel string
}

func main() {

	rootCmd := NewEtcdDiscoveryCmd()

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func NewEtcdDiscoveryCmd() *cobra.Command {
	app := &cobra.Command{
		Use:   "esd",
		Short: "etcd service discovery",
		Long:  "etcd service discovery is a service discovery tool based on etcd",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			setLoggerLevel()
		},
	}

	addFlags(app)

	app.AddCommand(
		cmd.NewCmdDiscovery(),
		cmd.NewCmdRegistry(),
		cmd.NewCmdStatus(),
	)

	return app
}

func addFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().BoolVarP(&opt.Debug, "debug", "d", false, "debug mode")
	rootCmd.PersistentFlags().StringVar(&opt.LogLevel, "log-level", "info", "Log level")
}

func setLoggerLevel() {
	var lvl log.Level

	if parsedLvl, err := log.ParseLevel(opt.LogLevel); err == nil {
		lvl = parsedLvl
	}

	if opt.Debug {
		lvl = log.TraceLevel
	}

	log.SetLevel(lvl)
}
