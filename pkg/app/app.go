package app

import (
	"flag"
	"os"
)

type App struct {
	Name      string
	BuildDate string
	GitHash   string
	BuildOn   string
	OnStart   string

	versionFlag bool
	flagSet     *flag.FlagSet
	logFn       func(format string, a ...any)
}

func New(name, buildDate, gitHash, buildOn, onStart string) *App {
	return &App{
		Name:      name,
		BuildDate: buildDate,
		GitHash:   gitHash,
		BuildOn:   buildOn,
		OnStart:   onStart,
	}
}

func (app *App) Usage() {
	app.logFn(app.OnStart)
	app.flagSet.PrintDefaults()
}

func (app *App) Init(logFn func(format string, a ...any), arguments []string) error {
	app.logFn = logFn
	app.flagSet = flag.NewFlagSet("app", flag.ExitOnError)

	app.flagSet.BoolVar(&app.versionFlag, "version", false, "show version")
	app.flagSet.BoolVar(&app.versionFlag, "v", false, "show version")
	app.flagSet.Usage = app.Usage

	if err := app.flagSet.Parse(arguments); err != nil {
		return err
	}

	if app.versionFlag {
		app.logFn("%s-%s\n", app.Name, app.GitHash)
		app.logFn("Build date: %s\n", app.BuildDate)
		app.logFn("Build on: %s\n", app.BuildOn)
		os.Exit(0)
	}

	return nil
}
