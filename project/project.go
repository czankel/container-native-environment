// Package project manages the project configuration
package project

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/czankel/cne/errdefs"
)

const (
	ProjectFileName    = "cneproject"
	projectFileVersion = "1.0"
	projectFilePerm    = 0600

	WorkspaceDefaultName = "main"

	LayerTypeCustom = ""
	LayerTypeImage  = "image"
	LayerTypeUbuntu = "ubuntu"
	LayerTypeApt    = "apt"
	LayerTypeDnf    = "dnf"
	LayerTypePip    = "pip"

	LayerNameImage = "image"
	LayerNameTop   = ""
)

var SystemLayerTypes = [...]string{
	LayerTypeImage,
	LayerTypeUbuntu,
	LayerTypeApt,
	LayerTypeDnf,
	LayerTypePip,
}

type Header struct {
	Version string
}

// Project is the current persistent project definition.
type Project struct {
	Name                 string
	UUID                 string // Universal Unique id for the project
	CurrentWorkspaceName string
	Workspaces           []Workspace
	path                 string // path to the project file
	instanceID           uint64
	modifiedAt           time.Time
}

// Workspace is a specific environment of the project. They allow for building a development
// pipeline by propagating results to the following workspace.
// Note that Image cannot be changed and requires to create a new workspace
type Workspace struct {
	Name        string // Name of the workspace (must be unique)
	ProjectUUID string `yaml:"-" output:"-"`
	Environment Environment
}

// Environment describes the container-native environment
// The options for the update strategy are as follows:
//  * "never"  -  packages are never updated
//  * "manual" -  manually (re-)building the image will update the packages
//  * "auto"   -  packages will be updated whenever the package layer(s) are rebuild
// Note that the image needs to be pulled manually to cause an update (using 'pull')
type Environment struct {
	Origin string // Name or link of the base image
	Update string // Update package strategy: One of "never", "manual", "auto"
	Layers []Layer
}

// Layer describes an 'overlay' layer. This can be virtual or explicit using an overlay FS
// Note that ideally we could use compositions for apt and other handlers
type Layer struct {
	Name     string // Unique name for the layer in the workspace; must not contain '/'
	Type     string // one of the system layer types or custom layer
	Digest   string `output:"-"` // Images/Snaps for faster rebuilds
	Commands []Command
}

// Command describes the command and its argument(s).
// The Name is optional and used by support functions to manage the command list.
type Command struct {
	Name string
	Envs []string `output:"flat" yaml:",flow"`
	Args []string `output:"flat" yaml:",flow"`
}

// Create creates the project in the provide path
// The path can be empty to use the current working directory.
// Callers can provide a directory or a filename. If a filename is provided, it must not exist.
// In general, using a different configuration file than the standard "cneproject" is
// discouraged.
func Create(name, path string) (*Project, error) {

	isDir := true
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return nil, errdefs.SystemError(err, "failed to get work directory")
		}
	} else {
		fileInfo, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, errdefs.InvalidArgument("invalid project path: '%s' (%v)", path, err)
		}
		isDir = fileInfo.Mode().IsDir()
		if isDir && err != nil {
			return nil, errdefs.InvalidArgument("invalid project path: '%s' (%v)", path, err)
		} else if !isDir && err == nil {
			return nil, errdefs.AlreadyExists("project", path)
		}
	}
	if isDir {
		path = path + "/" + ProjectFileName
	}

	flags := os.O_RDONLY | os.O_CREATE | os.O_EXCL | os.O_SYNC
	file, err := os.OpenFile(path, flags, projectFilePerm)
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
				"failed to change file permissions of the project file in '%s'", path)
		}
	}
	file.Close()

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, errdefs.SystemError(err, "failed to stat file '%s'", path)
	}

	prj := &Project{
		Name:       name,
		UUID:       uuid.New().String(),
		path:       path,
		modifiedAt: fileInfo.ModTime(),
	}

	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if ok {
		prj.instanceID = stat.Ino
	}

	err = prj.Write()
	return prj, err
}

// Load loads the project from the provided path.
// It also scans all parent paths for the project file if path is a directory.
func Load(path string) (*Project, error) {

	if len(path) == 0 {
		return nil, errdefs.InvalidArgument("invalid project path: '%s'", path)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, errdefs.InvalidArgument("invalid project path: '%s' (%v)", path, err)
	}

	var prjStr []byte
	if fileInfo.Mode().IsDir() {
		if path[len(path)-1] != '/' {
			path = path + "/"
		}
		dir := path
		path = path + ProjectFileName

		prjStr, err = ioutil.ReadFile(path)
		for err != nil && os.IsNotExist(err) {
			if dir == "/" || dir == "." {
				return nil, errdefs.NotFound("project", path)
			}
			dir = filepath.Dir(dir)
			path = dir + "/" + ProjectFileName
			prjStr, err = ioutil.ReadFile(path)
		}
	} else {
		prjStr, err = ioutil.ReadFile(path)
	}
	if err != nil {
		return nil, errdefs.SystemError(err, "failed to read project file '%s'", path)
	}

	var header Header
	err = yaml.Unmarshal(prjStr, &header)
	if err != nil {
		return nil, errdefs.InvalidArgument("project file corrupt: %v", err)
	}

	var prj Project
	err = yaml.Unmarshal(prjStr, &prj)
	if err != nil {
		return nil, errdefs.InvalidArgument("project file corrupt: %v", err)
	}

	fileInfo, err = os.Stat(path)
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

	err = ioutil.WriteFile(prj.path, append(hStr, pStr...), projectFilePerm)
	if err != nil {
		return errdefs.SystemError(err, "failed to write project")
	}
	return nil
}

