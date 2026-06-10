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
	app.logFn(emptyStr(app.OnStart))
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
		app.logFn("%s-%s\n", emptyStr(app.Name), emptyStr(app.GitHash))
		app.logFn("Build date: %s\n", emptyStr(app.BuildDate))
		app.logFn("Build on: %s\n", emptyStr(app.BuildOn))
		os.Exit(0)
	}
}

func emptyStr(str string) string {
	if str == "" {
		return "<none>"
	}

	return str
}
