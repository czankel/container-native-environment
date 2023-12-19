module github.com/czankel/cne

go 1.13

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/containerd/console v1.0.3
	github.com/containerd/containerd v1.6.26
	github.com/containerd/typeurl v1.0.2
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/google/uuid v1.3.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc2.0.20221005185240-3a7f492d3f1b
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/spf13/cobra v1.3.0
	golang.org/x/sys v0.13.0
	golang.org/x/term v0.13.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
