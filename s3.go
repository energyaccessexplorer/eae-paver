package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
}

func s3config_get(name string) (s3config, error) {
	s3 := s3config{}

	j, err := os.ReadFile(buckets)
	if err != nil {
		return s3, err
	}

	var array []s3config

	if err := json.Unmarshal(j, &array); err != nil {
		return s3, err
	}

	for _, e := range array {
		if e.Name == name {
			s3 = e
			break
		}
	}

	return s3, nil
}

func s3timestamp() string {
	return time.Now().UTC().Format(time.RFC1123Z)
}

func s3sign(secret string, strs ...string) string {
	hash := hmac.New(sha1.New, []byte(secret))
	hash.Write([]byte(strings.Join(strs, "\n")))

	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func s3put(fname filename, s3 s3config) bool {
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

	content, _ := os.ReadFile(fname)
	contenttype := http.DetectContentType(content)

	timestamp := s3timestamp()

	destination := strings.Join([]string{s3.Bucket, s3.Directory, _uuid(fname)}, "/")

	signature := s3sign(
		s3.Secret,
		"PUT",
		contentmd5,
		contenttype,
		timestamp,
		"x-amz-acl:"+s3.ACL,
		"/"+destination,
	)

	endpoint := fmt.Sprintf("https://%s/%s", s3.Provider, destination)

	client := &http.Client{}

	q, err := http.NewRequest("PUT", endpoint, bytes.NewReader(content))
	q.Header.Add("Date", timestamp)
	q.Header.Add("Content-Type", contenttype)
	q.Header.Add("Content-MD5", contentmd5)
	q.Header.Add("X-AMZ-ACL", s3.ACL)
	q.Header.Add("Authorization", fmt.Sprintf("AWS %s:%s", s3.Key, signature))

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
