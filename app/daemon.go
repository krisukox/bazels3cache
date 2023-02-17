package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Daemon struct {
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

func (a *Daemon) methodNotAllowed(w http.ResponseWriter, method, s3key string) {
	a.infoLog.Printf("%s: %s", method, s3key)
	a.errorLog.Printf("%s not implemented yet", method)
	w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut)
	w.WriteHeader(http.StatusNotFound)
}

func (a *Daemon) all(w http.ResponseWriter, r *http.Request) {
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

func (a *Daemon) shutdown(w http.ResponseWriter, r *http.Request) {
	a.infoLog.Printf("Shutting down")
	fmt.Fprintf(w, "Shutting down")
	go os.Exit(0)
}

func daemonProcess(bucketName, s3url string, port int, infoLog, errorLog *log.Logger) error {
	rootPort, err := strconv.Atoi(os.Getenv(rootPortEnv))
	if err != nil {
		return fmt.Errorf("internal application error: %v", err)
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		sendError(rootPort, err)
		return fmt.Errorf("can't open port %d: %v", port, err)
	}
	defer ln.Close()

	s3Client, err := createAndCheckS3Client(bucketName, s3url)
	if err != nil {
		sendError(rootPort, err)
		return err
	}

	app := &Daemon{
		s3client:   s3Client,
		bucketName: bucketName,
		infoLog:    infoLog,
		errorLog:   errorLog,
	}

	routerServeMux := http.NewServeMux()
	routerServeMux.HandleFunc("/", app.all)
	routerServeMux.HandleFunc("/shutdown", app.shutdown)

	server := &http.Server{Handler: routerServeMux, ErrorLog: errorLog}

	if err = sendSuccess(rootPort); err != nil {
		return err
	}

	return server.Serve(ln)
}

func killRoot(rootPid int) {
	syscall.Kill(rootPid, syscall.SIGKILL)
}

func sendMsg(rootPort int, msg string) error {
	conn, err := net.Dial("tcp", ":"+strconv.Itoa(rootPort))
	if err != nil {
		return err
	}
	conn.Write([]byte(msg))
	conn.Close()
	return nil
}

func sendSuccess(rootPort int) error {
	return sendMsg(rootPort, successMsg)
}

func sendError(rootPort int, err error) error {
	return sendMsg(rootPort, err.Error())
}
