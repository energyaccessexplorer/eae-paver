package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/satori/go.uuid"
	"io"
	"log"
	"os"
	"regexp"
	"syscall"
)

var UUID_REGEXP = regexp.MustCompile("[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}")

var (
	run_server = true
	pubkeyfile string
	tmpdir     string
	socket     string
	buckets    string
)

var (
	logfile     *os.File
	logfilename string
	logger      *log.Logger
)

type filename = string

func main() {
	parse_flags()

	logger_setup()

	serve()
}

func parse_flags() {
	flag.StringVar(&pubkeyfile, "pubkey", "", "Public key file to check JWTs")
	flag.StringVar(&socket, "socket", "/tmp/paver-server.sock", "Socket file to run on")
	flag.StringVar(&logfilename, "log", "/tmp/paver.log", "")
	flag.StringVar(&tmpdir, "tmpdir", "/tmp", "")
	flag.StringVar(&buckets, "buckets", "/etc/paver-buckets.json", "")

	flag.Parse()
}

func logger_setup() {
	var err error

	logfile, err = os.OpenFile(logfilename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}

	var logbuf bytes.Buffer
	logger = log.New(&logbuf, "", log.LstdFlags)
	logger.SetOutput(logfile)

	println("Logging to:", logfilename)
}

func _filename() filename {
	return tmpdir + "/" + uuid.NewV4().String()
}

func _uuid(s string) string {
	return fmt.Sprintf("%s", UUID_REGEXP.Find([]byte(s)))
}

func trash(files ...filename) {
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			logger.Println(err.Error())
		}
	}
}

func capture() func() string {
	r, w, _ := os.Pipe()

	ostderr, _ := syscall.Dup(syscall.Stderr)
	syscall.Dup2(int(w.Fd()), syscall.Stderr)

	return func() string {
		w.Close()
		syscall.Close(syscall.Stderr)

		var b bytes.Buffer
		io.Copy(&b, r)
		syscall.Dup2(ostderr, syscall.Stderr)

		return b.String()
	}
}
