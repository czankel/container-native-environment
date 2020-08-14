// Package project manages the project configuration
package project

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"

	"github.com/czankel/cne/errdefs"
)

const projectDirName = ".cne/"
const projectFileName = "project"
const projectFileVersion = "1.0"
const projectFilePerm = 0600
const projectDirPerm = 0770

type Header struct {
	Version string
}

// Project is the current persistent project definition.
type Project struct {
	Name                 string
	UUID                 string // Universal Unique id for the project
	Workspaces           []Workspace
	CurrentWorkspaceName string
	path                 string
	instanceID           uint64
	modifiedAt           time.Time
}

// Workspace is a specific environment of the project. They allow for building a development
// pipeline by propagating results to the following workspace.
// Note that Image cannot be changed and requires to create a new workspace
type Workspace struct {
	Name        string // Name of the workspace (must be unique)
	Environment Environment
	ProjectUUID string `yaml:"-" output:"-"`
}

// Environment describes the container-native environment
type Environment struct {
	Origin       string // Name or link of the base image - cannot be changed
	Capabilities []string
	Layers       []Layer
}

// Layer describes an 'overlay' layer. This can be virtual or explicit using an overlay FS
type Layer struct {
	Name     string   // Unique name for the layer in the workspace; must not contain '/'
	Commands []string // Commands to build the layer (except for hidden layers)
	Digest   string   `hash:"-"` // Images/Snaps for faster rebuilds. Intermediates opt.
}

// Create creates the project in the provide path
// The path can be empty to use the current working directory.
func Create(name string, path string) (*Project, error) {

	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return nil, errdefs.SystemError(err, "failed to get work directory")
		}
	} else if path[len(path)-1] != '/' {
		path = path + "/"
	}

	path = path + projectDirName
	_, err := os.Stat(path)
	if err == nil {
		return nil, errdefs.AlreadyExists("project", path)
	} else if !os.IsNotExist(err) {
		return nil, errdefs.SystemError(err, "cannot read project in '%s'", path)
	}

	err = os.MkdirAll(path, projectDirPerm)
	if err != nil {
		return nil, errdefs.SystemError(err,
			"failed to create project directory '%s'", path)
	}

	flags := os.O_RDONLY | os.O_CREATE | os.O_EXCL | os.O_SYNC
	file, err := os.OpenFile(path+projectFileName, flags, projectFilePerm)
	if err != nil {
		return nil, errdefs.SystemError(err,
			"failed to create project file '%s'", path)
	}

	euid := os.Geteuid()
	uid := os.Getuid()
	gid := os.Getgid()
	if euid != uid {
		if err = file.Chown(uid, gid); err != nil {
			return nil, errdefs.SystemError(err,
				"failed to change file permissions '%s'", path)
		}
	}
	file.Close()

	fileInfo, err := os.Stat(path)

	prj := &Project{
		Name:       name,
		UUID:       uuid.New().String(),
		modifiedAt: fileInfo.ModTime(),
		path:       path,
	}

	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if ok {
		prj.instanceID = stat.Ino
	}

	err = prj.Write()
	return prj, err
}

// LoadFrom loads the project from the provided path.
func LoadFrom(path string) (*Project, error) {

	if len(path) == 0 {
		return nil, errdefs.InvalidArgument("invalid path: '%s'", path)
	}
	if path[len(path)-1] != '/' {
		path = path + "/"
	}

	path = path + projectDirName

	str, err := ioutil.ReadFile(path + projectFileName)
	if os.IsNotExist(err) {
		return nil, errdefs.NotFound("project", path)
	}
	if err != nil {
		return nil, errdefs.SystemError(err, "failed to read project file '%s'", path)
	}

	var header Header
	yaml.Unmarshal(str, &header)

	var prj Project
	yaml.Unmarshal(str, &prj)

	fileInfo, err := os.Stat(path)
	prj.path = path
	prj.modifiedAt = fileInfo.ModTime()
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if ok {
		prj.instanceID = stat.Ino
	}

	// Fixup workspaces
	for i := 0; i < len(prj.Workspaces); i++ {
		prj.Workspaces[i].ProjectUUID = prj.UUID
	}

	return &prj, nil
}

