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
func TestWorkspaceAddRemove(t *testing.T) {

	dir, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir)

	prj, err := Create("test", dir)
	if err != nil {
		t.Fatalf("Failed to create project")
	}

	ws0 := Workspace{
		Name:   "Workspace 0",
		Origin: "image0",
	}
	ws1 := Workspace{
		Name:   "Workspace 1",
		Origin: "image1",
	}
	ws2 := Workspace{
		Name:   "Workspace 2",
		Origin: "image2",
	}
	ws3 := Workspace{
		Name:   "Workspace 3",
		Origin: "image3",
	}

	// default should have no workspaces
	if len(prj.Workspaces) != 0 {
		t.Fatalf("Invalid number of workspaces for new project")
	}

	// append: [Workspace 0]
	err = prj.InsertWorkspace(ws0, "")
	if err != nil {
		t.Errorf("Failed to add Workspace 0")
	}
	if len(prj.Workspaces) != 1 {
		t.Errorf("Invalid number of workspaces after adding Workspace 0")
	}
	if prj.Workspaces[0].Name != ws0.Name {
		t.Errorf("Workspace 0 should be at index 0: " + prj.Workspaces[0].Name)
	}

	// append: [ws0], [ws1]
	err = prj.InsertWorkspace(ws1, "")
	if err != nil {
		t.Errorf("Failed to add Workspace 1")
	}
	if len(prj.Workspaces) != 2 {
		t.Errorf("Invalid number of workspaces after adding Workspace 1")
	}
	if prj.Workspaces[0].Name != ws0.Name {
		t.Errorf("Workspace 0 should be at index 0: " + prj.Workspaces[0].Name)
	}
	if prj.Workspaces[1].Name != ws1.Name {
		t.Errorf("Workspace 1 should be at index 1: " + prj.Workspaces[1].Name)
	}

	// insert: [ws0], [ws2], [ws1]
	err = prj.InsertWorkspace(ws2, "Workspace 1")
	if err != nil {
		t.Errorf("Failed to add Workspace 2")
	}
	if len(prj.Workspaces) != 3 {
		t.Errorf("Invalid number of workspaces after adding Workspace3")
	}
	if prj.Workspaces[0].Name != ws0.Name {
		t.Errorf("Workspace 0 should be at index 0: " + prj.Workspaces[0].Name)
	}
	if prj.Workspaces[1].Name != ws2.Name {
		t.Errorf("Workspace 2 should be at index 1: " + prj.Workspaces[1].Name)
	}
	if prj.Workspaces[2].Name != ws1.Name {
		t.Errorf("Workspace 1 should be at index 2: " + prj.Workspaces[2].Name)
	}

	// insert first: [ws3], [ws0], [ws2], [ws1]
	err = prj.InsertWorkspace(ws3, "Workspace 0")
	if err != nil {
		t.Errorf("Failed to add Workspace 3")
	}
	if len(prj.Workspaces) != 4 {
		t.Errorf("Invalid number of workspaces after adding Workspace 3")
	}
	if prj.Workspaces[0].Name != ws3.Name {
		t.Errorf("Workspace 3 should be at index 0: " + prj.Workspaces[0].Name)
	}
	if prj.Workspaces[1].Name != ws0.Name {
		t.Errorf("Workspace 0 should be at index 1: " + prj.Workspaces[1].Name)
	}
	if prj.Workspaces[2].Name != ws2.Name {
		t.Errorf("Workspace 2 should be at index 2: " + prj.Workspaces[2].Name)
	}
	if prj.Workspaces[3].Name != ws1.Name {
		t.Errorf("Workspace 1 should be at index 3: " + prj.Workspaces[3].Name)
	}

	ws2a := ws2
	err = prj.InsertWorkspace(ws2a, "")
	if err == nil {
		t.Errorf("Should have failed to add workspace with same name")
	}

	// remove ws0 inside: [ws3], [ws0], [ws2], [ws1]
	if prj.RemoveWorkspace("Workspace 0") != nil {
		t.Errorf("Failed to remove Workspace 0")
	}
	if len(prj.Workspaces) != 3 {
		t.Errorf("Invalid number of workspaces after removing Workspace 0")
	}

	// remove ws1 at the end: [ws3], [ws2], [ws1]
	if prj.RemoveWorkspace("Workspace 1") != nil {
		t.Errorf("Failed to remove Workspace 1")
	}
	if len(prj.Workspaces) != 2 {
		t.Errorf("Invalid number of workspaces after removing Workspace 1")
	}
	// remove ws3 at the beginning: [ws3], [ws2]
	if prj.RemoveWorkspace("Workspace 3") != nil {
		t.Errorf("Failed to remove Workspace 3")
	}
	if len(prj.Workspaces) != 1 {
		t.Errorf("Invalid number of workspaces after removing Workspace 3")
	}
	// remove last ws2: [ws2]
	if prj.RemoveWorkspace("Workspace 2") != nil {
		t.Errorf("Failed to remove Workspace 2")
	}
	if len(prj.Workspaces) != 0 {
		t.Errorf("Invalid number of workspaces after removing Workspace 2")
	}

}
