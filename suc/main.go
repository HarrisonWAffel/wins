package main

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-colorable"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/suc/pkg"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	/*
		Likely need to do things in a specific order

			1. Determine if we are modifying any binaries, if so we likely need to stop the service
			2. Update the config file as needed
			3. Update the binaries as needed
			4. configure the service start types and dependencies
			5. Start the service back up
			6. If an error is encountered, revert changes and restart service to last known working state

		The initial work for the cve fix will only be implementing 1, 2, 5, and 6 - 3 and 4 will come in 2.10
	*/

	defer panics.Log()
	app := cli.NewApp()
	app.Version = defaults.AppVersion
	app.Name = defaults.WindowsSUCName
	app.Usage = "A way to modify rancher-wins via the Rancher System Upgrade Controller"
	app.Action = pkg.Run
	app.Description = fmt.Sprintf(`%s (%s)`, defaults.WindowsSUCName, defaults.AppCommit)
	app.Writer = colorable.NewColorableStdout()
	app.ErrWriter = colorable.NewColorableStderr()
	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if c.Bool("quiet") {
			logrus.SetOutput(io.Discard)
		}
		logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, FullTimestamp: true})
		logrus.SetOutput(c.App.Writer)
		return nil
	}

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Turn on verbose debug logging",
		},
		&cli.BoolFlag{
			Name:  "quiet",
			Usage: "Turn on off all logging",
		},
		&cli.BoolFlag{
			Name:  "update-config",
			Usage: "Update the rancher-wins config file using environment variables",
		},

		// TODO: The below flags.
		//&cli.BoolFlag{
		//	Name:  "upgrade",
		//	Usage: "Upgrade wins.exe using the embedded binary",
		//},
		//&cli.BoolFlag{
		//	Name: "wins-delayed-start",
		//	Usage: "Configure the rancher-wins service with a start type of auto-delayed. If this flag is not provided, the start type will default to auto",
		//},
		//&cli.StringFlag{
		//	Name: "create-service-dependency",
		//	Usage: "Create a service dependency between rancher-wins and the service provided in this flag. Providing an empty string will remove all service dependencies",
		//},
	}

	if err := app.Run(os.Args); err != nil && err != io.EOF {
		logrus.Fatal(err)
	}
}
