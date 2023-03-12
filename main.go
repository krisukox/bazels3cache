package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/krisukox/bazels3cache/app"
	"github.com/sevlyar/go-daemon"
)

func sendShutdownToDaemon(port int) error {
	resp, err := http.Get(fmt.Sprintf(app.ShutdownUrlTmpl, port))
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			return fmt.Errorf("server is not running")
		}
		return fmt.Errorf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("internal application error: %v", err)
	}
	fmt.Println(string(response))

	return nil
}

func createDaemonContext(rootPort int) *daemon.Context {
	usr, err := user.Current()
	workDir := ""
	if err == nil {
		workDir = usr.HomeDir
	}

	return &daemon.Context{
		PidFileName: filepath.Join(workDir, ".bazels3cache.pid"),
		PidFilePerm: 0644,
		LogFileName: filepath.Join(workDir, ".bazels3cache.log"),
		LogFilePerm: 0640,
		Umask:       027,
		Env:         append(os.Environ(), app.RootPortEnv+"="+strconv.Itoa(rootPort)),
		Args:        append(os.Args, "[bazels3cache-daemon]"),
	}
}

func rootProcess(port int, logFile string, ln net.Listener) error {
	defer ln.Close()
	conn, err := ln.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	buf, err := io.ReadAll(conn)
	if err != nil {
		return err
	}
	if string(buf) == app.SuccessMsg {
		executablePath, err := os.Executable()
		if err != nil {
			executablePath = ""
		}
		executableName := filepath.Base(executablePath)

		portSwitch := ""
		if port != app.DefaultPort {
			portSwitch = " --port " + strconv.Itoa(port)
		}

		fmt.Printf(
			"Server `%[1]s` is running, to stop it run `%[1]s --stop%[2]s` or `curl %[3]s`\n",
			executableName, portSwitch, fmt.Sprintf(app.ShutdownUrlTmpl, port),
		)
		fmt.Printf("Logging to %s\n", logFile)
		return nil
	}
	return fmt.Errorf("failed to initialize application: %s", string(buf))
}

func listenOnAny(avoidPort int) (net.Listener, int, error) {
	if !daemon.WasReborn() {
		for port := 2000; port <= 65535; port++ {
			if port == avoidPort {
				continue
			}
			ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
			if err == nil {
				return ln, port, nil
			}
		}
		return nil, 0, fmt.Errorf("could not find an available port")
	}
	return nil, 0, nil
}

func main() {
	bucketName := flag.String("bucket", "", "S3 bucket name")
	daemonPort := flag.Int("port", app.DefaultPort, "Server HTTP port number")
	stop := flag.Bool("stop", false, "Stop application")
	s3url := flag.String("s3url", "", "S3 url used for testing")
	flag.Parse()

	if *stop {
		if err := sendShutdownToDaemon(*daemonPort); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *bucketName == "" {
		fmt.Printf("Error: Please specify S3 bucket name: --bucket string\n")
		return
	}

	ln, rootPort, err := listenOnAny(*daemonPort)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	cntxt := createDaemonContext(rootPort)

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Internal application error: ", err)
	}
	if d != nil {
		if err := rootProcess(*daemonPort, cntxt.LogFileName, ln); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	defer cntxt.Release()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ltime)

	if err := app.DaemonProcess(*bucketName, *s3url, *daemonPort, infoLog, errorLog); err != nil {
		errorLog.Fatal(err)
	}
}
