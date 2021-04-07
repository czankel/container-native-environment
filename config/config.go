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

// CneVersion is set in Makefile by a linker option to the git hash/version
var CneVersion string

type Runtime struct {
	Name       string `toml:"Name,omitempty"`
	SocketName string `toml:"SocketName,omitempty"`
	Namespace  string `cne:"ReadOnly" toml:"Namespace,omitempty"`
}

type Registry struct {
	Domain   string
	RepoName string
}

type Config struct {
	Runtime  Runtime `toml:"Runtime,omitempty"`
	Registry map[string]*Registry
}

// update updates the configuration with the values from the specified configuration file
func (conf *Config) update(path string) error {
	_, err := toml.DecodeFile(path, conf)
	if err != nil && !os.IsNotExist(err) {
		return errdefs.InvalidArgument("config file '%s' corrupt", path)
	}
	return nil
}

// Load returns the default configuration amended by the configuration stored in the
// system and user configuration file.
func Load() (*Config, error) {

	conf := &Config{
		Runtime: Runtime{
			Name:       DefaultExecRuntimeName,
			SocketName: DefaultExecRuntimeSocketName,
			Namespace:  DefaultExecRuntimeNamespace,
		},
		Registry: map[string]*Registry{
			DefaultRegistryName: &Registry{
				Domain:   DefaultRegistryDomain,
				RepoName: DefaultRegistryRepoName,
			},
		},
	}

	conf.update(SystemConfigFile)

	usr, err := user.Current()
	if err == nil {
		err = conf.update(usr.HomeDir + "/" + UserConfigFile)
	}

	return conf, err
}

// LoadSystemConfig loads only the system configuration
func LoadSystemConfig() (*Config, error) {

	conf := &Config{}
	err := conf.update(SystemConfigFile)

	return conf, err
}

// LoadUserConfig loads only the system configuration
func LoadUserConfig() (*Config, error) {

	conf := &Config{}

	usr, err := user.Current()
	if err == nil {
		conf.update(usr.HomeDir + "/" + UserConfigFile)
	}

	return conf, err
}

// getValue returns the reflect.Value for the element in the nested structure by the
// concatenated filter (using '.' as the separator). The filter is case-insensitive.
// This function also returns the actual path using the correctly capitalized letters
func (conf *Config) getValue(filter string, makeMap bool) (string, reflect.Value, string) {
	var realPath string
	var tag string

	elem := reflect.ValueOf(conf).Elem()
	path := strings.Split(filter, "/")

	for i, fieldName := range path {
		curElem := elem
		if elem.Kind() == reflect.Struct {
			elem = elem.FieldByNameFunc(func(fn string) bool {
				if strings.ToLower(fieldName) == strings.ToLower(fn) {
					fieldName = fn
					return true
				}
				return false
			})
		} else if elem.Kind() == reflect.Map {
			elem = elem.MapIndex(reflect.ValueOf(fieldName))
			if !elem.IsValid() {
				if !makeMap {
					return realPath, elem, ""
				}

				if curElem.IsNil() {
					curElem.Set(reflect.MakeMap(curElem.Type()))
				}
				elem = reflect.New(curElem.Type().Elem().Elem()).Elem().Addr()
				curElem.SetMapIndex(reflect.ValueOf(fieldName), elem)
			}
			elem = elem.Elem()
		} else {
			return realPath, elem, ""
		}
		realPath = realPath + fieldName

		if !elem.IsValid() {
			return realPath, elem, ""
		}
		if i != len(path)-1 {
			realPath = realPath + "/"
		} else if elem.Kind() == reflect.String {
			field, _ := curElem.Type().FieldByName(fieldName)
			tag = field.Tag.Get("cne")
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

func (conf *Config) SetByName(name string, value string) (string, string, error) {

	path, field, tag := conf.getValue(name, true)
	if !field.IsValid() {
		return "", path, errdefs.NotFound("configuration", name)
	}
	if field.Kind() != reflect.String {
		return "", "", errdefs.InvalidArgument("cannot set configuration '%s'", name)
	}

	if tag == "ReadOnly" {
		return "", "", errdefs.InvalidArgument("configuration '%s' is read-only", name)
	}

	oldValue := field.String()
	field.SetString(value)

	return oldValue, path, nil
}

// Get returns the value of the configuration field specified by name
// Errors:
//  - ErrNotFound if the specified configuration field cannot be found
func (conf *Config) GetByName(name string) (string, string, error) {

	path, field, _ := conf.getValue(name, false)
	if !field.IsValid() {
		return "", "", errdefs.NotFound("configuration", name)
	}

	return path, field.String(), nil
}

// GetAllByName returns a 'reflect.Value' for the selected field, which
// can be a structure for nested structures.
func (conf *Config) GetAllByName(name string) (string, interface{}, error) {

	path, field, _ := conf.getValue(name, false)
	if !field.IsValid() {
		return "", reflect.Value{}, errdefs.NotFound("configuration", name)
	}

	return path, field.Interface(), nil
}

// WriteSystemConfig writes the system configuration to /etc/cneconfig.
// This
func (conf *Config) WriteSystemConfig() error {

	file, err := os.OpenFile(SystemConfigFile, os.O_TRUNC|os.O_RDWR|os.O_CREATE, ConfigFilePerms)
	if err != nil {
		return errdefs.SystemError(err, "failed to open configuration file: %s",
			SystemConfigFile)
	}
	defer file.Close()
	defer file.Sync()

	writer := bufio.NewWriter(file)
	err = toml.NewEncoder(writer).Encode(conf)
	if err != nil {
		return errdefs.SystemError(err, "failed to write configuration file")
	}
	return nil
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
		return errdefs.SystemError(err, "failed to write configuration file '%s'", path)
	}
	defer file.Close()
	defer file.Sync()

	euid := os.Geteuid()
	uid := os.Getuid()
	if euid != uid {
		gid := os.Getgid()
		if err = file.Chown(uid, gid); err != nil {
			return errdefs.SystemError(err,
				"failed to update permissions for '%s'", path)
		}
	}

	writer := bufio.NewWriter(file)
	err = toml.NewEncoder(writer).Encode(conf)
	if err != nil {
		return errdefs.SystemError(err, "failed to write configuration file '%s'", path)
	}
	return nil
}

func (conf *Config) FullImageName(name string) string {

	reg, foundReg := conf.Registry[DefaultRegistryName]
	domEnd := strings.Index(name, "/") + 1
	if domEnd > 1 {
		reg, foundReg = conf.Registry[name[:domEnd-1]]
	}

	if foundReg {
		name = reg.Domain + "/" + reg.RepoName + "/" + name[domEnd:]
	}

	v := strings.LastIndex(name, ":")
	if v == -1 || v < domEnd {
		name = name + ":" + DefaultPackageVersion
	}

	return name
}

// GetUser returns the details and credentials of the current user
func (conf *Config) User() (User, error) {
	return getProcessUser()
}