// Load loads the project from the current working directory.
func Load() (*Project, error) {

	path, err := os.Getwd()
	if err != nil {
		return nil, errdefs.SystemError(err, "failed to load project file in '%s'", path)
	}

	return LoadFrom(path)
}

// Write writes the project to the project path
func (prj *Project) Write() error {

	header := &Header{
		projectFileVersion,
	}
	hStr, err := yaml.Marshal(header)
	if err != nil {
		return errdefs.InvalidArgument("project file corrupt")
	}

	pStr, err := yaml.Marshal(prj)
	if err != nil {
		return errdefs.InvalidArgument("project file corrupt")
	}

	err = ioutil.WriteFile(prj.path+projectFileName, append(hStr, pStr...), projectFilePerm)
	if err != nil {
		return errdefs.SystemError(err, "failed to write project")
	}
	return nil
}

// CurrentWorkspace retuns a pointer to the current workspace or nil if unset or no workspaces.
// Returns an error if the workspace wasn't found.
func (prj *Project) CurrentWorkspace() (*Workspace, error) {
	return prj.Workspace(prj.CurrentWorkspaceName)
}

// Workspace returns a pointer to the workspace in the project specified by the provided name
// or error if it doesn't exist in the project
func (prj *Project) Workspace(name string) (*Workspace, error) {

	if name == "" && len(prj.Workspaces) > 0 {
		return &prj.Workspaces[0], nil
	}

	for i, w := range prj.Workspaces {
		if name == w.Name {
			return &prj.Workspaces[i], nil
		}
	}
	return nil, errdefs.NotFound("workspace", name)
}

// SetCurrentWorkspace sets the current workspace;
func (prj *Project) SetCurrentWorkspace(name string) error {

	if name == "" {
		prj.CurrentWorkspaceName = ""
		return nil
	}

	for _, ws := range prj.Workspaces {
		if name == ws.Name {
			prj.CurrentWorkspaceName = name
			return nil
		}
	}

	return errdefs.NotFound("workspace", name)
}

// CreateWorkspace creates a new workspace in the project before the provided workspace
// or at the end if 'before' is an empty string.
// It returns the pointer to the workspace in the current project.
func (prj *Project) CreateWorkspace(name string, origin string, before string) (*Workspace, error) {

	if name == "" {
		name = "main"
		idx := 0
		for i := 0; i < len(prj.Workspaces); i++ {
			if name == prj.Workspaces[i].Name {
				name = "ws-" + strconv.Itoa(idx)
				idx++
				i = 0
			}
		}
	}

	workspace := Workspace{
		Name:        name,
		ProjectUUID: prj.UUID,
		Environment: Environment{Origin: origin},
	}
	idx := len(prj.Workspaces)
	for i, ws := range prj.Workspaces {
		if before == ws.Name {
			idx = i
		}
		if ws.Name == workspace.Name {
			return nil, errdefs.AlreadyExists("workspace", workspace.Name)
		}
	}
	if before != "" && idx == len(prj.Workspaces) {
		return nil, errdefs.NotFound("workspace", workspace.Name)
	}

	prj.Workspaces = append(prj.Workspaces[:idx],
		append([]Workspace{workspace}, prj.Workspaces[idx:]...)...)

	return &prj.Workspaces[idx], nil
}

// DeleteWorkspace removes the specified workspace.
// If it was the current workspace, the current workspace will become unset
func (prj *Project) DeleteWorkspace(name string) error {

	for i, ws := range prj.Workspaces {
		if name == ws.Name {
			prj.Workspaces = append(prj.Workspaces[:i], prj.Workspaces[i+1:]...)
			if prj.CurrentWorkspaceName == ws.Name {
				prj.CurrentWorkspaceName = ""
			}
			return nil
		}
	}

	return errdefs.NotFound("workspace", name)
}

