module github.com/czankel/cne

go 1.13

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/containerd/console v1.0.3
	github.com/containerd/containerd v1.6.18
	github.com/containerd/typeurl v1.0.2
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/google/uuid v1.3.0
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc3
	github.com/opencontainers/runtime-spec v1.1.0-rc.2
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	golang.org/x/sys v0.5.0
	golang.org/x/term v0.5.0
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
