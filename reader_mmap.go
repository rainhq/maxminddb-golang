//go:build !appengine && !plan9 && !js && !wasip1 && !wasi
// +build !appengine,!plan9,!js,!wasip1,!wasi

package maxminddb

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
)

// Open takes a string path to a MaxMind DB file and returns a Reader
// structure or an error. The database file is opened using a memory map
// on supported platforms. On platforms without memory map support, such
// as WebAssembly or Google App Engine, the database is loaded into memory.
// Use the Close method on the Reader object to return the resources to the system.
func Open(file fs.File) (*Reader, error) {
	stats, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	fileSize := int(stats.Size())

	var mmap []byte
	var hasMappedFile bool

	// Try to use mmap if the file implements *os.File
	if osFile, ok := file.(*os.File); ok {
		mmap, err = mmap(int(osFile.Fd()), fileSize)
		if err == nil {
			hasMappedFile = true
		}
	}

	// If mmap failed or wasn't possible, read the entire file
	if mmap == nil {
		mmap = make([]byte, fileSize)
		_, err = io.ReadFull(file, mmap)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
	}

	reader, err := FromBytes(mmap)
	if err != nil {
		if hasMappedFile {
			//nolint:errcheck // we prefer to return the original error
			munmap(mmap)
		}
		return nil, fmt.Errorf("failed to create reader from bytes: %w", err)
	}

	reader.hasMappedFile = hasMappedFile
	if hasMappedFile {
		runtime.SetFinalizer(reader, (*Reader).Close)
	}
	return reader, nil
}

// Close returns the resources used by the database to the system.
func (r *Reader) Close() error {
	var err error
	if r.hasMappedFile {
		runtime.SetFinalizer(r, nil)
		r.hasMappedFile = false
		err = munmap(r.buffer)
	}
	r.buffer = nil
	return err
}