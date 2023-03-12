//go:build debug
// +build debug

package app

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func loadConfig(s3url string) (aws.Config, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return config.LoadDefaultConfig(
		ctxTimeout,
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               s3url,
						Source:            aws.EndpointSourceCustom,
						SigningRegion:     "localhost",
						HostnameImmutable: true,
					}, nil
				})))
}

func createS3Client(cfg aws.Config, _ string) (*s3.Client, error) {
	return s3.NewFromConfig(cfg), nil
}
