package main

import (
	"github.com/codegangsta/cli"
	"github.com/glerchundi/fix-attrs/command"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "fix-attrs"
	app.Version = "0.3.0"
	app.Usage = "fixes files attributes based on configuration file"
	app.Commands = []cli.Command{
		command.NewFixCommand(),
	}
	app.Run(os.Args)
}
