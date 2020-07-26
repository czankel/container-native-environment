package project

import (
	"testing"
	"time"

	"io"
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

	timediff := time.Now().Sub(prj.modifiedAt)
	if timediff < 0 {
		t.Errorf("modifiedAt later than 'Now': %v", timediff)
	}
	if timediff > 1000000000 {
		t.Errorf("modifiedAt earlier than 1s: %v", timediff)
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

// create project in dir test1
// copy project to test2
// update project
func TestCopy(t *testing.T) {

	dir, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir)

	prj, err := Create("test", dir+"/test1")
	if err != nil {
		t.Errorf("Failed to create new project: %v", err)
	}
	if prj.instanceID == 0 {
		t.Errorf("Project ID still 0")
	}

	src, err := os.Open(dir + "/test1/" + projectDirName + projectFileName)
	if err != nil {
		t.Fatalf("Failed to open project file: %v", err)
	}
	defer src.Close()

	err = os.MkdirAll(dir+"/test2/"+projectDirName, 0755)
	if err != nil {
		t.Fatalf("Failed to create destination directory")
	}

	dst, err := os.Create(dir + "/test2/" + projectDirName + projectFileName)
	if err != nil {
		t.Fatalf("Failed to create destination file")
	}
	defer dst.Close()

	wr, err := io.Copy(dst, src)
	if err != nil {
		t.Fatalf("Failed to copy project %s", err)
	}
	if wr == 0 {
		t.Fatalf("0 bytes copied")
	}

	prjChk, err := LoadFrom(dir + "/test2")
	if prjChk.instanceID == prj.instanceID {
		t.Fatalf("Copying project should have changed node id")
	}

	err = prjChk.Write()

	prj3, err := LoadFrom(dir + "/test2")
	if err != nil {
		t.Fatalf("Failed to load project")
	} else if prj3.instanceID != prjChk.instanceID {
		t.Fatalf("Project instanceID should have been saved")
	}

}

func TestTwoProjects(t *testing.T) {
	dir1, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir1)

	dir2, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir2)

	prj1, err := Create("test1", dir1)
	if err != nil {
		t.Fatalf("Failed to create first project")
	}

	prj2, err := Create("test2", dir2)
	if err != nil {
		t.Fatalf("Failed to create second project")
	}

	if prj1.instanceID == 0 {
		t.Errorf("Invalid project id for first project")
	}
	if prj1.instanceID == prj2.instanceID {
		t.Errorf("Two projects cannot have the same instanceID")
	}
}

func TestProjectWorkspace(t *testing.T) {

	dir, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir)

	prj, err := Create("test", dir)
	if err != nil {
		t.Fatalf("Failed to create project")
	}

	ws0Name := "Workspace 0"
	ws0Origin := "image0"
	ws1Name := "Workspace 1"
	ws1Origin := "image1"
	ws2Name := "Workspace 2"
	ws2Origin := "image2"
	ws3Name := "Workspace 3"
	ws3Origin := "image3"

	// default should have no workspaces
	if len(prj.Workspaces) != 0 {
		t.Fatalf("Invalid number of workspaces for new project")
	}

	// append: [Workspace 0]
	_, err = prj.CreateWorkspace(ws0Name, ws0Origin, "")
	if err != nil {
		t.Errorf("Failed to add Workspace 0")
	}
	if len(prj.Workspaces) != 1 {
		t.Errorf("Invalid number of workspaces after adding Workspace 0")
	}
	if prj.Workspaces[0].Name != ws0Name {
		t.Errorf("Workspace 0 should be at index 0: " + prj.Workspaces[0].Name)
	}

	// append: [ws0], [ws1]
	_, err = prj.CreateWorkspace(ws1Name, ws1Origin, "")
	if err != nil {
		t.Errorf("Failed to add Workspace 1")
	}
	if len(prj.Workspaces) != 2 {
		t.Errorf("Invalid number of workspaces after adding Workspace 1")
	}
	if prj.Workspaces[0].Name != ws0Name {
		t.Errorf("Workspace 0 should be at index 0: " + prj.Workspaces[0].Name)
	}
	if prj.Workspaces[1].Name != ws1Name {
		t.Errorf("Workspace 1 should be at index 1: " + prj.Workspaces[1].Name)
	}

	// insert: [ws0], [ws2], [ws1]
	_, err = prj.CreateWorkspace(ws2Name, ws2Origin, "Workspace 1")
	if err != nil {
		t.Errorf("Failed to add Workspace 2")
	}
	if len(prj.Workspaces) != 3 {
		t.Errorf("Invalid number of workspaces after adding Workspace3")
	}
	if prj.Workspaces[0].Name != ws0Name {
		t.Errorf("Workspace 0 should be at index 0: " + prj.Workspaces[0].Name)
	}
	if prj.Workspaces[1].Name != ws2Name {
		t.Errorf("Workspace 2 should be at index 1: " + prj.Workspaces[1].Name)
	}
	if prj.Workspaces[2].Name != ws1Name {
		t.Errorf("Workspace 1 should be at index 2: " + prj.Workspaces[2].Name)
	}

	// insert first: [ws3], [ws0], [ws2], [ws1]
	_, err = prj.CreateWorkspace(ws3Name, ws3Origin, "Workspace 0")
	if err != nil {
		t.Errorf("Failed to add Workspace 3")
	}
	if len(prj.Workspaces) != 4 {
		t.Errorf("Invalid number of workspaces after adding Workspace 3")
	}
	if prj.Workspaces[0].Name != ws3Name {
		t.Errorf("Workspace 3 should be at index 0: " + prj.Workspaces[0].Name)
	}
	if prj.Workspaces[1].Name != ws0Name {
		t.Errorf("Workspace 0 should be at index 1: " + prj.Workspaces[1].Name)
	}
	if prj.Workspaces[2].Name != ws2Name {
		t.Errorf("Workspace 2 should be at index 2: " + prj.Workspaces[2].Name)
	}
	if prj.Workspaces[3].Name != ws1Name {
		t.Errorf("Workspace 1 should be at index 3: " + prj.Workspaces[3].Name)
	}

	_, err = prj.CreateWorkspace(ws2Name, ws2Origin, "")
	if err == nil {
		t.Errorf("Should have failed to add workspace with same name")
	}

	// remove ws0 inside: [ws3], [ws0], [ws2], [ws1]
	if prj.DeleteWorkspace(ws0Name) != nil {
		t.Errorf("Failed to remove Workspace 0")
	}
	if len(prj.Workspaces) != 3 {
		t.Errorf("Invalid number of workspaces after removing Workspace 0")
	}

	// remove ws1 at the end: [ws3], [ws2], [ws1]
	if prj.DeleteWorkspace(ws1Name) != nil {
		t.Errorf("Failed to remove Workspace 1")
	}
	if len(prj.Workspaces) != 2 {
		t.Errorf("Invalid number of workspaces after removing Workspace 1")
	}
	// remove ws3 at the beginning: [ws3], [ws2]
	if prj.DeleteWorkspace(ws3Name) != nil {
		t.Errorf("Failed to remove Workspace 3")
	}
	if len(prj.Workspaces) != 1 {
		t.Errorf("Invalid number of workspaces after removing Workspace 3")
	}
	// remove last ws2: [ws2]
	if prj.DeleteWorkspace(ws2Name) != nil {
		t.Errorf("Failed to remove Workspace 2")
	}
	if len(prj.Workspaces) != 0 {
		t.Errorf("Invalid number of workspaces after removing Workspace 2")
	}

}
