package config

const (
	UserConfigFile    = ".cneconfig"
	SystemConfigFile  = "/etc/cneconfig"
	ProjectConfigFile = "cneconfig"
	ConfigFilePerms   = 0644

	DefaultPackageVersion = "latest"

	DefaultExecRuntimeName       = "containerd"
	DefaultExecRuntimeSocketName = "/run/containerd/containerd.sock"
	DefaultExecRuntimeNamespace  = "cne"

	DefaultRegistryName     = "docker.io"
	DefaultRegistryDomain   = "docker.io"
	DefaultRegistryRepoName = "library"
)
