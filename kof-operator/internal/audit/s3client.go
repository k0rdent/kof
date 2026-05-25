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

package audit

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

// s3API is the subset of the AWS S3 client used by S3Client.
// Extracted as an interface to allow unit-testing without a real S3 service.
type s3API interface {
	GetObjectLockConfiguration(ctx context.Context, params *s3.GetObjectLockConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetObjectLockConfigurationOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// S3Client wraps the AWS S3 SDK with the operations required by the exporter.
type S3Client struct {
	client   s3API
	uploader *transfermanager.Client
	bucket   string
}

// NewS3Client creates an S3Client configured for an S3-compatible endpoint.
// When S3AccessKey and S3SecretKey are both set, static credentials are used.
// Otherwise, the default AWS credential chain is used (environment variables,
// shared config file, IRSA, EC2 instance metadata, etc.).
func NewS3Client(cfg *Config) (*S3Client, error) {
	var httpClient aws.HTTPClient
	if cfg.S3ForceHTTP {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // dev only
			},
		}
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.S3Region),
	}
	// Use explicit static credentials only when both keys are provided.
	// Otherwise the default AWS credential chain applies (env vars, ~/.aws,
	// IRSA, EC2 instance metadata, etc.).
	if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" {
		loadOpts = append(loadOpts,
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, ""),
			),
		)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = cfg.S3UsePathStyle
			if httpClient != nil {
				o.HTTPClient = httpClient
			}
			if cfg.S3Endpoint != "" {
				o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			}
		},
	}

	rawClient := s3.NewFromConfig(awsCfg, s3Opts...)
	return &S3Client{
		client:   rawClient,
		uploader: transfermanager.New(rawClient),
		bucket:   cfg.S3Bucket,
	}, nil
}

// PreflightBucket checks the bucket's object-lock status.
// Returns a non-fatal warning string when WORM is absent in non-compliance mode.
// Returns an error when compliance mode requires WORM but it is not present.
func (c *S3Client) PreflightBucket(ctx context.Context, complianceMode bool) (warn string, err error) {
	out, lockErr := c.client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(c.bucket),
	})

	wormEnabled := lockErr == nil &&
		out.ObjectLockConfiguration != nil &&
		out.ObjectLockConfiguration.ObjectLockEnabled == types.ObjectLockEnabledEnabled

	// Spec table:
	// compliance=on  + worm=no  → error; abort; page operator
	// compliance=off + worm=no  → warn; proceed
	// compliance=on  + worm=yes → ok
	// compliance=off + worm=yes → ok
	if complianceMode && !wormEnabled {
		return "", fmt.Errorf(
			"compliance mode is ON but bucket %q does not have WORM/Object-lock enabled: aborting",
			c.bucket,
		)
	}
	if !complianceMode && !wormEnabled {
		return fmt.Sprintf(
			"bucket %q does not have WORM/Object-lock enabled; proceeding without WORM",
			c.bucket,
		), nil
	}
	return "", nil
}

// ObjectExists returns true if the given key already exists in the bucket.
// Used to implement idempotency: a completed window always has a manifest.json.
// Returns (false, nil) for any 404 response. Returns (false, err) for all
// other errors so that callers can distinguish "not found" from "S3 is down".
func (c *S3Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
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

// GetObject downloads an object and returns its raw bytes.
// Returns (nil, nil) when the object does not exist.
func (c *S3Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, nil
		}
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return nil, nil
		}
		// S3-compatible backends may surface a missing key as an untyped HTTP 404.
		var respErr *smithyhttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("GetObject %q: %w", key, err)
	}
	defer func() { _ = out.Body.Close() }()
	return io.ReadAll(out.Body)
}

// PutObject uploads data to the given key with the specified content type.
func (c *S3Client) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
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
func (c *S3Client) UploadStream(ctx context.Context, key string, r io.Reader, contentType string) error {
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
