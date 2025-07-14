package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	S3KEY       string
	S3SECRET    string
	S3PROVIDER  string
	S3BUCKET    string
	S3DIRECTORY string
	S3ACL       string
)

func s3timestamp() string {
	return time.Now().UTC().Format(time.RFC1123Z)
}

func s3sign(strs ...string) string {
	hash := hmac.New(sha1.New, []byte(S3SECRET))
	hash.Write([]byte(strings.Join(strs, "\n")))

	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func s3put(fname filename) bool {
	file, err := os.Open(fname)
	if err != nil {
		logger.Println(err.Error())
		return false
	}

	chksum := md5.New()
	if _, err := io.Copy(chksum, file); err != nil {
		logger.Println(err.Error())
		return false
	}
	contentmd5 := base64.StdEncoding.EncodeToString(chksum.Sum(nil))

	content, _ := ioutil.ReadFile(fname)
	contenttype := http.DetectContentType(content)

	timestamp := s3timestamp()

	destination := strings.Join([]string{S3BUCKET, S3DIRECTORY, _uuid(fname)}, "/")

	signature := s3sign(
		"PUT",
		contentmd5,
		contenttype,
		timestamp,
		"x-amz-acl:"+S3ACL,
		"/"+destination,
	)

	endpoint := fmt.Sprintf("https://%s/%s", S3PROVIDER, destination)

	client := &http.Client{}

	q, err := http.NewRequest("PUT", endpoint, bytes.NewReader(content))
	q.Header.Add("Date", timestamp)
	q.Header.Add("Content-Type", contenttype)
	q.Header.Add("Content-MD5", contentmd5)
	q.Header.Add("X-AMZ-ACL", S3ACL)
	q.Header.Add("Authorization", fmt.Sprintf("AWS %s:%s", S3KEY, signature))

	r, err := client.Do(q)
	if err != nil {
		return false
	}

	file.Close()

	c, err := ioutil.ReadAll(r.Body)
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
