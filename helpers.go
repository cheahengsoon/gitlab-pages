package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"gitlab.com/gitlab-org/labkit/errortracking"
)

// Be careful: if you let either of the return values get garbage
// collected by Go they will be closed automatically.
func createSocket(addr string) (net.Listener, *os.File) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("failed to net.Listen: %+v\n", addr)
		cmd := exec.Command("netstat", "-plnut")

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			fmt.Printf("NETSTAT FAILED: %+v\n", err)
		}
		//}()
		time.Sleep(2 * time.Second)

		fatal(err, "could not create socket")
	}

	return l, fileForListener(l)
}

func fileForListener(l net.Listener) *os.File {
	type filer interface {
		File() (*os.File, error)
	}

	f, err := l.(filer).File()
	if err != nil {
		fatal(err, "could not find file for listener")
	}

	return f
}

func capturingFatal(err error, fields ...errortracking.CaptureOption) {
	errortracking.Capture(err, fields...)
	fatal(err, "capturing fatal")
}
