package virtual_ext

import "time"

type Event struct {
	ID        string
	Timestamp time.Time
	Data      []byte
}
