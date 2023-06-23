package support

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

type testCase struct {
	description string
	// arguments
	install   bool
	aptUpdate bool // For AptInstall
	apts      []string
	// return values from then injected BuildExec
	code uint32
	err  error
	// expected results
	isError       bool
	isCodeNotZero bool
	opApts        []string
	installedApts []string
}

type testContainer struct {
	runtime.Container
	testCase *testCase
	cmdlines [][]string // BuildExec arguments
}

type testProcess struct {
	runtime.Process
	ctr *testContainer
}

func (ctr *testContainer) Exec(ctx context.Context, stream runtime.Stream,
	procSpec *runtime.ProcessSpec) (runtime.Process, error) {
	ctr.cmdlines = append(ctr.cmdlines, procSpec.Args)
	return &testProcess{ctr: ctr}, nil
}

func (proc *testProcess) Signal(ctx context.Context, sig os.Signal) error {
	return nil
}

func (proc *testProcess) Wait(ctx context.Context) (<-chan runtime.ExitStatus, error) {
	ctr := proc.ctr
	stat := make(chan runtime.ExitStatus)
	go func() {
		defer close(stat)
		err := ctr.testCase.err
		code := ctr.testCase.code
		if len(ctr.cmdlines) == 1 {
			for _, s := range ctr.cmdlines[0] {
				if s == "update" {
					err = nil
					code = 0
					break
				}
			}
		}
		stat <- runtime.ExitStatus{time.Now(), err, code}
	}()
	return stat, nil
}

func setupProject(t *testing.T) (*project.Project, *project.Workspace) {

	prjName := "project"
	prjOrigin := "image"
	wsName := ""
	nextWs := ""

	prj := &project.Project{
		Name: prjName,
	}
	ws, err := prj.CreateWorkspace(wsName, prjOrigin, nextWs)
	if err != nil {
		t.Fatalf("Failed to create workspace")
	}

	return prj, ws
}

