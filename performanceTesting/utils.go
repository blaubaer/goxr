package main

import (
	"fmt"
	"github.com/c2h5oh/datasize"
	"github.com/echocat/goxr/common"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

var (
	executable = func() string {
		result, err := os.Executable()
		must(err)
		return result
	}()
	rootDirectory, filesDirectory = func(executable string) (rootDirectory string, filesDirectory string) {
		rootDirectory = filepath.Dir(executable)
		filesDirectory = filepath.Join(rootDirectory, "files")
		if isDirectory(filesDirectory) {
			return
		}
		_, sourceFile, _, _ := runtime.Caller(0)
		rootDirectory = filepath.Dir(sourceFile)
		filesDirectory = filepath.Join(rootDirectory, "files")
		if isDirectory(filesDirectory) {
			return
		}
		panic(fmt.Sprintf("files directory neither exists in %s nor %s", filepath.Dir(executable), filepath.Dir(sourceFile)))
	}(executable)
)

func generateFile(target string, size datasize.ByteSize) {
	mkdirAll(filepath.Dir(target), 0755)
	f := open(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	defer close(f)

	rng := rand.New(rand.NewSource(666))
	buf := make([]byte, 4096)
	bufSize := datasize.ByteSize(len(buf))
	var written datasize.ByteSize
	for written < size {
		target := size - written
		if target > bufSize {
			target = bufSize
		}
		common.MustRead(rng, buf[:target])
		if n, err := f.Write(buf[:target]); err != nil {
			panic(err)
		} else {
			written += datasize.ByteSize(n)
		}
	}
}

func mkdirAll(name string, mode os.FileMode) {
	must(os.MkdirAll(name, mode))
}

func open(name string, flag int, perm os.FileMode) *os.File {
	f, err := os.OpenFile(name, flag, perm)
	must(err)
	return f
}

func remove(name string) {
	must(os.RemoveAll(name))
}

func fileInfo(name string) os.FileInfo {
	fi, err := os.Stat(name)
	must(err)
	return fi
}

func exists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	must(err)
	return true
}

func isDirectory(name string) bool {
	fi, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	must(err)
	return fi.IsDir()
}

//noinspection GoReservedWordUsedAsName
func close(closer io.Closer) {
	if err := closer.Close(); err != nil && common.UnderlyingError(err) != os.ErrClosed {
		panic(err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func isTemporary(err error) bool {
	if tErr, ok := err.(temporary); ok && tErr.Temporary() {
		return true
	}
	return isTemporaryX(err)
}

type temporary interface {
	Temporary() bool
}

type closeIdleTransport interface {
	CloseIdleConnections()
}

func closeIdleConnections(of http.RoundTripper) {
	if cit, ok := of.(closeIdleTransport); ok {
		cit.CloseIdleConnections()
	}
}
