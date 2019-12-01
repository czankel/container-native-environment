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
	Name       string
	SocketName string
	Namespace  string `cne:"ReadOnly"`
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
			Name:       DefaultExecRuntimeName,
			SocketName: DefaultExecRuntimeSocketName,
			Namespace:  DefaultExecRuntimeNamespace,
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
// concatenated filter (using '.' as the separator). The filter is case-insensitive.
// This function also returns the actual path using the correctly capitalized letters
func (conf *Config) getValue(filter string) (string, reflect.Value, string) {

	var realPath string
	var tag string

	elem := reflect.ValueOf(conf).Elem()
	path := strings.Split(filter, ".")

	for i, f := range path {
		var fieldName string
		strElem := elem
		elem = elem.FieldByNameFunc(func(fn string) bool {
			if strings.ToLower(f) != strings.ToLower(fn) {
				return false
			}

			realPath = realPath + fn
			return true
		})
		if !elem.IsValid() {
			return realPath, elem, ""
		}
		if i == len(path)-1 && elem.Kind() == reflect.String {
			field, _ := strElem.Type().FieldByName(fieldName)
			tag = field.Tag.Get("cne")
		} else {
			realPath = realPath + "."
		}
	}

	return realPath, elem, tag
}

// Set updates the value of the configuration field
// Returns the actual case-corrected path and value of the field
// Errors:
//  - ErrNoSuchResource if the specified configuration field cannot be found
//  - ErrInvalidArgument if the specified configuration field is a structure
//  - ErrReadOnly if the specified configuration field cannot be written

func (conf *Config) SetByName(name string, value string) (string, error) {

	path, field, tag := conf.getValue(name)
	if !field.IsValid() {
		return path, errdefs.ErrNoSuchResource
	}
	if field.Kind() != reflect.String {
		return "", errdefs.ErrInvalidArgument
	}

	if tag == "ReadOnly" {
		return "", errdefs.ErrReadOnly
	}

	field.SetString(value)
	return path, nil
}

// Get returns the value of the configuration field specified by the filter
// Errors:
//  - ErrNoSuchResource if the specified configuration field cannot be found
func (conf *Config) GetByName(name string) (string, string, error) {

	path, field, _ := conf.getValue(name)
	if !field.IsValid() {
		return "", "", errdefs.ErrNoSuchResource
	}

	return path, field.String(), nil
}

// GetAllByName returns a 'reflect.Value' for the selected field, which
// can be a structure for nested structures.
// Errors:
//  - ErrNoSuchResource if the specified configuration field cannot be found
func (conf *Config) GetAllByName(filter string) (string, interface{}, error) {

	path, field, _ := conf.getValue(filter)
	if !field.IsValid() {
		return "", reflect.Value{}, errdefs.ErrNoSuchResource
	}

	return path, field.Interface(), nil
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
