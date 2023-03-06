//go:build !debug
// +build !debug

package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func loadConfig(_ string) (aws.Config, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return config.LoadDefaultConfig(ctxTimeout)
}

func createS3Client(cfg aws.Config, bucketName string) (*s3.Client, error) {
	s3Client := s3.NewFromConfig(cfg)

	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, err := s3Client.HeadBucket(ctxTimeout, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}); err != nil {
		return nil, err
	}

	return s3Client, nil
}
