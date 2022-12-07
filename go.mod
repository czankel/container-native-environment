module github.com/czankel/cne

go 1.13

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/containerd/console v1.0.3
	github.com/containerd/containerd v1.6.12
	github.com/containerd/typeurl v1.0.2
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/google/uuid v1.3.0
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.3-0.20211202183452-c5a74bcca799
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/spf13/cobra v1.3.0
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
