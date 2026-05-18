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
	"encoding/json"
	"fmt"
	"sort"
)

// marshalCanonical serialises v to canonical JSON: objects with keys sorted
// alphabetically at every level, arrays in original order.
// This satisfies the spec requirement for deterministic serialisation.
func marshalCanonical(v interface{}) ([]byte, error) {
	// Round-trip through JSON to get a plain map[string]interface{} tree.
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal to intermediate JSON: %w", err)
	}
	var tree interface{}
	if err := json.Unmarshal(raw, &tree); err != nil {
		return nil, fmt.Errorf("unmarshal to interface tree: %w", err)
	}
	return marshalSorted(tree)
}

// marshalSorted recursively writes canonical JSON for the given value.
func marshalSorted(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			keyBytes, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			buf.Write(keyBytes)
			buf.WriteByte(':')
			valBytes, err := marshalSorted(val[k])
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", k, err)
			}
			buf.Write(valBytes)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil

	case []interface{}:
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				buf.WriteByte(',')
			}
			itemBytes, err := marshalSorted(item)
			if err != nil {
				return nil, err
			}
			buf.Write(itemBytes)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil

	default:
		return json.Marshal(v)
	}
}
