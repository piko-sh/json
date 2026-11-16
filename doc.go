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

// Package json wraps JSON encoding and decoding behind a pluggable [Provider], so an
// application can swap encoding/json for a faster engine in one place and every library
// that encodes through this package follows.
//
// By default every operation delegates to encoding/json. A provider, such as Piko's sonic
// provider (piko.sh/piko/wdk/json/json_provider_sonic), replaces the package-level functions
// ([Marshal], [Unmarshal] and the rest) and the [ConfigStd] and [ConfigDefault] APIs when
// its Activate method runs, which an application does once while it starts.
//
// # Dependency injection
//
// A library that would rather be handed an encoder than reach for package state takes
// an [API]. [Freeze] returns one that resolves lazily on its first call, so an API frozen
// before the application activates its provider still uses that provider. An API also
// implements [Pretoucher], which a library checks for to compile the codecs of the types
// it decodes ahead of their first use.
//
// # Pretouching
//
// [Pretouch] compiles the codec for a type ahead of its first use, for a provider whose
// codecs are compiled, such as sonic's. Before such a provider is active the type is
// recorded, and the provider pretouches every recorded type when it activates, so
// packages may call Pretouch from init functions in any order. Pretouch is safe for
// concurrent use.
//
// Activating a provider is not safe while other goroutines encode or decode, so activate
// it before serving work.
package json
