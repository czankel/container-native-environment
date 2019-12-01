package config

const (
	UserConfigFile   = ".cneconfig"
	SystemConfigFile = "/etc/cneconfig"
	ConfigFilePerms  = 0644

	DefaultExecRuntime   = "containerd"
	DefaultExecRunSock   = "/run/containerd/containerd.sock"
	DefaultExecNamespace = "cne"
)
