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

package record

import (
	"sync"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

var (
	initOnce        sync.Once
	DefaultRecorder record.EventRecorder
)

// InitFromRecorder initializes the global default recorder. It can only be called once.
// Subsequent calls are considered noops.
func InitFromRecorder(recorder record.EventRecorder) {
	initOnce.Do(func() {
		DefaultRecorder = recorder
	})
}

// Event constructs an event from the given information and puts it in the queue for sending.
func Event(object runtime.Object, annotations map[string]string, reason, message string) {
	DefaultRecorder.AnnotatedEventf(object, annotations, corev1.EventTypeNormal, title(reason), message)
}

// Eventf is just like Event, but with Sprintf for the message field.
func Eventf(object runtime.Object, annotations map[string]string, reason, message string, args ...any) {
	DefaultRecorder.AnnotatedEventf(object, annotations, corev1.EventTypeNormal, title(reason), message, args...)
}

// Warn constructs a warning event from the given information and puts it in the queue for sending.
func Warn(object runtime.Object, annotations map[string]string, reason, message string) {
	DefaultRecorder.AnnotatedEventf(object, annotations, corev1.EventTypeWarning, title(reason), message)
}

// Warnf is just like Warn, but with Sprintf for the message field.
func Warnf(object runtime.Object, annotations map[string]string, reason, message string, args ...any) {
	DefaultRecorder.AnnotatedEventf(object, annotations, corev1.EventTypeWarning, title(reason), message, args...)
}

// title returns a copy of the string source with all Unicode letters that begin words
// mapped to their Unicode title case.
func title(source string) string {
	return cases.Title(language.Und, cases.NoLower).String(source)
}
