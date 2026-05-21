package time

import "time"

func Timestamp() string { return time.Now().Format("2006-01-02_15-04-05") }
func Timer(start func() error, end func(time.Duration)) error {
	now := time.Now()
	err := start()

	end(time.Since(now))

	return err
}
