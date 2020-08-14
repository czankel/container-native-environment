package project

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

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
	if !errors.Is(err, errdefs.ErrNotFound) {
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
func TestProjectCopyProjects(t *testing.T) {

	dir, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir)

	prj, err := Create("test", dir+"/test1")
	if err != nil {
		t.Fatalf("Failed to create new project: %v", err)
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

//
func TestProjectCreateTwoProjects(t *testing.T) {
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
	if prj.DeleteWorkspace("Workspace 0") != nil {
		t.Errorf("Failed to remove Workspace 0")
	}
	if len(prj.Workspaces) != 3 {
		t.Errorf("Invalid number of workspaces after removing Workspace 0")
	}

	// remove ws1 at the end: [ws3], [ws2], [ws1]
	if prj.DeleteWorkspace("Workspace 1") != nil {
		t.Errorf("Failed to remove Workspace 1")
	}
	if len(prj.Workspaces) != 2 {
		t.Errorf("Invalid number of workspaces after removing Workspace 1")
	}
	// remove ws3 at the beginning: [ws3], [ws2]
	if prj.DeleteWorkspace("Workspace 3") != nil {
		t.Errorf("Failed to remove Workspace 3")
	}
	if len(prj.Workspaces) != 1 {
		t.Errorf("Invalid number of workspaces after removing Workspace 3")
	}
	// remove last ws2: [ws2]
	if prj.DeleteWorkspace("Workspace 2") != nil {
		t.Errorf("Failed to remove Workspace 2")
	}
	if len(prj.Workspaces) != 0 {
		t.Errorf("Invalid number of workspaces after removing Workspace 2")
	}
}

func TestProjectWorkspaceNaming(t *testing.T) {
}

func TestProjectCurrentWorkspace(t *testing.T) {

	dir, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir)

	prj, err := Create("test", dir)
	if err != nil {
		t.Fatalf("Failed to create new project: %v", err)
	}

	cws, err := prj.CurrentWorkspace()
	if err == nil {
		t.Fatalf("CurrentWorkspace should return an error for new project")
	}
	err = prj.SetCurrentWorkspace("main")
	if err == nil {
		t.Fatalf("SetCurrentWorkspace should fail on new project")
	}

	ws1, err := prj.CreateWorkspace("Workspace1", "Image1", "")
	if err != nil {
		t.Fatalf("Inserting Workspace1 should have succeeded")
	}

	cws, err = prj.CurrentWorkspace()
	if err != nil {
		t.Fatalf("CurrentWorkspace should have passed")
	}
	if cws.Name != ws1.Name {
		t.Fatalf("CurrentWorkspace should return new workspace")
	}

	ws2, err := prj.CreateWorkspace("Workspace2", "Image2", "Workspace1")
	if err != nil {
		t.Fatalf("Inserting Workspace2 should have succeeded")
	}
	cws, err = prj.CurrentWorkspace()
	if err != nil {
		t.Fatalf("CurrentWorkspace should have passed")
	}
	if cws.Name != ws2.Name {
		t.Fatalf("CurrentWorkspace should have changed")
	}

	err = prj.SetCurrentWorkspace(ws1.Name)
	if err != nil {
		t.Fatalf("SetCurrentWorkspace failed")
	}
	cws, err = prj.CurrentWorkspace()
	if err != nil {
		t.Fatalf("CurrentWorkspace should have passed")
	}
	if cws.Name != ws1.Name {
		t.Fatalf("CurrentWorkspace should return the ")
	}

	_, err = prj.CreateWorkspace("Workspace3", "Image3", "Workspace2")
	if err != nil {
		t.Fatalf("Inserting Workspace2 should have succeeded")
	}
	cws, err = prj.CurrentWorkspace()
	if err != nil {
		t.Fatalf("CurrentWorkspace should have passed")
	}
	if cws.Name != ws1.Name {
		t.Fatalf("CurrentWorkspace should not have changed")
	}

}

func TestProjectLayers(t *testing.T) {

	dir, err := ioutil.TempDir("", testDir)
	if err != nil {
		t.Fatalf("Failed to create a temporary directory")
	}
	defer os.RemoveAll(dir)

	prj, err := Create("test", dir)
	if err != nil {
		t.Fatalf("Failed to create new project: %v", err)
	}

	ws, err := prj.CreateWorkspace("ws0", "image", "")
	if err != nil {
		t.Errorf("Failed to add Workspace 0")
	}

	err = ws.DeleteLayer("")
	if err == nil {
		t.Fatalf("DeleteLayer in empty workspace should fail")
	}

	layer := ws.TopLayer()
	if layer != nil {
		t.Fatalf("TopLayer should return nil for empty workspace")
	}

	layer1Name := "Layer1" // -> layer[1]
	layer2Name := "Layer2" // -> layer[2]
	layer3Name := "Layer3" // -> layer[0]

	_, err = ws.CreateLayer(layer1Name, -2)
	if err == nil {
		t.Fatalf("Inserting Layer1 at negative index should fail")
	}
	_, err = ws.CreateLayer(layer1Name, 1)
	if err == nil {
		t.Fatalf("Inserting Layer1 at invalid index should fail")
	}

	_, err = ws.CreateLayer(layer1Name, 0)
	if err != nil {
		t.Fatalf("Inserting Layer1 at 0 should succeed %v", err)
	}

	layer = ws.TopLayer()
	if layer.Name != layer1Name {
		t.Fatalf("TopLayer should be layer1")
	}

	_, err = ws.CreateLayer(layer2Name, 1)
	if err != nil {
		t.Fatalf("Appending layer at the end should succeed")
	}

	if len(ws.Environment.Layers) != 2 {
		t.Fatalf("Number of layers should be 2")
	}
	if ws.Environment.Layers[0].Name != layer1Name {
		t.Fatalf("Layer1 should be the first layer")
	}
	if ws.Environment.Layers[1].Name != layer2Name {
		t.Fatalf("Layer2 should be the second layer")
	}
	layer = ws.TopLayer()
	if layer.Name != layer2Name {
		t.Fatalf("TopLayer should be layer1")
	}

	_, err = ws.CreateLayer(layer3Name, 0)
	if err != nil {
		t.Fatalf("Inserting Layer3 at 0 should succeed")
	}
	if len(ws.Environment.Layers) != 3 {
		t.Fatalf("Number of layers should be 3")
	}
	if ws.Environment.Layers[0].Name != layer3Name {
		t.Fatalf("Layer3 should be the first layer")
	}
	if ws.Environment.Layers[1].Name != layer1Name {
		t.Fatalf("Layer1 should be the third layer")
	}
	if ws.Environment.Layers[2].Name != layer2Name {
		t.Fatalf("Layer2 should be the second layer")
	}
	layer = ws.TopLayer()
	if layer.Name != layer2Name {
		t.Fatalf("TopLayer should be layer1")
	}

	err = ws.DeleteLayer("invalid")
	if err == nil {
		t.Fatalf("DeleteLayer 'invalid' should have failed")
	}

	err = ws.DeleteLayer(layer1Name)
	if err != nil {
		t.Fatalf("DeleteLayer Layer1 should have succeeded")
	}
	if len(ws.Environment.Layers) != 2 {
		t.Fatalf("Number of layers should be 2")
	}
	if ws.Environment.Layers[0].Name != layer3Name {
		t.Fatalf("Layer3 should be the first layer")
	}
	if ws.Environment.Layers[1].Name != layer2Name {
		t.Fatalf("Layer2 should be the second layer")
	}

	err = ws.DeleteLayer(layer2Name)
	if err != nil {
		t.Fatalf("DeleteLayer Layer2 should have succeeded")
	}
	if len(ws.Environment.Layers) != 1 {
		t.Fatalf("Number of layers should be 1")
	}
	if ws.Environment.Layers[0].Name != layer3Name {
		t.Fatalf("Layer3 should be the only layer")
	}

	err = ws.DeleteLayer(layer3Name)
	if err != nil {
		t.Fatalf("DeleteLayer Layer3 should have succeeded")
	}
	if len(ws.Environment.Layers) != 0 {
		t.Fatalf("Number of layers should be 0")
	}
}
