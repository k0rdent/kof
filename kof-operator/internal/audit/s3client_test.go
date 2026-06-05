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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	s3pkg "github.com/k0rdent/kof/kof-operator/internal/s3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAudit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Audit Suite")
}

// stubS3 implements s3pkg.RawAPI with configurable responses for
// GetObjectLockConfiguration and HeadObject. All other methods panic — they
// are not exercised by the tests that use this stub.
type stubS3 struct {
	lockOut      *s3.GetObjectLockConfigurationOutput
	lockErr      error
	existingKeys map[string]bool // keys that HeadObject should report as present
	headErr      error           // when set, HeadObject returns this error for every key
}

func (s *stubS3) GetObjectLockConfiguration(_ context.Context, _ *s3.GetObjectLockConfigurationInput, _ ...func(*s3.Options)) (*s3.GetObjectLockConfigurationOutput, error) {
	return s.lockOut, s.lockErr
}
func (s *stubS3) HeadObject(_ context.Context, params *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if s.headErr != nil {
		return nil, s.headErr
	}
	if s.existingKeys[aws.ToString(params.Key)] {
		return &s3.HeadObjectOutput{}, nil
	}
	return nil, &types.NotFound{}
}
func (s *stubS3) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	panic("not implemented")
}
func (s *stubS3) PutObject(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	panic("not implemented")
}

// Compile-time checks.
var _ s3pkg.RawAPI = (*stubS3)(nil)
var _ s3pkg.RawAPI = (*s3.Client)(nil)
var _ = aws.String // import sanity

func wormEnabledStub() *stubS3 {
	return &stubS3{
		lockOut: &s3.GetObjectLockConfigurationOutput{
			ObjectLockConfiguration: &types.ObjectLockConfiguration{
				ObjectLockEnabled: types.ObjectLockEnabledEnabled,
			},
		},
	}
}

func wormDisabledStub() *stubS3 {
	return &stubS3{lockErr: errors.New("ObjectLockConfigurationNotFoundError")}
}

func newTestClient(stub s3pkg.RawAPI) *S3Client {
	return &S3Client{Client: s3pkg.NewClientFromRaw(stub, "test-bucket")}
}

var _ = Describe("S3Client.PreflightBucket", func() {
	const bucket = "test-bucket"

	DescribeTable("WORM enabled",
		func(complianceMode bool) {
			c := newTestClient(wormEnabledStub())
			warn, err := c.PreflightBucket(context.Background(), complianceMode)
			Expect(err).NotTo(HaveOccurred())
			Expect(warn).To(BeEmpty())
		},
		Entry("compliance mode on", true),
		Entry("compliance mode off", false),
	)

	DescribeTable("WORM disabled",
		func(complianceMode bool, expectErr bool) {
			c := newTestClient(wormDisabledStub())
			warn, err := c.PreflightBucket(context.Background(), complianceMode)
			if expectErr {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("compliance mode is ON"))
				Expect(err.Error()).To(ContainSubstring(bucket))
				Expect(warn).To(BeEmpty())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(warn).To(ContainSubstring("does not have WORM"))
				Expect(warn).To(ContainSubstring(bucket))
			}
		},
		Entry("compliance mode on → error", true, true),
		Entry("compliance mode off → warn and proceed", false, false),
	)

	DescribeTable("nil ObjectLockConfiguration treated as no WORM",
		func(complianceMode bool, expectErr bool) {
			stub := &stubS3{lockOut: &s3.GetObjectLockConfigurationOutput{
				ObjectLockConfiguration: nil,
			}}
			c := &S3Client{Client: s3pkg.NewClientFromRaw(stub, bucket)}
			_, err := c.PreflightBucket(context.Background(), complianceMode)
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("compliance mode on → error", true, true),
		Entry("compliance mode off → no error", false, false),
	)

	DescribeTable("ObjectLockEnabled field not set to Enabled treated as no WORM",
		func(complianceMode bool, expectErr bool) {
			stub := &stubS3{lockOut: &s3.GetObjectLockConfigurationOutput{
				ObjectLockConfiguration: &types.ObjectLockConfiguration{
					ObjectLockEnabled: types.ObjectLockEnabled(""),
				},
			}}
			c := &S3Client{Client: s3pkg.NewClientFromRaw(stub, bucket)}
			_, err := c.PreflightBucket(context.Background(), complianceMode)
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("compliance mode on → error", true, true),
		Entry("compliance mode off → no error", false, false),
	)

	DescribeTable("API error (e.g. context cancelled) treated as no WORM",
		func(complianceMode bool, expectErr bool) {
			stub := &stubS3{lockErr: context.Canceled}
			c := newTestClient(stub)
			warn, err := c.PreflightBucket(context.Background(), complianceMode)
			if expectErr {
				Expect(err).To(HaveOccurred())
				Expect(warn).To(BeEmpty())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(warn).To(ContainSubstring("does not have WORM"))
			}
		},
		Entry("compliance mode on → error", true, true),
		Entry("compliance mode off → warn", false, false),
	)
})
