package log

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	api "github.com/weak-head/proglog/api/v1"
)

func TestLog(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T, log *Log,
	){
		"append and read a record succeeds": testAppendRead,
		"offset out of range error":         testOutOfRangeErr,
		"init with existing segments":       testInitExisting,
	} {
		t.Run(scenario, func(t *testing.T) {
			dir, err := ioutil.TempDir(os.TempDir(), "log_test")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			c := Config{}
			c.Segment.MaxStoreBytes = 32
			log, err := NewLog(dir, c)
			require.NoError(t, err)

			fn(t, log)
		})
	}
}

func testAppendRead(t *testing.T, log *Log) {
	append := &api.Record{
		Value: []byte("Hello World"),
	}

	off, err := log.Append(append)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	read, err := log.Read(off)
	require.NoError(t, err)
	require.Equal(t, append, read)
}

func testOutOfRangeErr(t *testing.T, log *Log) {
	read, err := log.Read(1001)
	require.Nil(t, read)
	apiErr := err.(*api.ErrOffsetOutOfRange)
	require.Equal(t, uint64(1001), apiErr.Offset)
}

func testInitExisting(t *testing.T, log *Log) {
	append := &api.Record{
		Value: []byte("Hello World"),
	}

	for i := 0; i < 3; i++ {
		off, err := log.Append(append)
		require.NoError(t, err)
		require.Equal(t, uint64(i), off)
	}
	require.NoError(t, log.Close())

	n, err := NewLog(log.Dir, log.Config)
	require.NoError(t, err)

	off, err := n.Append(append)
	require.NoError(t, err)
	require.Equal(t, uint64(3), off)
}
