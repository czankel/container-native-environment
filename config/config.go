// Package config manages project-independent configurations.
// Configurations describe the environment for the projects for a user, and system-wide
// stored in /etc/cneconfig
package config

import (
	"bufio"
	"errors"
	"os"
	"os/user"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	refdocker "github.com/containerd/containerd/reference/docker"

	"github.com/czankel/cne/errdefs"
)

// CneVersion is set in Makefile by a linker option to the git hash/version
var CneVersion string
var contextName string

type Settings struct {
	Context string `toml:",omitempty"`
}

type Runtime struct {
	Engine     string `toml:"Engine,omitempty"`
	SocketName string `toml:"SocketName,omitempty"`
	Namespace  string `cne:"ReadOnly" toml:"Namespace,omitempty"`
}

type Registry struct {
	Domain   string
	RepoName string
}

type Context struct {
	Runtime  string
	Registry string
	Options  map[string]string `cne:"inline"`
}

type Config struct {
	Settings Settings
	Context  map[string]*Context
	Runtime  map[string]*Runtime
	Registry map[string]*Registry
}

func SetContextName(name string) {
	contextName = name
}

// update updates the configuration with the values from the specified configuration file
func (conf *Config) update(path string) error {
	_, err := toml.DecodeFile(path, conf)
	if err != nil && !os.IsNotExist(err) {
		return errdefs.InvalidArgument("config file '%s' corrupt", path)
	}
	return nil
}

func (conf *Config) CreateContext(name string) (*Context, error) {

	if _, found := conf.Context[name]; found {
		return nil, errdefs.AlreadyExists("context", name)
	}
	if len(conf.Context) == 0 {
		conf.Context = make(map[string]*Context, 1)
	}
	conf.Context[name] = &Context{}
	return conf.Context[name], nil
}

func (conf *Config) CreateRegistry(name string) (*Registry, error) {

	if _, found := conf.Registry[name]; found {
		return nil, errdefs.AlreadyExists("context", name)
	}
	if len(conf.Registry) == 0 {
		conf.Registry = make(map[string]*Registry, 1)
	}
	conf.Registry[name] = &Registry{}
	return conf.Registry[name], nil
}

func (conf *Config) CreateRuntime(name, engine string) (*Runtime, error) {

	if _, found := conf.Runtime[name]; found {
		return nil, errdefs.AlreadyExists("context", name)
	}
	if len(conf.Runtime) == 0 {
		conf.Runtime = make(map[string]*Runtime, 1)
	}
	conf.Runtime[name] = &Runtime{
		Engine:    engine,
		Namespace: DefaultRuntimeNamespace,
	}
	return conf.Runtime[name], nil
}

// GetContext returns the context defined in the configuration files or from
// the --context flag when CNE was started.
func (conf *Config) GetContext() (*Context, string, error) {

	name := contextName
	if name == "" {
		name = conf.Settings.Context
	}

	if c, found := conf.Context[name]; found {
		return c, name, nil
	}
	return nil, name, errdefs.NotFound("context", name)
}

// GetRuntime returns the specified runtime or context-specific runtime if name is empty
func (conf *Config) GetRuntime(args ...string) (*Runtime, error) {

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		cfgCtx, _, err := conf.GetContext()
		if err != nil {
			return nil, err
		}
		name = cfgCtx.Runtime
	}

	if r, found := conf.Runtime[name]; found {
		if r.Engine == "" {
			r.Engine = name
		}
		return r, nil
	}
	return nil, errdefs.InvalidArgument("invalid Engine '%s'", name)
}

// GetRegistry returns the specified registry or context-specific registry if name is empty
func (conf *Config) GetRegistry(args ...string) (*Registry, error) {

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		cfgCtx, _, err := conf.GetContext()
		if err != nil {
			return nil, err
		}
		name = cfgCtx.Registry
	}

	if r, found := conf.Registry[name]; found {
		return r, nil
	}

	return nil, errdefs.InvalidArgument("invalid registry '%s'", name)
}

func (conf *Config) RenameContext(from, to string) error {

	if _, ok := conf.Context[to]; ok {
		return errdefs.InvalidArgument("context with new name already exists")
	}

	conf.Context[to] = conf.Context[from]
	delete(conf.Context, from)

	return nil
}

func (conf *Config) RenameRuntime(from, to string) error {

	if _, ok := conf.Runtime[to]; ok {
		return errdefs.InvalidArgument("context with new name already exists")
	}

	conf.Runtime[to] = conf.Runtime[from]
	delete(conf.Runtime, from)

	for _, cc := range conf.Context {
		if cc.Runtime == from {
			cc.Runtime = to
		}
	}

	return nil
}

func (conf *Config) RenameRegistry(from, to string) error {

	if _, ok := conf.Registry[to]; ok {
		return errdefs.InvalidArgument("context with new name already exists")
	}

	conf.Registry[to] = conf.Registry[from]
	delete(conf.Registry, from)

	for _, cc := range conf.Context {
		if cc.Registry == from {
			cc.Registry = to
		}
	}

	return nil
}

func (conf *Config) RemoveContext(name string) error {

	if name == "" {
		return errdefs.InvalidArgument("no context provided")
	}
	if name == contextName {
		return errdefs.InvalidArgument("cannot delete current context")
	}
	delete(conf.Context, name)
	return nil
}

