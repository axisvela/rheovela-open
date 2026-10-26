module github.com/axisvela/rheovela-open/sdk/go/harmovela/v3

go 1.26.0

toolchain go1.26.10

require (
	github.com/axisvela/harmovela/implementations/go/v3 v3.0.0-00010101000000-000000000000
	github.com/axisvela/rheovela-open/sdk/go/v3 v3.1.0
)

replace github.com/axisvela/rheovela-open/sdk/go/v3 => ../

replace github.com/axisvela/harmovela/implementations/go/v3 => ../../../../Harmovela/implementations/go
