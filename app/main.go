package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/sevlyar/go-daemon"
)

type App struct {
	s3client   *s3.Client
	bucketName string
}

func NewApp(bucketName string, s3url string) App {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var loadOptions []func(*config.LoadOptions) error

	if s3url != "" {
		loadOptions = append(loadOptions, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               s3url,
						Source:            aws.EndpointSourceCustom,
						SigningRegion:     "us-east-1",
						HostnameImmutable: true,
					}, nil
				})))
	}

	cfg, err := config.LoadDefaultConfig(ctxTimeout, loadOptions...)
	if err != nil {
		panic(err.Error())
	}

	return App{
		s3client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			cred, err := o.Credentials.Retrieve(context.TODO())
			if err != nil {
				fmt.Printf("Error with retriving credentials")
			} else {
				fmt.Printf("%+v\n", cred)
			}
		}),
		bucketName: bucketName,
	}
}

func (a *App) CheckBucket(bucketName string) (bool, error) {
	if _, err := a.s3client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) All(w http.ResponseWriter, r *http.Request) {
	s3key := r.URL.Path[1:]

	if r.Method == http.MethodGet {
		fmt.Printf("GET: %+v\n", s3key)

		result, err := a.s3client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: aws.String(a.bucketName),
			Key:    aws.String(s3key),
		})

		if err != nil {
			var nsk *types.NoSuchKey
			if errors.As(err, &nsk) {
				fmt.Println("ERRO\tNO SUCH KEY")
			} else {
				fmt.Printf("%+v\n", err)
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer result.Body.Close()
		buf, err := io.ReadAll(result.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(buf)
	} else if r.Method == http.MethodPut {
		fmt.Printf("PUT: %+v\n", s3key)
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = a.s3client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(a.bucketName),
			Key:    aws.String(s3key),
			Body:   bytes.NewReader(buf),
		})
		if err != nil {
			log.Printf("Couldn't upload %v : %v\n", s3key, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else if r.Method == http.MethodHead {
		fmt.Printf("HEAD: %+v\n", s3key)
		panic("HEAD NOT IMPLEMENTED YET")
	} else if r.Method == http.MethodDelete {
		fmt.Printf("DELETE: %+v\n", s3key)
		panic("DELETE NOT IMPLEMENTED YET")
	}
}

func (a *App) Shutdown(w http.ResponseWriter, r *http.Request) {
	log.Println("Shutting down")
	fmt.Fprintf(w, "Shutting down")
	go os.Exit(0)
}

func SendShutdown(shutdownUrl string) error {
	resp, err := http.Get(shutdownUrl)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			fmt.Printf("Server is not running\n")
			return err
		}
	}
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(response))
	return nil
}

func main() {
	const defaultPort = 7777

	s3url := flag.String("s3url", "", "s3 url used for testing")
	bucketName := flag.String("bucket", "", "s3 bucket name")
	port := flag.Int("port", defaultPort, "s3 bucket name")
	stop := flag.Bool("stop", false, "s3 bucket name")
	flag.Parse()

	portString := strconv.Itoa(*port)
	shutdownUrl := fmt.Sprintf("http://localhost:%s/shutdown", portString)

	if *stop {
		if SendShutdown(shutdownUrl) != nil {
			os.Exit(1)
		}
		return
	}

	if *bucketName == "" {
		fmt.Printf("ERROR\tPlease specify S3 bucket name: -bucket string\n")
		return
	}

	usr, err := user.Current()
	workDir := ""
	if err == nil {
		workDir = usr.HomeDir
		fmt.Printf("WORKDIR: %s\n", workDir)
	}

	cntxt := &daemon.Context{
		PidFileName: filepath.Join(workDir, ".bazels3cache.pid"),
		PidFilePerm: 0644,
		LogFileName: filepath.Join(workDir, ".bazels3cache.log"),
		LogFilePerm: 0640,
		Umask:       027,
		Args:        append(os.Args, "[bazels3cache-daemon]"),
	}

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
	}
	if d != nil {
		executablePath, _ := os.Executable()
		portSwitch := ""
		if *port != defaultPort {
			portSwitch = " -port " + portString
		}
		fmt.Printf("Server `%[1]s` is running, to stop it run `curl %[2]s` or `%[1]s -stop%[3]s`\n", filepath.Base(executablePath), shutdownUrl, portSwitch)
		fmt.Printf("Logging to %s/.bazels3cache.log\n", workDir)
		return
	}

	defer cntxt.Release()

	app := NewApp(*bucketName, *s3url)

	if *s3url == "" {
		if ok, err := app.CheckBucket(*bucketName); !ok {
			fmt.Printf("ERROR\tYou don't have access to the bucket: %v\n", err)
			return
		}
	}

	routerServeMux := http.NewServeMux()
	routerServeMux.HandleFunc("/", app.All)
	routerServeMux.HandleFunc("/shutdown", app.Shutdown)
	err = http.ListenAndServe(":"+portString, routerServeMux)

	panic(err)
}
