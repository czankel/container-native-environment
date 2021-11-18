module github.com/czankel/cne

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/containerd/console v1.0.2
	github.com/containerd/containerd v1.5.2
	github.com/containerd/typeurl v1.0.2
	github.com/google/uuid v1.2.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runtime-spec v1.0.3-0.20200929063507-e6143ca7d51d
	github.com/opencontainers/selinux v1.8.2 // indirect
	github.com/spf13/cobra v1.0.0
	golang.org/x/sys v0.0.0-20210324051608-47abb6519492
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
