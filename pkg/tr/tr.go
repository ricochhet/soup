package tr

import (
	"sync"
)

type Localization struct {
	Mutex sync.Mutex

	m map[string]map[string]string
	l string
}

func New() *Localization { return &Localization{m: map[string]map[string]string{}} }

func (l *Localization) Locale(name string) bool {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	_, ok := l.m[name]
	if ok {
		l.l = name
	}

	return ok
}

func (l *Localization) GetLocalization(name string) (map[string]string, bool) {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	translation, ok := l.m[name]

	return translation, ok
}

func (l *Localization) SetLocalization(name string, kvps map[string]string) {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	l.m[name] = kvps
}

func (l *Localization) GetAllLocalizations() map[string]map[string]string {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	return l.m
}

func (l *Localization) SetAllLocalizations(localizations map[string]map[string]string) {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	l.m = localizations
}

func (l *Localization) Tr(key string) string {
	s, _ := l.Get(key)
	return s
}

func (l *Localization) Get(key string) (string, bool) { return l.GetFromLocale(l.l, key) }
func (l *Localization) GetFromLocale(name, key string) (string, bool) {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	translation, ok := l.m[name]
	if ok {
		return translation[key], ok
	}

	return "<none>", ok
}

func (l *Localization) Set(key, value string) bool { return l.SetFromLocale(l.l, key, value) }
func (l *Localization) SetFromLocale(name, key, value string) bool {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	_, ok := l.m[name]
	if ok {
		l.m[name][key] = value
	}

	return ok
}
