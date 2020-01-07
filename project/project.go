// Package project manages the project configuration
package project

import (
	"io/ioutil"
	"os"
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

// Project
type Project struct {
	Name                  string
	UUID                  string // Unique id for the project
	Workspaces            []Workspace
	path                  string
	currentWorkspaceIndex int // the main workspace must be at index 0!
	instanceID            uint64
	modifiedAt            time.Time
}

// Workspace is a specific environment of the project. They allow for building a development
// pipeline by propagating results to the following workspace.
// Note that Image cannot be changed and requires to create a new workspace
type Workspace struct {
	Name   string // Name of the workspace (must be unique)
	Origin string // Name or link of the base image
}

// New defines a new project with the provided name and path.
// The path can be empty, in which case the current working directory will be used.
// Create creates the project in the provide path
func Create(name string, path string) (*Project, error) {

	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	} else if path[len(path)-1] != '/' {
		path = path + "/"
	}

	path = path + projectDirName
	_, err := os.Stat(path)
	if err == nil {
		return nil, errdefs.ErrResourceExists
	} else if !os.IsNotExist(err) {
		return nil, errdefs.ErrResourceExists
	}

	err = os.MkdirAll(path, projectDirPerm)
	if err != nil {
		return nil, errdefs.ErrInvalidArgument
	}

	flags := os.O_RDONLY | os.O_CREATE | os.O_EXCL | os.O_SYNC
	file, err := os.OpenFile(path+projectFileName, flags, projectFilePerm)
	if err != nil {
		return nil, err
	}

	euid := os.Geteuid()
	uid := os.Getuid()
	gid := os.Getgid()
	if euid != uid {
		if err = file.Chown(uid, gid); err != nil {
			return nil, err
		}
	}
	file.Close()

	fileinfo, err := os.Stat(path)

	prj := &Project{
		Name:       name,
		UUID:       uuid.New().String(),
		modifiedAt: fileinfo.ModTime(),
		path:       path,
	}

	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if ok {
		prj.instanceID = stat.Ino
	}

	err = prj.Write()
	return prj, err
}

// LoadFrom loads the project from the provided path.
func LoadFrom(path string) (*Project, error) {

	if len(path) == 0 {
		return nil, errdefs.ErrInvalidArgument
	}
	if path[len(path)-1] != '/' {
		path = path + "/"
	}

	path = path + projectDirName

	str, err := ioutil.ReadFile(path + projectFileName)
	if os.IsNotExist(err) {
		return nil, errdefs.ErrNoSuchResource
	}
	if err != nil {
		return nil, errdefs.ErrInvalidArgument
	}

	var header Header
	yaml.Unmarshal(str, &header)

	var prj Project
	yaml.Unmarshal(str, &prj)

	fileinfo, err := os.Stat(path)
	prj.path = path
	prj.modifiedAt = fileinfo.ModTime()
	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if ok {
		prj.instanceID = stat.Ino
	}

	return &prj, nil
}

// Load loads the project from the current working directory.
func Load() (*Project, error) {

	path, err := os.Getwd()
	if err != nil {
		return nil, err
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
		return err
	}

	pStr, err := yaml.Marshal(prj)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(prj.path+projectFileName, append(hStr, pStr...), projectFilePerm)
}

// NewWorkspace creates a new workspace
func (prj *Project) NewWorkspace(name string) Workspace {

	ws := Workspace{
		Name: name,
	}

	return ws
}

// CurrentWorkspace retuns the current workspace. Returns NIL if current index it out of scope.
func (prj *Project) CurrentWorkspace() *Workspace {

	if prj.currentWorkspaceIndex >= len(prj.Workspaces) {
		return nil
	}

	return &prj.Workspaces[prj.currentWorkspaceIndex]
}

// SetCurrentWorkspace sets the current workspace;
func (prj *Project) SetCurrentWorkspace(name string) error {

	for i, ws := range prj.Workspaces {
		if name == ws.Name {
			prj.currentWorkspaceIndex = i
			return nil
		}
	}

	return errdefs.ErrNoSuchResource
}

// InsertWorkspace adds a workspace to the project before the provided workspace or at the end
// if 'before' is an empty string
func (prj *Project) InsertWorkspace(workspace Workspace, before string) error {

	idx := len(prj.Workspaces)
	for i, ws := range prj.Workspaces {
		if before == ws.Name {
			idx = i
		}
		if ws.Name == workspace.Name {
			return errdefs.ErrResourceExists
		}
	}
	if before != "" && idx == len(prj.Workspaces) {
		return errdefs.ErrNoSuchResource
	}
	prj.Workspaces = append(prj.Workspaces[:idx],
		append([]Workspace{workspace}, prj.Workspaces[idx:]...)...)

	return nil
}

// RemoveWorkspace removes the specified workspace.
// If it was the current workspace, the current workspace will be
// set to the main workspace. Note that the main workspace cannot be removed
func (prj *Project) RemoveWorkspace(name string) error {

	for i, ws := range prj.Workspaces {
		if name == ws.Name {
			prj.Workspaces = append(prj.Workspaces[:i], prj.Workspaces[i+1:]...)
			// FIXME simply go back index or use invalid index??
			if prj.currentWorkspaceIndex == i &&
				prj.currentWorkspaceIndex < len(prj.Workspaces) &&
				prj.currentWorkspaceIndex > 0 {
				prj.currentWorkspaceIndex--
			}
			return nil
		}
	}

	return errdefs.ErrNoSuchResource
}
