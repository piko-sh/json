// Copyright 2026 PolitePixels Limited
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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package json_test

import (
	"bytes"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/json"
)

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	HTML  string `json:"html"`
}

func TestDefaultsFollowEncodingJSON(t *testing.T) {
	value := sample{Name: "a", Count: 2, HTML: "<b>"}

	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"a","count":2,"html":"<b>"}`, string(encoded))

	var decoded sample
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, value, decoded)

	text, err := json.MarshalString(value)
	require.NoError(t, err)
	assert.Equal(t, string(encoded), text)
	assert.True(t, json.ValidString(text))
	assert.False(t, json.ValidString("{"))
}

func TestConfigStdEscapesHTMLAndConfigDefaultDoesNot(t *testing.T) {
	escaped, err := json.ConfigStd.Marshal(sample{HTML: "<b>"})
	require.NoError(t, err)
	assert.Contains(t, string(escaped), `\u003cb\u003e`)

	plain, err := json.ConfigDefault.Marshal(sample{HTML: "<b>"})
	require.NoError(t, err)
	assert.Contains(t, string(plain), `<b>`)
	assert.NotContains(t, string(plain), "\n")
}

func TestEncoderAndDecoderStream(t *testing.T) {
	var buffer bytes.Buffer
	require.NoError(t, json.NewEncoder(&buffer).Encode(sample{Name: "x"}))

	var decoded sample
	require.NoError(t, json.NewDecoder(&buffer).Decode(&decoded))
	assert.Equal(t, "x", decoded.Name)
}

func TestEveryAPIIsAPretoucher(t *testing.T) {
	for name, api := range map[string]json.API{
		"frozen":  json.Freeze(json.Config{}),
		"std":     json.ConfigStd,
		"default": json.ConfigDefault,
	} {
		_, ok := api.(json.Pretoucher)
		assert.True(t, ok, name)
	}
}

// recordingProvider stands in for a provider that compiles codecs, recording what it is
// asked to pretouch and what it decodes.
type recordingProvider struct {
	pretouched []reflect.Type
	frozen     int
}

func (provider *recordingProvider) Activate() {
	json.Pretouch = func(t reflect.Type) error {
		provider.pretouched = append(provider.pretouched, t)
		return nil
	}
	json.FreezeImpl = func(config json.Config) json.API {
		provider.frozen++
		return json.ConfigStd
	}
	provider.pretouched = append(provider.pretouched, json.DrainPretouchBuffer()...)
}

func TestAProviderActivatedLaterServesEarlierFreezesAndPretouches(t *testing.T) {
	previousPretouch, previousFreeze := json.Pretouch, json.FreezeImpl
	t.Cleanup(func() { json.Pretouch, json.FreezeImpl = previousPretouch, previousFreeze })
	_ = json.DrainPretouchBuffer()

	frozen := json.Freeze(json.Config{EscapeHTML: true})
	pretoucher, ok := frozen.(json.Pretoucher)
	require.True(t, ok)
	require.NoError(t, pretoucher.Pretouch(reflect.TypeFor[sample]()))

	provider := &recordingProvider{}
	provider.Activate()

	assert.Equal(t, []reflect.Type{reflect.TypeFor[sample]()}, provider.pretouched, "the buffered type is pretouched on activation")
	_, err := frozen.Marshal(sample{})
	require.NoError(t, err)
	assert.Equal(t, 1, provider.frozen, "the frozen API resolves through the provider active at first use")

	require.NoError(t, pretoucher.Pretouch(reflect.TypeFor[[]sample]()))
	assert.Len(t, provider.pretouched, 2, "later pretouches go straight to the provider")
	assert.Empty(t, json.DrainPretouchBuffer())
}

func TestPretouchIsSafeFromManyGoroutines(t *testing.T) {
	_ = json.DrainPretouchBuffer()
	var group sync.WaitGroup
	for range 16 {
		group.Go(func() {
			for range 50 {
				_ = json.Pretouch(reflect.TypeFor[sample]())
			}
		})
	}
	group.Wait()

	assert.Len(t, json.DrainPretouchBuffer(), 16*50)
}
