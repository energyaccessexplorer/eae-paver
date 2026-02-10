package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"net/http"
	"os"
	"time"
)

type s3config struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	Secret    string `json:"secret"`
	Provider  string `json:"provider"`
	Bucket    string `json:"bucket"`
	Directory string `json:"directory"`
	ACL       string `json:"acl"`
	Region    string `json:"region"`
}

func s3config_get(name string) (_ s3config, err error) {
	j, err := os.ReadFile(buckets)
	if err != nil {
		return
	}

	var array []s3config

	if err = json.Unmarshal(j, &array); err != nil {
		return
	}

	for _, e := range array {
		if e.Name == name {
			return e, nil
		}
	}

	err = errors.New("bucket config not found: " + name)

	return
}

func s3put(fname filename, s3 s3config) bool {
	file, err := os.Open(fname)
	if err != nil {
		logger.Println(err.Error())
		return false
	}

	content, err := os.ReadFile(fname)
	if err != nil {
		return false
	}

	endpoint, err := s3presigned(s3.Name, "PUT", s3.Directory, _uuid(fname))
	if err != nil {
		return false
	}

	client := &http.Client{}

	q, err := http.NewRequest("PUT", endpoint, bytes.NewReader(content))
	if err != nil {
		return false
	}

	r, err := client.Do(q)
	if err != nil {
		return false
	}

	file.Close()

	c, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Println(r.Status, err.Error(), c)
		return false
	}

	if r.StatusCode > 399 {
		logger.Println(r.Status, c)
		return false
	}

	return true
}

func s3presigned(platform string, method string, directory string, filename string) (url string, err error) {
	conf, err := s3config_get(platform)
	if err != nil {
		return
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFn()

	c, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(conf.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				conf.Key,
				conf.Secret,
				"", // session token
			),
		),
	)
	if err != nil {
		return
	}

	cli := s3.NewPresignClient(s3.NewFromConfig(c))

	var request *v4.PresignedHTTPRequest
	if method == "GET" {
		request, err = cli.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(conf.Bucket),
			Key:    aws.String(directory + "/" + filename),
		}, s3.WithPresignExpires(10*time.Minute))
	} else if method == "PUT" {
		request, err = cli.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(conf.Bucket),
			Key:    aws.String(directory + "/" + filename),
		}, s3.WithPresignExpires(10*time.Minute))
	} else {
		err = errors.New("Parameter 'method' is incorrect")
		return
	}

	return request.URL, err
}
