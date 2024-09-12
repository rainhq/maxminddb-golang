//go:build !appengine && !plan9 && !js && !wasip1 && !wasi
// +build !appengine,!plan9,!js,!wasip1,!wasi

package maxminddb

import (
	"fmt"
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
		return nil, err
	}

	fileSize := int(stats.Size())

	// Check if the file implements os.File to get the file descriptor
	osFile, ok := file.(*os.File)
	if !ok {
		return nil, fmt.Errorf("file does not implement *os.File, cannot mmap")
	}

	mmap, err := mmap(int(osFile.Fd()), fileSize)
	if err != nil {
		return nil, err
	}

	reader, err := FromBytes(mmap)
	if err != nil {
		//nolint:errcheck // we prefer to return the original error
		munmap(mmap)
		return nil, err
	}

	reader.hasMappedFile = true
	runtime.SetFinalizer(reader, (*Reader).Close)
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