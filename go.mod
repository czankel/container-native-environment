module github.com/czankel/cne

go 1.13

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/Microsoft/hcsshim v0.9.1 // indirect
	github.com/containerd/cgroups v1.0.2 // indirect
	github.com/containerd/console v1.0.3
	github.com/containerd/containerd v1.5.9
	github.com/containerd/continuity v0.2.1 // indirect
	github.com/containerd/typeurl v1.0.2
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/google/uuid v1.3.0
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/moby/sys/mountinfo v0.5.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runc v1.0.3 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/opencontainers/selinux v1.10.0 // indirect
	github.com/spf13/cobra v1.3.0
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	google.golang.org/genproto v0.0.0-20211223182754-3ac035c7e7cb // indirect
	google.golang.org/grpc v1.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
