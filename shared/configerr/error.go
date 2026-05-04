package configerr

import "fmt"

type ConfigError struct {
	Key     string
	Message string
	Err     error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("config error %q: %s (%v)", e.Key, e.Message, e.Err)
	}

	return fmt.Sprintf("config error %q: %s", e.Key, e.Message)
}
