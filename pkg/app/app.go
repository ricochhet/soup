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
	app.logFn(empty(app.OnStart))
	app.flagSet.PrintDefaults()
}

func (app *App) Init(logFn func(format string, a ...any), arguments []string) {
	app.logFn = logFn
	app.flagSet = flag.NewFlagSet("app", flag.ExitOnError)

	app.flagSet.BoolVar(&app.versionFlag, "version", false, "show version")
	app.flagSet.BoolVar(&app.versionFlag, "v", false, "show version")
	app.flagSet.Usage = app.Usage
	app.flagSet.Parse(arguments)

	if app.versionFlag {
		app.logFn("%s-%s\n", empty(app.Name), empty(app.GitHash))
		app.logFn("Build date: %s\n", empty(app.BuildDate))
		app.logFn("Build on: %s\n", empty(app.BuildOn))
		os.Exit(0)
	}
}

func empty(str string) string {
	if str == "" {
		return "<none>"
	}

	return str
}
