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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	s3pkg "github.com/k0rdent/kof/kof-operator/internal/s3"
)

// S3Client extends s3pkg.Client with audit-specific operations
// (compliance / WORM checking and direct object retrieval).
type S3Client struct {
	*s3pkg.Client
}

// NewS3Client creates an S3Client configured for an S3-compatible endpoint.
// When S3AccessKey and S3SecretKey are both set, static credentials are used.
// Otherwise, the default AWS credential chain is used.
func NewS3Client(cfg *Config) (*S3Client, error) {
	core, err := s3pkg.NewClient(s3pkg.Config{
		Endpoint:     cfg.S3Endpoint,
		Bucket:       cfg.S3Bucket,
		Region:       cfg.S3Region,
		AccessKey:    cfg.S3AccessKey,
		SecretKey:    cfg.S3SecretKey,
		UsePathStyle: cfg.S3UsePathStyle,
		ForceHTTP:    cfg.S3ForceHTTP,
	})
	if err != nil {
		return nil, err
	}
	return &S3Client{Client: core}, nil
}

// PreflightBucket checks the bucket's object-lock status.
// Returns a non-fatal warning string when WORM is absent in non-compliance mode.
// Returns an error when compliance mode requires WORM but it is not present.
func (c *S3Client) PreflightBucket(ctx context.Context, complianceMode bool) (warn string, err error) {
	out, lockErr := c.Raw().GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(c.Bucket()),
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
			c.Bucket(),
		)
	}
	if !complianceMode && !wormEnabled {
		return fmt.Sprintf(
			"bucket %q does not have WORM/Object-lock enabled; proceeding without WORM",
			c.Bucket(),
		), nil
	}
	return "", nil
}

// GetObject downloads an object and returns its raw bytes.
// Returns (nil, nil) when the object does not exist.
func (c *S3Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	out, err := c.Raw().GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket()),
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
