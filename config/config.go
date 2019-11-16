// Package config manages project-independent configurations.
// Configurations describe the environment for the projects for a user, and system-wide
// stored in /etc/cneconfig
package config

import (
	"bufio"
	"os"
	"os/user"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/czankel/cne/errdefs"
)

type Runtime struct {
	Name   string
	Socket string
}

type Config struct {
	Runtime Runtime
}

// update updates the configuration with the values from the specified configuration file
func (conf *Config) update(path string) error {
	_, err := toml.DecodeFile(path, conf)
	return err
}

// Load returns the default configuration amended by the configuration stored in the
// system and user configuration file.
func Load() *Config {

	conf := &Config{
		Runtime: Runtime{
			Name:   DefaultExecRuntime,
			Socket: DefaultExecRunSock,
		},
	}

	conf.update(SystemConfigFile)

	usr, err := user.Current()
	if err == nil {
		conf.update(usr.HomeDir + "/" + UserConfigFile)
	}

	return conf
}

// LoadSystemConfig loads only the system configuration
func LoadSystemConfig() *Config {

	conf := &Config{}
	conf.update(SystemConfigFile)

	return conf
}

// LoadUserConfig loads only the system configuration
func LoadUserConfig() *Config {

	conf := &Config{}

	usr, err := user.Current()
	if err == nil {
		conf.update(usr.HomeDir + "/" + UserConfigFile)
	}

	return conf
}

// getValue returns the reflect.Value for the element in the nested structure by the
// concatenated name (using '.' as the separator). The name is case-insensitive.
// This function also returns the actual path using the correctly capitalized letters
func (conf *Config) getValue(path string) (string, reflect.Value) {

	var realPath string

	elem := reflect.ValueOf(conf).Elem()
	fields := strings.Split(path, ".")

	for i, f := range fields {
		elem = elem.FieldByNameFunc(func(fn string) bool {
			if len(realPath) != 0 {
				realPath = realPath + "."
			}
			realPath = realPath + fn
			return strings.ToLower(f) == strings.ToLower(fn)
		})
		if !elem.IsValid() {
			return realPath, elem
		}
		if i < len(fields)-1 && elem.Kind() != reflect.Struct {
			return realPath, reflect.Value{}
		}

	}

	return realPath, elem
}

// Set updates the value of the configuration field
func (conf *Config) Set(name string, value string) (string, error) {

	path, field := conf.getValue(name)
	if !field.IsValid() {
		return path, errdefs.ErrNoSuchResource
	}

	field.SetString(value)
	return path, nil
}

// Get returns the value of the configuration field
func (conf *Config) Get(name string) (string, string, error) {

	path, field := conf.getValue(name)
	if !field.IsValid() {
		return "", "", errdefs.ErrNoSuchResource
	}

	return path, field.String(), nil
}

// WriteSystemConfig writes the system configuration to /etc/cneconfig.
// This
func (conf *Config) WriteSystemConfig() error {

	file, err := os.OpenFile(SystemConfigFile, os.O_TRUNC|os.O_RDWR|os.O_CREATE, ConfigFilePerms)
	if err != nil {
		return err
	}
	defer file.Sync()

	writer := bufio.NewWriter(file)
	err = toml.NewEncoder(writer).Encode(conf)
	return err
}

// WriteUserConfig writes the user configuration in the home directory of the current user.
func (conf *Config) WriteUserConfig() error {

	usr, err := user.Current()
	if err != nil {
		return err
	}

	path := usr.HomeDir + "/" + UserConfigFile
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, ConfigFilePerms)
	if err != nil {
		return err
	}
	defer file.Sync()

	euid := os.Geteuid()
	uid := os.Getuid()
	if euid != uid {
		gid := os.Getgid()
		if err = file.Chown(uid, gid); err != nil {
			return err
		}
	}

	writer := bufio.NewWriter(file)
	return toml.NewEncoder(writer).Encode(conf)
}
