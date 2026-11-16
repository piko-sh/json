# json

A small library that puts JSON encoding and decoding behind a **pluggable provider**, so an
application chooses its JSON engine once and every package that encodes through this one
follows.

Go's `encoding/json` is correct and everywhere, but it is not the fastest engine, and a
large application ends up with JSON calls scattered through many packages and libraries.
Swapping the engine means touching every one of them. `json` gives those packages one
place to call instead. By default every call goes to `encoding/json`. When the
application activates a provider, such as one backed by sonic, every call made through
this package moves to it, including calls from libraries that know nothing about the
provider.

Module `piko.sh/json`. Go 1.27. Apache-2.0. It depends on nothing outside the standard
library.

## Install

```sh
go get piko.sh/json
```

## Usage

Packages call the package-level functions, as they would call `encoding/json`.

```go
package main

import (
	"fmt"

	"piko.sh/json"
)

type order struct {
	ID   int64    `json:"id"`
	Tags []string `json:"tags"`
}

func main() {
	encoded, err := json.Marshal(order{ID: 1, Tags: []string{"gift"}})
	if err != nil {
		panic(err)
	}

	var decoded order
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		panic(err)
	}
	fmt.Println(string(encoded), decoded.ID)
}
```

The application activates a provider once while it starts, before it serves work.

```go
provider.Activate() // every json call from here on uses the provider
```

A library that would rather be handed its encoder takes an `API` instead.

```go
type Store struct {
	codec json.API
}

func NewStore(codec json.API) *Store {
	if pretoucher, ok := codec.(json.Pretoucher); ok {
		_ = pretoucher.Pretouch(reflect.TypeFor[order]())
	}
	return &Store{codec: codec}
}

store := NewStore(json.Freeze(json.Config{}))
```

## What it provides

- **`Marshal`, `Unmarshal`, `MarshalIndent`, `MarshalString`, `UnmarshalString`,
  `ValidString`, `NewEncoder` and `NewDecoder`** call the active provider, which is
  `encoding/json` until one is activated.
- **`Provider`** is what an engine implements. Its `Activate` method replaces the
  package-level functions and the `ConfigStd` and `ConfigDefault` APIs with the engine's
  own. `FreezeImpl` is the hook a provider replaces to build frozen APIs.
- **`API`** is an encoder and decoder value with the same operations, for code that takes
  its encoder as a dependency. **`ConfigStd`** escapes HTML like `encoding/json`;
  **`ConfigDefault`** does not.
- **`Freeze(Config)`** returns an `API` for a `Config` (`CopyString`, `UseInt64`,
  `EscapeHTML`, `SortMapKeys`). It resolves on its first call, so an API frozen before the
  application activates its provider still uses that provider.
- **`Pretouch(reflect.Type)`** compiles a type's codec ahead of its first use, for engines
  that compile codecs. **`Pretoucher`** is the same operation on an `API`, which every
  `API` this package and its providers return implements.
- **`DrainPretouchBuffer`** returns the types pretouched before a provider was active. A
  provider calls it in `Activate` and pretouches each one.
- **`Encoder`** and **`Decoder`** are the streaming interfaces.

## Semantics

- **Until a provider is activated, everything is `encoding/json`.** Output matches it,
  except that `ConfigDefault` does not escape HTML.
- **Frozen APIs bind late.** `Freeze` returns an API that resolves through `FreezeImpl` on
  its first call, then keeps that engine. Freeze APIs in `init` functions or constructors
  freely; they use the provider active when they are first used.
- **Pretouching before activation is not lost.** `Pretouch` records the type until a
  provider that compiles codecs is active, and the provider pretouches every recorded type
  when it activates, so packages may pretouch from `init` in any order. With
  `encoding/json` alone it records the type and does nothing else.
- **`Pretouch` is safe for concurrent use. `Activate` is not.** Activation replaces
  package-level functions, so run it once while the application starts, before other
  goroutines encode or decode.

## Writing a provider

A provider replaces the package-level functions in `Activate` and then pretouches what
was recorded.

```go
func (sonicProvider) Activate() {
	json.Marshal = sonic.Marshal
	json.Unmarshal = sonic.Unmarshal
	// ... the remaining functions, ConfigStd, ConfigDefault and FreezeImpl
	json.Pretouch = sonicPretouch
	for _, t := range json.DrainPretouchBuffer() {
		_ = sonicPretouch(t)
	}
}
```

Piko ships a provider backed by sonic in `piko.sh/piko/wdk/json/json_provider_sonic`.

## Development

```sh
make test        # run the test suite
make test-race   # run it under the race detector
make crossarch   # cross-compile for arm64
```

Run `make help` for the full list of targets.
