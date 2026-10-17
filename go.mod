module github.com/axisvela/rheovela-open/v3

go 1.25.0

require github.com/stretchr/testify v1.12.1

require github.com/axisvela/rheovela-open/sdk/go/v3 v3.0.0

replace github.com/axisvela/rheovela-open/sdk/go/v3 => ./sdk/go

require go.yaml.in/yaml/v3 v3.0.5 // indirect
