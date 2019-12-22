package project

import (
	"testing"

	"io/ioutil"
	"os"

	"github.com/czankel/cne/errdefs"
)

const testDir = "cnetest"

func TestProjectCreate(t *testing.T) {

	dir, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir)

	_, err = LoadFrom(dir)
	if err != errdefs.ErrNoSuchResource {
		t.Fatalf("Should have failed to load non-existent project: %v", err)
	}

	prj, err := Create("test", dir)
	if err != nil {
		t.Fatalf("Failed to create new project: %v", err)
	}

	prjChk, err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("Failed to load project: %v", err)
	}

	if prjChk == nil {
		t.Fatalf("Failed to load project: empty project")
	}

	if prjChk.Name != prj.Name {
		t.Errorf("Loaded project: Name field mismatch")
	}

	if prjChk.path != prj.path {
		t.Errorf("Loaded project: path field mismatch")
	}
}

func TestProjectCreateExistingProject(t *testing.T) {

	dir, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir)

	_, err = Create("test", dir)
	if err != nil {
		t.Errorf("Failed to create new project")
	} else {
		_, err = Create("test", dir)
		if err == nil {
			t.Errorf("Create over existing project should have failed")
		}
	}
}
