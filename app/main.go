package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type App struct {
	s3client   *s3.Client
	bucketName string
}

func NewApp(bucketName string) App {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(
		ctxTimeout,
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               "http://localhost:9444/s3",
						Source:            aws.EndpointSourceCustom,
						SigningRegion:     "us-east-1",
						HostnameImmutable: true,
					}, nil
				})),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"AKIAIOSFODNN7EXAMPLE",
			"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"",
		)),
	)
	if err != nil {
		panic(err.Error())
	}

	return App{
		s3client:   s3.NewFromConfig(cfg),
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
				fmt.Println(">>NO SUCH KEY")
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

func main() {
	bucketName := "bazel"
	app := NewApp(bucketName)
	if ok, err := app.CheckBucket(bucketName); !ok {
		fmt.Printf("ERROR\tYou don't have access to the bucket: %v", err)
		return
	}

	routerServeMux := http.NewServeMux()
	routerServeMux.HandleFunc("/", app.All)
	err := http.ListenAndServe(":7777", routerServeMux)

	panic(err)
}
