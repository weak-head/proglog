package server

import (
	"fmt"
	"sync"
)

var (
	// ErrOffsetNotFound happens when the specified offset doesn't exist
	ErrOffsetNotFound = fmt.Errorf("offset not fount")
)

// Record is an entry in the commit log
type Record struct {
	Value  []byte `json:"value"`
	Offset uint64 `json:"offset"`
}

// Log represents commit log
type Log struct {
	mu     sync.Mutex
	recods []Record
}

// NewLog creates and initializes a new instance of the commit log
func NewLog() *Log {
	return &Log{}
}

// Append appends a new record to the commit log
func (c *Log) Append(r Record) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	r.Offset = uint64(len(c.recods))
	c.recods = append(c.recods, r)

	return r.Offset, nil
}

// Read reads a record a the specified offset from the commit log
func (c *Log) Read(offset uint64) (Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if offset >= uint64(len(c.recods)) {
		return Record{}, ErrOffsetNotFound
	}

	return c.recods[offset], nil
}
