module github.com/axisrobo/rheovela-open

go 1.25.0

require github.com/stretchr/testify v1.12.1

require github.com/axisvela/rheovela-open/sdk/go v0.0.0

replace github.com/axisvela/rheovela-open/sdk/go => ./sdk/go

require go.yaml.in/yaml/v3 v3.0.5 // indirect