func TestSupportApt(t *testing.T) {

	user, err := config.CurrentUser()
	if err != nil {
		t.Fatalf("Failed to get current user")
	}

	_, ws := setupProject(t)

	testCases := [][]testCase{{
		{"install a with update", true, true, []string{"a"}, 0, nil,
			false, false, []string{"a"}, []string{"a"}},
		{"install b and c", true, false, []string{"b", "c"}, 0, nil,
			false, false, []string{"b", "c"}, []string{"a", "b", "c"}},
		{"install a again", true, false, []string{"a"}, 0, nil,
			false, false, []string{}, []string{"a", "b", "c"}},
		{"install d and again c", true, false, []string{"d", "c"}, 0, nil,
			false, false, []string{"d"}, []string{"a", "b", "c", "d"}},
		{"remove c and a", false, false, []string{"c", "a"}, 0, nil,
			false, false, []string{"c", "a"}, []string{"b", "d"}},
		{"remove c again", false, false, []string{"c"}, 0, nil,
			false, false, []string{}, []string{"b", "d"}},
		{"remove b", false, false, []string{"b"}, 0, nil,
			false, false, []string{"b"}, []string{"d"}},
		{"remove a again", false, false, []string{"a"}, 0, nil,
			false, false, []string{}, []string{"d"}},
		{"remove d", false, false, []string{"d"}, 0, nil,
			false, false, []string{}, []string{}},
		{"remove d again", false, false, []string{"d"}, 0, nil,
			false, false, []string{}, []string{}},
	}, {
		{"install unknown package", true, false, []string{"a"}, 100, nil,
			false, true, []string{}, []string{}},
	}, {
		{"error in update", true, true, []string{"a"}, 1, errdefs.ErrInternalError,
			true, false, []string{}, []string{}},
	}, {
		{"error in install", true, true, []string{"x"}, 1, errdefs.ErrInternalError,
			true, false, []string{}, []string{}},
	}, {
		{"install a with update", true, true, []string{"x"}, 0, nil,
			false, false, []string{"x"}, []string{"x"}},
		{"error in remove", false, false, []string{"x"}, 1, errdefs.ErrInternalError,
			true, false, []string{}, []string{}},
	}}

	stream := runtime.Stream{}
	ctr := &testContainer{}
	aptLayerIdx := 0

	ctx := context.Background()

	for _, testGroup := range testCases {

		_, aptLayer, err := ws.CreateLayer("apt-test", "")
		if err != nil {
			t.Fatalf("Failed to create Apt layer")
		}

		err = AptLayerInit(aptLayer)
		if err != nil {
			t.Fatalf("Failed to create Apt layers")
		}

		for _, test := range testGroup {

			ctr.cmdlines = [][]string{}
			ctr.testCase = &test

			var code int
			if test.install {
				code, err = AptInstall(ctx, ws, aptLayerIdx, user, ctr, stream,
					test.aptUpdate, test.apts)
			} else {
				code, err = AptRemove(ctx, ws, aptLayerIdx, user, ctr, stream, test.apts)
			}

			if err != nil && test.isError {
				continue
			} else if err != nil && !test.isError {
				t.Errorf("test '%s' failed: AptInstall/AptRemove failed",
					test.description)
				continue
			} else if err == nil && test.isError {
				t.Errorf("test '%s' failed: Expected AptInstall/AptRemove to fail",
					test.description)
				continue
			}

			if code != 0 && test.isCodeNotZero {
				continue
			} else if code != 0 && !test.isCodeNotZero {
				t.Errorf("test '%s' failed: AptInstall/AptRemove failed with code",
					test.description)
				continue
			} else if code == 0 && test.isCodeNotZero {
				t.Errorf("test '%s' failed: Expected AptInstall/AptRemove to fail with code",
					test.description)
				continue
			}

			// validate executed commands
			var args []string
			if test.aptUpdate {
				if len(ctr.cmdlines) != 2 {
					t.Errorf("test '%s' failed: update enabled but not executed",
						test.description)
				} else {
					args = ctr.cmdlines[1]
				}
			} else if len(ctr.cmdlines) > 1 {
				t.Errorf("test '%s' failed: multiple commands %v",
					test.description, ctr.cmdlines)
			} else if len(ctr.cmdlines) > 0 {
				args = ctr.cmdlines[0]
			}

			// validate layer commands
			if len(test.opApts) > 0 && len(args) < 3 {
				t.Errorf("test '%s' failed: invalid install/remove command %v",
					test.description, args)
				continue
			}
			if len(test.opApts) > 0 && len(args) != len(test.opApts)+3 {
				t.Errorf("test '%s' failed: expected packages not installed or "+
					"removed: expected %v vs %v",
					test.description, test.opApts, args[3:])
				continue
			}
			for i, a := range test.opApts {
				if a != args[i+3] {
					t.Errorf("test '%s' failed: expected to install/remove "+
						"apt '%v' vs '%v' at index %d",
						test.description, a, args[i+3], i)
				}
			}

			_, cmd := getAptInstallCommand(aptLayer)
			if cmd == nil {
				if len(test.installedApts) != 0 {
					t.Errorf("test '%s' failed: expected apts to be installed vs 0",
						test.description)
				}
				continue
			}
			if len(test.installedApts) == 0 {
				t.Errorf("test '%s' failed: expected no apts to be installed vs 0",
					test.description)
				continue
			}
			if len(cmd.Args) <= 3 {
				t.Errorf("test '%s' failed: invalid command args in layer: %v",
					test.description, cmd.Args)
				continue
			}
			if len(cmd.Args) != len(test.installedApts)+3 {
				t.Errorf("test '%s' failed: installed apts mismatch: "+
					"expected %v vs %v", test.description,
					test.installedApts, cmd.Args[3:])
				continue
			}
			for i, a := range test.installedApts {
				if a != cmd.Args[i+3] {
					t.Errorf("test '%s' failed:failed: "+
						"expected intsalled apt '%v' vs '%v' at index %d",
						test.description, a, cmd.Args[i+3], i)
				}
			}
		}
		err = ws.DeleteLayer("apt-test")
		if err != nil {
			t.Fatalf("failed to delete test layer for apt")
		}
	}
}
