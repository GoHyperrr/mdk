# mdk — Hyperrr Module Development Kit

The `mdk` package contains the interfaces and types needed to build a Hyperrr module.
It has minimal dependencies and is the only Hyperrr package third-party modules need to import.

## Install

```bash
go get github.com/GoHyperrr/mdk
```

## Implementing a Module

```go
package mymodule

import (
	"context"
	"github.com/GoHyperrr/mdk"
)

func init() {
	mdk.Register(func() mdk.Module { return &MyModule{} })
}

type MyModule struct{ rt mdk.Runtime }

func (m *MyModule) ID() string       { return "mymodule" }
func (m *MyModule) Models() []any    { return []any{&MyModel{}} }
func (m *MyModule) Routes() []mdk.Route { return nil }

func (m *MyModule) Init(ctx context.Context, rt mdk.Runtime) error {
	m.rt = rt
	return nil
}

func (m *MyModule) Shutdown(ctx context.Context) error { return nil }
```

## Configuration Access

Inside your module's `Init(ctx context.Context, rt mdk.Runtime)` method, you can access environment-expanded configurations defined in `hyperrr.yml` using:
```go
val, ok := rt.Config("my_module_key").(string)
```
This preserves full domain isolation since modules only read config keys from the shared `mdk.Runtime` context.

Then declare it in your `hyperrr.yml` and run `hyperrr build`.
