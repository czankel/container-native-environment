// Package project manages the project configuration
package project

import (
	"io/ioutil"
	"os"

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
	Name string
	UUID string // Unique id for the project
	path string
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

	prj := &Project{
		Name: name,
		UUID: uuid.New().String(),
		path: path,
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

	prj.path = path

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
