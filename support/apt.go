package support

import (
	"github.com/czankel/cne/config"
	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

const (
	ImageLayerName = "Image"

	aptLayerCmdUpdate  = "apt-update"
	aptLayerCmdInstall = "apt-install"
	aptLayerCmdRemove  = "apt-remove"
)

type AptPackage struct {
	Name    string
	Version string
	Layer   string
}

func AptCreateLayers(ws *project.Workspace, atIndex int) error {

	_, aptLayer := ws.FindLayer(project.LayerTypeApt)
	if aptLayer != nil {
		return errdefs.AlreadyExists("Layer", project.LayerTypeApt)
	}

	aptLayer, err := ws.CreateLayer(true, project.LayerTypeApt, -1)
	if err != nil {
		return err
	}
	aptLayer.Commands = []project.CommandGroup{{
		aptLayerCmdUpdate,
		[][]string{{
			"apt", "update",
		}, {
			"{{if .Environment.Update == auto || " +
				".Environment.Update == manual && " +
				".Parameters.Upgrade in [apt, all]}}",
			"apt", "upgrade", "-y",
			"{{end}}",
		}},
	}}
	return nil
}

// AptInstall attempts to install the specified app and adds it to the apt layer if successful.
func AptInstall(ws *project.Workspace, aptLayerIdx int, user config.User, ctr *container.Container,
	stream runtime.Stream, aptUpdate bool, aptNames []string) (int, error) {

	aptLayer := &ws.Environment.Layers[aptLayerIdx]

	var cmdGrp *project.CommandGroup
	for i := 0; i < len(aptLayer.Commands); i++ {
		l := &aptLayer.Commands[i]
		if l.Name == aptLayerCmdInstall {
			cmdGrp = l
			break
		}
	}

	if cmdGrp == nil {
		cmds := append([]string{"apt", "install", "-y"}, aptNames...)
		aptLayer.Commands = append(aptLayer.Commands,
			project.CommandGroup{aptLayerCmdInstall, [][]string{cmds}})
		ws.UpdateLayer(aptLayer)
	} else {
		if len(cmdGrp.Cmdlines) > 1 {
			return 0, errdefs.InternalError("multiple lines in command group %s",
				aptLayerCmdInstall)
		}
		cmdline := cmdGrp.Cmdlines[0]
		if len(cmdline) < 4 && cmdline[0] != "apt" && cmdline[1] != "install" {
			return 0, errdefs.InternalError("malformed command group %v",
				aptLayerCmdInstall)
		}

		for i := 3; i < len(cmdline); i++ {
			for j, a := range aptNames {
				if cmdline[i] == a {
					aptNames = append(aptNames[:j], aptNames[j+1:]...)
				}
			}
		}
		if len(aptNames) > 0 {
			cmdGrp.Cmdlines[0] = append(cmdGrp.Cmdlines[0], aptNames...)
			ws.UpdateLayer(aptLayer)
		}
	}

	// try to install the additional packages

	// TODO: --------------------------------------------------
	// TODO: running as root inside the container during build!
	// TODO: --------------------------------------------------

	user.BuildUID = 0
	user.BuildGID = 0

	if aptUpdate {
		cmds := []string{"apt", "update"}
		code, err := ctr.BuildExec(&user, stream, cmds)
		if err != nil {
			return 0, err
		}
		if code != 0 {
			return int(code), nil
		}
	}

	cmds := append([]string{"apt", "install", "-y"}, aptNames...)
	code, err := ctr.BuildExec(&user, stream, cmds)
	if err != nil {
		return 0, err
	}
	if code != 0 {
		return int(code), nil
	}

	return 0, nil
}

// AptRemove
// TODO: implement apt remove
func AptRemove(ws *project.Workspace, aptNames []string) error {
	return nil
}
