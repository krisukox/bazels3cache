package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/sevlyar/go-daemon"
)

const (
	defaultPort     = 7777
	sigSuccess      = syscall.SIGUSR1
	sigError        = syscall.SIGUSR2
	shutdownUrlTmpl = "http://localhost:%d/shutdown"
	rootPidEnv      = "ROOT_PID"
)

type App struct {
	s3client   *s3.Client
	bucketName string
	infoLog    *log.Logger
	errorLog   *log.Logger
}

func createAndCheckS3Client(bucketName, s3url string) (*s3.Client, error) {
	cfg, err := loadConfig(s3url)
	if err != nil {
		return nil, fmt.Errorf("cannot load config: %v", err)
	}

	s3Client, err := createS3Client(cfg, bucketName)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the bucket: %v", err)
	}
	return s3Client, nil
}

func (a *App) methodNotAllowed(w http.ResponseWriter, method, s3key string) {
	a.infoLog.Printf("%s: %s", method, s3key)
	a.errorLog.Printf("%s not implemented yet", method)
	w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut)
	w.WriteHeader(http.StatusNotFound)
}

func (a *App) all(w http.ResponseWriter, r *http.Request) {
	s3key := r.URL.Path[1:]

	if r.Method == http.MethodGet {
		a.infoLog.Printf("GET: %s", s3key)

		result, err := a.s3client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: aws.String(a.bucketName),
			Key:    aws.String(s3key),
		})

		if err != nil {
			var nsk *types.NoSuchKey
			if errors.As(err, &nsk) {
				a.errorLog.Printf("No such key: %s", s3key)
			} else {
				a.errorLog.Printf("S3 error: %v", err)
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer result.Body.Close()
		buf, err := io.ReadAll(result.Body)
		if err != nil {
			a.errorLog.Printf("Internal Server Error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(buf)
	} else if r.Method == http.MethodPut {
		a.infoLog.Printf("PUT: %s", s3key)

		buf, err := io.ReadAll(r.Body)
		if err != nil {
			a.errorLog.Printf("Internal Server Error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if _, err = a.s3client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(a.bucketName),
			Key:    aws.String(s3key),
			Body:   bytes.NewReader(buf),
		}); err != nil {
			a.errorLog.Printf("Couldn't upload %v : %v", s3key, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else if r.Method == http.MethodHead {
		a.methodNotAllowed(w, http.MethodHead, s3key)
	} else if r.Method == http.MethodDelete {
		a.methodNotAllowed(w, http.MethodDelete, s3key)
	}
}

func (a *App) shutdown(w http.ResponseWriter, r *http.Request) {
	log.Println("Shutting down")
	fmt.Fprintf(w, "Shutting down")
	go os.Exit(0)
}

func sendShutdown(port int) error {
	resp, err := http.Get(fmt.Sprintf(shutdownUrlTmpl, port))
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

func createDaemonContext() *daemon.Context {
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
		Env:         append(os.Environ(), rootPidEnv+"="+strconv.Itoa(os.Getpid())),
		Args:        append(os.Args, "[bazels3cache-daemon]"),
	}
}

func rootProcess(port int, logFile string) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, sigSuccess, sigError)

	switch sig := <-sigs; sig {
	case sigSuccess:
		executablePath, err := os.Executable()
		if err != nil {
			executablePath = ""
		}
		executableName := filepath.Base(executablePath)

		portSwitch := ""
		if port != defaultPort {
			portSwitch = " --port " + strconv.Itoa(port)
		}

		fmt.Printf(
			"Server `%[1]s` is running, to stop it run `%[1]s --stop%[2]s` or `curl %[3]s`\n",
			executableName, portSwitch, fmt.Sprintf(shutdownUrlTmpl, port),
		)
		fmt.Printf("Logging to %s\n", logFile)
		return nil
	case sigError:
		return fmt.Errorf("failed to initialize application")
	}
	return fmt.Errorf("failed to initialize application")
}

func daemonProcess(bucketName, s3url string, port int, infoLog, errorLog *log.Logger) error {
	rootPid, err := strconv.Atoi(os.Getenv(rootPidEnv))
	if err != nil {
		return fmt.Errorf("internal application error: %v", err)
	}

	addr := ":" + strconv.Itoa(port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		sendError(rootPid)
		return fmt.Errorf("can't open port %d: %v", port, err)
	}

	s3Client, err := createAndCheckS3Client(bucketName, s3url)
	if err != nil {
		return err
	}

	app := &App{
		s3client:   s3Client,
		bucketName: bucketName,
		infoLog:    infoLog,
		errorLog:   errorLog,
	}

	routerServeMux := http.NewServeMux()
	routerServeMux.HandleFunc("/", app.all)
	routerServeMux.HandleFunc("/shutdown", app.shutdown)

	server := &http.Server{Addr: addr, Handler: routerServeMux, ErrorLog: errorLog}

	sendSuccess(rootPid)

	return server.Serve(ln)
}

func sendSuccess(rootPid int) {
	syscall.Kill(rootPid, sigSuccess)
}

func sendError(rootPid int) {
	syscall.Kill(rootPid, sigError)
}

func main() {
	bucketName := flag.String("bucket", "", "s3 bucket name")
	s3url := flag.String("s3url", "", "s3 url used for testing")
	port := flag.Int("port", defaultPort, "s3 bucket name")
	stop := flag.Bool("stop", false, "s3 bucket name")
	flag.Parse()

	if *stop {
		if err := sendShutdown(*port); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *bucketName == "" {
		fmt.Printf("Error: Please specify S3 bucket name: --bucket string\n")
		return
	}

	cntxt := createDaemonContext()

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Internal application error: ", err)
	}
	if d != nil {
		if err := rootProcess(*port, cntxt.LogFileName); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	defer cntxt.Release()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ltime)

	if err := daemonProcess(*bucketName, *s3url, *port, infoLog, errorLog); err != nil {
		errorLog.Fatal(err)
	}
}
