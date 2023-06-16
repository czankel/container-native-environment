package config

const (
	UserConfigFile    = ".cneconfig"
	SystemConfigFile  = "/etc/cneconfig"
	ProjectConfigFile = "cneconfig"
	ConfigFilePerms   = 0644

	DefaultPackageVersion = "latest"

	DefaultContextName = "default"

	DefaultRuntimeName       = "containerd"
	DefaultRuntimeEngine     = "containerd"
	DefaultRuntimeSocketName = "/run/containerd/containerd.sock"
	DefaultRuntimeNamespace  = "cne"

	DefaultRegistryName     = "docker.io"
	DefaultRegistryDomain   = "docker.io"
	DefaultRegistryRepoName = "library"
)
