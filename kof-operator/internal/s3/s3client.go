// Copyright 2025
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s3

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// RawAPI is the subset of the AWS S3 client used by Client.
// Extracted as an interface to allow unit-testing without a real S3 service.
// It is the superset of the methods needed by both the audit and cold-storage
// exporters; callers that only use a subset may leave unneeded methods unimplemented
// in their test stubs (panicking on unexpected calls is acceptable for stubs).
type RawAPI interface {
	GetObjectLockConfiguration(ctx context.Context, params *s3.GetObjectLockConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetObjectLockConfigurationOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// Config holds the S3-compatible storage configuration shared by all exporters.
type Config struct {
	Endpoint     string // base URL of the S3-compatible endpoint (empty → AWS default)
	Bucket       string
	Region       string // default: us-east-1
	AccessKey    string // optional; uses default AWS credential chain when empty
	SecretKey    string // optional; must be provided together with AccessKey
	UsePathStyle bool   // set true for MinIO / other path-style endpoints
	ForceHTTP    bool   // skip TLS verification — dev only
}

// Client wraps the AWS S3 SDK with the operations required by the exporters.
type Client struct {
	rawClient RawAPI
	uploader  *transfermanager.Client
	bucket    string
}

// NewClient creates a Client from a Config.
// When AccessKey and SecretKey are both set, static credentials are used.
// Otherwise the default AWS credential chain applies (env vars, ~/.aws,
// IRSA, EC2 instance metadata, etc.).
func NewClient(cfg Config) (*Client, error) {
	var httpClient aws.HTTPClient
	if cfg.ForceHTTP {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // dev only
			},
		}
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		loadOpts = append(loadOpts,
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
			),
		)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = cfg.UsePathStyle
			if httpClient != nil {
				o.HTTPClient = httpClient
			}
			if cfg.Endpoint != "" {
				o.BaseEndpoint = aws.String(cfg.Endpoint)
			}
		},
	}

	rawClient := s3.NewFromConfig(awsCfg, s3Opts...)
	return &Client{
		rawClient: rawClient,
		uploader:  transfermanager.New(rawClient),
		bucket:    cfg.Bucket,
	}, nil
}

// NewClientFromRaw constructs a Client from a pre-built raw client and a
// bucket name. Intended for unit tests that inject a stub implementation.
func NewClientFromRaw(rawClient RawAPI, bucket string) *Client {
	return &Client{rawClient: rawClient, bucket: bucket}
}

// Raw returns the underlying RawAPI client. Callers that need access to
// methods beyond the common set (e.g. GetObjectLockConfiguration for
// compliance checks) may use this accessor.
func (c *Client) Raw() RawAPI { return c.rawClient }

// Bucket returns the configured bucket name.
func (c *Client) Bucket() string { return c.bucket }

// ObjectExists returns true if the given key already exists in the bucket.
// Returns (false, nil) for any 404 response. Returns (false, err) for all
// other errors so that callers can distinguish "not found" from "S3 is down".
func (c *Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := c.rawClient.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	var nsk *types.NoSuchKey
	var notFound *types.NotFound
	if errors.As(err, &nsk) || errors.As(err, &notFound) {
		return false, nil
	}
	// HeadObject returns an untyped HTTP response error for 404s on some
	// S3-compatible backends. Treat any HTTP 404 as "not found".
	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("HeadObject %q: %w", key, err)
}

// PutObject uploads data to the given key with the specified content type.
func (c *Client) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := c.rawClient.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("PutObject %q: %w", key, err)
	}
	return nil
}

// UploadStream streams r directly to S3 without buffering the full content in
// memory. It uses multipart upload under the hood so that content length does
// not need to be known in advance.
func (c *Client) UploadStream(ctx context.Context, key string, r io.Reader, contentType string) error {
	_, err := c.uploader.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("UploadStream %q: %w", key, err)
	}
	return nil
}
