package fileadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	eventslog "github.com/yunerou/niarb/pkg/events-log"
)

type fileAdapter struct {
	mu   sync.Mutex
	file *os.File
}

func New(filePath string) eventslog.Adapter {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("events-log fileadapter: failed to open file %s: %v", filePath, err))
	}
	return &fileAdapter{
		file: f,
	}
}

func (fa *fileAdapter) Log(_ context.Context, entry eventslog.EventEntry) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	fa.mu.Lock()
	defer fa.mu.Unlock()

	_, err = fa.file.Write(line)
	return err
}

func (fa *fileAdapter) Flush() {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	_ = fa.file.Sync()
	_ = fa.file.Close()
}