// Delete removes the CNE project file
func (prj *Project) Delete() error {
	return os.Remove(prj.path)
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
		name = WorkspaceDefaultName
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
		Environment: Environment{Origin: origin, Layers: []Layer{}},
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
func hashValueElem(w io.Writer, prefix string, elem reflect.Value, deep bool) {

	kind := elem.Kind()

	if deep && (kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice) {
		prefix = prefix + "/"
	}

	if kind == reflect.Struct {
		elemType := elem.Type()
		for i := 0; i < elem.NumField(); i++ {
			field := elemType.Field(i)
			tag := field.Tag.Get("hash")
			if tag != "-" {
				hashValueElem(w, prefix+field.Name, elem.Field(i), true)
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
			hashValueElem(w, prefix+k, elem.MapIndex(reflect.ValueOf(k)), true)
		}
	} else if kind == reflect.Slice {
		for i := 0; i < elem.Len(); i++ {
			hashValueElem(w, prefix+strconv.Itoa(i), elem.Index(i), true)
		}
	} else if kind == reflect.Ptr {
		hashValueElem(w, prefix, elem.Elem(), deep)
	} else if elem.CanInterface() {
		w.Write([]byte(prefix))
		str := fmt.Sprintf("%v", elem.Interface())
		w.Write([]byte(str))
	}
}

// ID returns an identification for the workspace
func (ws *Workspace) ID() [16]byte {
	return md5.Sum([]byte(ws.Name))
}

// BaseHash returns a unique hash value for a build container
func (ws *Workspace) BaseHash() [16]byte {

	var gen [16]byte

	hashVal := md5.New()
	hashValueElem(hashVal, "", reflect.ValueOf(ws.Environment), false /* deep */)
	copy(gen[:], hashVal.Sum(nil)[:])

	return gen
}

// ConfigHash returns a unique hash over the Workspace Environment.
func (ws *Workspace) ConfigHash() [16]byte {

	var gen [16]byte

	hashVal := md5.New()
	hashValueElem(hashVal, "", reflect.ValueOf(ws.Environment), true /* deep */)
	copy(gen[:], hashVal.Sum(nil)[:])

	return gen
}

// CreateLayer inserts a new layer (or layers) at the provided index, or at the end if index == -1
func (ws *Workspace) CreateLayer(systemLayer bool, name string, atIndex int) (*Layer, error) {

	layerType := LayerTypeCustom
	if systemLayer {
		layerType = name
	}

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
		append([]Layer{Layer{Type: layerType, Name: name}},
			ws.Environment.Layers[atIndex:]...)...)

	return &ws.Environment.Layers[atIndex], nil
}

// FindLayer looks up the layer by name and returns the layer index starting with 0 for the first
// layer in the list and a pointer to the Layer structure.
// If the layer cannot be found, it returns an index value of -1 and nil for the layer.
func (ws *Workspace) FindLayer(name string) (int, *Layer) {
	for i, l := range ws.Environment.Layers {
		if name == l.Name {
			return i, &ws.Environment.Layers[i]
		}
	}
	return -1, nil
}

// DeleteLayer removes the specified layer.
func (ws *Workspace) DeleteLayer(name string) error {

	deleted := 0
	for i := 0; i < len(ws.Environment.Layers); i++ {
		l := ws.Environment.Layers[i]
		prefix := name + "."
		if name == l.Name || strings.HasPrefix(l.Name, prefix) {
			ws.Environment.Layers = append(ws.Environment.Layers[:i],
				ws.Environment.Layers[i+1:]...)
			deleted++
			i--
		}
	}
	if deleted == 0 {
		return errdefs.NotFound("layer", name)
	}
	return nil
}

// TopLayer returns the pointer to the top layer.
func (ws *Workspace) TopLayer() *Layer {
	cnt := len(ws.Environment.Layers)
	if cnt == 0 {
		return nil
	}

	return &ws.Environment.Layers[cnt-1]
}

// UpdateLayer ensures that any cached or reference data is updated
func (ws *Workspace) UpdateLayer(layer *Layer) {

	invalidate := false
	for i := 0; i < len(ws.Environment.Layers); i++ {
		l := &ws.Environment.Layers[i]
		if l.Name == layer.Name {
			invalidate = true
		}
		if invalidate {
			l.Digest = ""
		}
	}
}
