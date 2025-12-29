package game

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// --------------------------
// WAL Event
// --------------------------

type WALEvent struct {
	Seq       uint64 `json:"seq"`
	MoveUCI   string `json:"move_uci"`
	ServerNs  int64  `json:"server_ns"`
	LagCompNs int64  `json:"lag_comp_ns"`
	WRem      int64  `json:"w_rem"`
	BRem      int64  `json:"b_rem"`
}

// --------------------------
// WAL
// --------------------------

type WAL struct {
	mu       sync.Mutex
	events   []WALEvent
	file     *os.File
	writer   *bufio.Writer
	filePath string
}

// --------------------------
// Create WAL: always in-memory + file
// --------------------------

func NewWAL(filePath string) (*WAL, error) {
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	return &WAL{
		events:   []WALEvent{},
		file:     f,
		writer:   bufio.NewWriter(f),
		filePath: filePath,
	}, nil
}

// --------------------------
// Append an event (both memory & file)
// --------------------------

func (w *WAL) Append(e WALEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// In-memory
	w.events = append(w.events, e)

	// File
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := w.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	return w.writer.Flush()
}

// --------------------------
// Load all events from file
// --------------------------

func (w *WAL) LoadFromFile() ([]WALEvent, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	events := []WALEvent{}
	f, err := os.Open(w.filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e WALEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue // skip invalid lines
		}
		events = append(events, e)
	}

	return events, scanner.Err()
}

// --------------------------
// Load all events from memory
// --------------------------

func (w *WAL) LoadFromMemory() []WALEvent {
	w.mu.Lock()
	defer w.mu.Unlock()

	eventsCopy := make([]WALEvent, len(w.events))
	copy(eventsCopy, w.events)
	return eventsCopy
}

// --------------------------
// Close WAL
// --------------------------

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writer != nil {
		w.writer.Flush()
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// --------------------------
// Helper: Create a WALEvent
// --------------------------

func NewWALEvent(seq uint64, moveUCI string, wRem, bRem int64, lagComp int64) WALEvent {
	return WALEvent{
		Seq:       seq,
		MoveUCI:   moveUCI,
		ServerNs:  time.Now().UnixNano(),
		LagCompNs: lagComp,
		WRem:      wRem,
		BRem:      bRem,
	}
}
