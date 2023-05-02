// Package config manages project-independent configurations.
// Configurations describe the environment for the projects for a user, and system-wide
// stored in /etc/cneconfig
package config

import (
	"bufio"
	"os"
	"os/user"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/czankel/cne/errdefs"
)

// CneVersion is set in Makefile by a linker option to the git hash/version
var CneVersion string

type Settings struct {
	Context string
}

type Runtime struct {
	Name       string `toml:"Name,omitempty"`
	SocketName string `toml:"SocketName,omitempty"`
	Namespace  string `cne:"ReadOnly" toml:"Namespace,omitempty"`
}

type Registry struct {
	Name     string
	Domain   string
	RepoName string
}

type Context struct {
	Name     string
	Runtime  string
	Registry string
}

type Config struct {
	Settings Settings
	Context  []*Context
	Runtime  []*Runtime
	Registry []*Registry
}

var ContextName string

// GetContext returns the context defined in the configuration files or from
// the --context flag when CNE was started.
func (conf *Config) GetContext() (*Context, error) {

	name := ContextName
	if name == "" {
		name = conf.Settings.Context
	}

	for _, c := range conf.Context {
		if name == c.Name {
			return c, nil
		}
	}
	return nil, errdefs.InvalidArgument("invalid context '%s'", name)
}

// GetRuntime returns the context-specific runtime.
func (conf *Config) GetRuntime() (*Runtime, error) {

	cfgCtx, err := conf.GetContext()
	if err != nil {
		return nil, err
	}

	for _, r := range conf.Runtime {
		if r.Name == cfgCtx.Runtime {
			return r, nil
		}
	}

	return nil, errdefs.InvalidArgument("invalid runtime '%s' for context '%s'",
		cfgCtx.Runtime, cfgCtx.Name)
}

// GetRegistry returns the context-specific registry.
func (conf *Config) GetRegistry() (*Registry, error) {

	cfgCtx, err := conf.GetContext()
	if err != nil {
		return nil, err
	}

	for _, r := range conf.Registry {
		if r.Name == cfgCtx.Registry {
			return r, nil
		}
	}

	return nil, errdefs.InvalidArgument("invalid registry '%s' for context '%s'",
		cfgCtx.Runtime, cfgCtx.Name)
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
		Settings: Settings{
			Context: DefaultContextName,
		},
		Context: []*Context{
			&Context{
				Name:     DefaultContextName,
				Runtime:  DefaultRuntimeName,
				Registry: DefaultRegistryName,
			}},
		Runtime: []*Runtime{
			&Runtime{
				Name:       DefaultRuntimeName,
				SocketName: DefaultRuntimeSocketName,
				Namespace:  DefaultRuntimeNamespace,
			}},
		Registry: []*Registry{
			&Registry{
				Name:     DefaultRegistryName,
				Domain:   DefaultRegistryDomain,
				RepoName: DefaultRegistryRepoName,
			}},
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

// LoadProjectConfig loads only the project configuration, if exists
func LoadProjectConfig(path string) (*Config, error) {

	conf := &Config{}
	err := conf.update(path + "/" + ProjectConfigFile)

	return conf, err
}

// UpdateProjectConfig updates the configuration from the config file in the project path
func (conf *Config) UpdateProjectConfig(path string) error {

	return conf.update(path + "/" + ProjectConfigFile)
}

// getValue returns the reflect.Value for the element in the nested structure by the
// concatenated filter (using '/' as the separator). The filter is case-insensitive.
// For arrays, both, the index and the name can be used, if a "Name" field exists.
// This function also returns the actual path using the correctly capitalized letters
// and converts arrays selected by name to the array index.
// Note that an invalid value returned indicates an error.
func (conf *Config) getValue(filter string, create bool) (string, reflect.Value, string) {
	var realPath string
	var tag string

	elem := reflect.ValueOf(conf).Elem()
	filter = strings.TrimRight(filter, "/")
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
				if !create {
					return realPath, elem, ""
				}

				if curElem.IsNil() {
					curElem.Set(reflect.MakeMap(curElem.Type()))
				}
				elem = reflect.New(curElem.Type().Elem().Elem()).Elem().Addr()
				curElem.SetMapIndex(reflect.ValueOf(fieldName), elem)
			}
			elem = elem.Elem()

		} else if elem.Kind() == reflect.Slice {

			idx, err := strconv.Atoi(fieldName)
			if err != nil {

				idx = -1
				fn := strings.ToLower(fieldName)
				for i := 0; i < curElem.Len(); i++ {

					e := curElem.Index(i).Elem()
					v := e.FieldByNameFunc(func(n string) bool {
						return strings.ToLower(n) == "name"
					})
					if v.IsValid() && strings.ToLower(v.String()) == fn {
						idx = i
						fieldName = strconv.Itoa(i)
						break
					}
				}
			}

			if idx >= 0 && idx < curElem.Len() {
				elem = curElem.Index(idx)
			} else if create {
				elem = reflect.New(curElem.Type().Elem().Elem()).Elem().Addr()
				reflect.Append(curElem, elem)
			} else {
				return "", reflect.ValueOf(nil), ""
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
//   - ErrNotFound if the specified configuration field cannot be found
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

// WriteLocalConfig writes the configuration to the project directory
func (conf *Config) WriteProjectConfig(path string) error {

	path = path + "/" + ProjectConfigFile
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, ConfigFilePerms)
	if err != nil {
		return errdefs.SystemError(err, "failed to write configuration file '%s'", path)
	}
	defer file.Close()
	defer file.Sync()

	writer := bufio.NewWriter(file)
	err = toml.NewEncoder(writer).Encode(conf)
	if err != nil {
		return errdefs.SystemError(err, "failed to write configuration file '%s'", path)
	}
	return nil
}

func (conf *Config) FullImageName(name string) string {

	reg, err := conf.GetRegistry()
	foundReg := err != nil

	domEnd := strings.Index(name, "/") + 1
	if domEnd > 1 {
		regName := name[:domEnd-1]
		for _, r := range conf.Registry {
			if regName == r.Name {
				reg = r
				foundReg = true
				break
			}
		}
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
	return CurrentUser()
}
