package main

import (
	"os"

	"github.com/jimjibone/woodhouse-4/log"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                 "woodhouse",
		Usage:                "Runs the woodhouse core",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "config",
				Usage:     "Load configuration from `DIR`",
				EnvVars:   []string{"WOODHOUSE_CONFIG"},
				Value:     "woodhouse-5.yaml",
				TakesFile: true,
			},
		},
		Action: func(args *cli.Context) error {
			log.SetOptions(log.WithExitOnFatal(false), log.WithMinLevel(log.InfoLevel))

			log.Debugf("debug format %d %f 0x%x", 1, 2.3, 25)
			log.Infof("info format %d %f 0x%x", 1, 2.3, 25)
			log.Warnf("warn format %d %f 0x%x", 1, 2.3, 25)
			log.Errorf("error format %d %f 0x%x", 1, 2.3, 25)
			log.Fatalf("fatal format %d %f 0x%x", 1, 2.3, 25)

			log.Debugln("debug line", 1, 2.3, 25)
			log.Infoln("info line", 1, 2.3, 25)
			log.Warnln("warn line", 1, 2.3, 25)
			log.Errorln("error line", 1, 2.3, 25)
			log.Fatalln("fatal line", 1, 2.3, 25)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
