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

package coldstorage

import (
	s3pkg "github.com/k0rdent/kof/kof-operator/internal/s3"
)

// S3Client is a type alias for s3pkg.Client.
// It provides ObjectExists, PutObject, and UploadStream.
type S3Client = s3pkg.Client

// NewS3Client creates an S3Client configured for an S3-compatible endpoint.
func NewS3Client(cfg *Config) (*S3Client, error) {
	return s3pkg.NewClient(s3pkg.Config{
		Endpoint:     cfg.S3Endpoint,
		Bucket:       cfg.S3Bucket,
		Region:       cfg.S3Region,
		AccessKey:    cfg.S3AccessKey,
		SecretKey:    cfg.S3SecretKey,
		UsePathStyle: cfg.S3UsePathStyle,
		ForceHTTP:    cfg.S3ForceHTTP,
	})
}