//
// Workspace
//

// hashValueElem is a helper function to recursively hash a Value
func hashValueElem(w io.Writer, prefix string, elem reflect.Value) {

	kind := elem.Kind()

	if prefix != "" && (kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice) {
		prefix = prefix + "/"
	}

	if kind == reflect.Struct {
		elemType := elem.Type()
		for i := 0; i < elem.NumField(); i++ {
			field := elemType.Field(i)
			if field.Tag.Get("hash") != "-" {
				hashValueElem(w, prefix+field.Name, elem.Field(i))
			}
		}
	} else if kind == reflect.Map {
		m := elem.MapKeys()
		keys := make([]string, len(m))
		for i := 0; i < len(m); i++ {
			keys[i] = m[i].String()
		}
		sort.Strings(keys)
		for _, k := range keys {
			hashValueElem(w, prefix+k, elem.MapIndex(reflect.ValueOf(k)))
		}
	} else if kind == reflect.Slice {
		for i := 0; i < elem.Len(); i++ {
			hashValueElem(w, prefix+strconv.Itoa(i), elem.Index(i))
		}
	} else if kind == reflect.Ptr {
		hashValueElem(w, prefix, elem.Elem())
	} else if elem.CanInterface() {
		w.Write([]byte(prefix))
		str := fmt.Sprintf("%v", elem.Interface())
		w.Write([]byte(str))
	}
}

// ID returns an identification for the workspace
func (ws *Workspace) ID() [16]byte {
	var id [16]byte

	hash := md5.New()
	hashValueElem(hash, "", reflect.ValueOf(ws.Name))
	copy(id[:], hash.Sum(nil)[:])

	return id
}

// ConfigHash returns a unique hash over the enitre Workspace structure.
func (ws *Workspace) ConfigHash() [16]byte {

	var gen [16]byte

	hash := md5.New()
	hashValueElem(hash, "", reflect.ValueOf(ws))
	copy(gen[:], hash.Sum(nil)[:])

	return gen
}

// CreateLayer inserts a new layer (or layers) at the provided index, or at the end if index == -1
func (ws *Workspace) CreateLayer(name string, atIndex int) (*Layer, error) {

	if atIndex < -1 || atIndex > len(ws.Environment.Layers) {
		return nil, errdefs.InvalidArgument("invalid index: %d", atIndex)
	}
	if atIndex == -1 {
		atIndex = len(ws.Environment.Layers)
	}

	for _, l := range ws.Environment.Layers {
		if name == l.Name {
			return nil, errdefs.AlreadyExists("layer", name)
		}
	}

	ws.Environment.Layers = append(ws.Environment.Layers[:atIndex],
		append([]Layer{Layer{Name: name}}, ws.Environment.Layers[atIndex:]...)...)

	return &ws.Environment.Layers[atIndex], nil
}

func (ws *Workspace) FindLayer(name string) *Layer {
	for i, l := range ws.Environment.Layers {
		if name == l.Name {
			return &ws.Environment.Layers[i]
		}
	}
	return nil
}

// DeleteLayer removes the specified layer.
func (ws *Workspace) DeleteLayer(name string) error {
	for i, l := range ws.Environment.Layers {
		if name == l.Name {
			ws.Environment.Layers = append(ws.Environment.Layers[:i],
				ws.Environment.Layers[i+1:]...)
			return nil
		}
	}
	return errdefs.NotFound("layer", name)
}

// TopLayer returns the pointer to the top layer.
func (ws *Workspace) TopLayer() *Layer {
	cnt := len(ws.Environment.Layers)
	if cnt == 0 {
		return nil
	}

	return &ws.Environment.Layers[cnt-1]
}