func (conf *Config) RemoveRuntime(name string) error {

	if name == "" {
		return errdefs.InvalidArgument("no runtime specified")
	}

	confCtx, _, err := conf.GetContext()
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	} else if err == nil {
		if name == confCtx.Runtime {
			return errdefs.InvalidArgument("cannot delete current runtime")
		}
		for k, c := range conf.Context {
			if c.Runtime == name {
				return errdefs.InvalidArgument("runtime still in use in context: '%s'", k)
			}
		}
	}

	delete(conf.Runtime, name)
	return nil
}

func (conf *Config) RemoveRegistry(name string) error {

	if name == "" {
		return errdefs.InvalidArgument("no registry specified")
	}
	confCtx, _, err := conf.GetContext()
	if err != nil {
		return err
	}

	if name == confCtx.Registry {
		return errdefs.InvalidArgument("cannot delete current registry")
	}
	for k, c := range conf.Context {
		if c.Registry == name {
			return errdefs.InvalidArgument("registry still in use in context: '%s'", k)
		}
	}

	delete(conf.Registry, name)
	return nil

}

func (conf *Config) GetEntryValue(entry string) (interface{}, error) {

	var val interface{} = conf
	var err error

	confCtx, _, err := conf.GetContext()
	if err != nil {
		return nil, err
	}

	switch entry {
	case "context":
		entry = "context/" + conf.Settings.Context
	case "runtime":
		entry = "runtime/" + confCtx.Runtime
	case "registry":
		entry = "registry/" + confCtx.Registry
	}

	return val, err
}

func (conf *Config) RenameEntry(entry, from, to string) error {

	var err error
	switch entry {
	case "context":
		err = conf.RenameContext(from, to)
	case "runtime":
		err = conf.RenameRuntime(from, to)
	case "registry":
		err = conf.RenameRegistry(from, to)
	}
	return err
}

// Load returns the default configuration amended by the configuration stored in the
// system and user configuration file.
func Load() (*Config, error) {

	conf := &Config{
		Settings: Settings{
			Context: DefaultContextName,
		},
		Context: map[string]*Context{
			DefaultContextName: &Context{
				Runtime:  DefaultRuntimeName,
				Registry: DefaultRegistryName,
			}},
		Registry: map[string]*Registry{
			DefaultRegistryName: &Registry{
				Domain:   DefaultRegistryDomain,
				RepoName: DefaultRegistryRepoName,
			}},
		Runtime: map[string]*Runtime{
			DefaultRuntimeEngine: &Runtime{
				Engine:     DefaultRuntimeEngine,
				SocketName: DefaultRuntimeSocketName,
				Namespace:  DefaultRuntimeNamespace,
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

func (confCtx *Context) UpdateContextOptions(line string) error {

	opts := strings.Split(line, ",")
	for _, o := range opts {
		kv := strings.Split(strings.TrimSpace(o), "=")
		if len(kv) != 2 {
			return errdefs.InvalidArgument("Option '%v' must be in the form key=[value]", o)
		}
		updated := false
		for k, _ := range confCtx.Options {
			if kv[0] == k {
				if kv[1] == "" {
					delete(confCtx.Options, k)
				} else {
					confCtx.Options[kv[0]] = kv[1]
				}
				updated = true
			}
		}
		if !updated {
			if len(confCtx.Options) == 0 {
				confCtx.Options = make(map[string]string, 1)
			}
			confCtx.Options[kv[0]] = kv[1]
		}
	}
	return nil
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
func (conf *Config) WriteSystemConfig() error {

	file, err := os.OpenFile(SystemConfigFile, os.O_TRUNC|os.O_RDWR|os.O_CREATE, ConfigFilePerms)
	if err != nil {
		return errdefs.SystemError(err, "failed to open configuration file: %s",
			SystemConfigFile)
	}
	defer file.Close()
	defer file.Sync()

	writer := bufio.NewWriter(file)

	// Skip Settings.Context in system
	oldCtx := conf.Settings.Context
	conf.Settings.Context = ""
	err = toml.NewEncoder(writer).Encode(conf)
	conf.Settings.Context = oldCtx
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
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_RDWR|os.O_CREATE, ConfigFilePerms)
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
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_RDWR|os.O_CREATE, ConfigFilePerms)
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

func (conf *Config) FullImageName(name string) (string, error) {

	cfgCtx, _, err := conf.GetContext()
	if err != nil {
		return "", err
	}

	regName := cfgCtx.Registry
	reg, haveReg := conf.Registry[regName]

	// check if a registry name was given, note that container names can also include a '/'
	domEnd := strings.Index(name, "/") + 1
	if domEnd > 1 {
		if r, ok := conf.Registry[name[:domEnd-1]]; !ok {
			domEnd = 0
		} else {
			regName = name[:domEnd-1]
			reg = r
			haveReg = true
		}
	}
	if haveReg {
		if regName == "docker.io" {
			named, err := refdocker.ParseDockerRef(name)
			if err != nil {
				return "", err
			}
			name = named.String()
		} else {
			name = reg.Domain + "/" + reg.RepoName + "/" + name[domEnd:]
		}
	}

	v := strings.LastIndex(name, ":")
	if v == -1 || v < domEnd {
		name = name + ":" + DefaultPackageVersion
	}

	return name, nil
}

// GetUser returns the details and credentials of the current user
func (conf *Config) User() (User, error) {
	return CurrentUser()
}
