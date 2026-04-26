package clip

import (
	"fmt"
	"time"
)

const ClearDelay = 30 * time.Second

// Write copies text to the system clipboard and schedules a clear after
// ClearDelay. The clear runs in a background goroutine; only the original
// value is erased — if something else was copied in the meantime it is left
// untouched.
func Write(text string) error {
	if err := writeAll(text); err != nil {
		return fmt.Errorf("clipboard write: %w", err)
	}
	go func() {
		time.Sleep(ClearDelay)
		current, err := readAll()
		if err == nil && current == text {
			_ = writeAll("")
		}
	}()
	return nil
}
