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
	aptLayerCmdUpgrade = "apt-upgrade"
	aptLayerCmdInstall = "apt-install"
	aptLayerCmdRemove  = "apt-remove"
)

type AptPackage struct {
	Name    string
	Version string
	Layer   string
}

func AptCreateLayer(ws *project.Workspace, atIndex int) error {

	_, aptLayer := ws.FindLayer(project.LayerTypeApt)
	if aptLayer != nil {
		return errdefs.AlreadyExists("Layer", project.LayerTypeApt)
	}

	aptLayer, err := ws.CreateLayer(true, project.LayerTypeApt, -1)
	if err != nil {
		return err
	}

	aptLayer.Commands = []project.Command{{
		aptLayerCmdUpdate,
		[]string{"apt", "update"},
	}, {
		aptLayerCmdUpgrade,
		[]string{
			"{{if .Environment.Update == auto || " +
				".Environment.Update == manual && " +
				".Parameters.Upgrade in [apt, all]}}",
			"apt", "upgrade", "-y",
			"{{end}}",
		},
	}}
	return nil
}

// AptInstall attempts to install the specified app and adds it to the apt layer if successful.
func AptInstall(ws *project.Workspace, aptLayerIdx int, user config.User, ctr *container.Container,
	stream runtime.Stream, aptUpdate bool, aptNames []string) (int, error) {

	aptLayer := &ws.Environment.Layers[aptLayerIdx]

	var cmds *project.Command
	for i := 0; i < len(aptLayer.Commands); i++ {
		c := &aptLayer.Commands[i]
		if c.Name == aptLayerCmdInstall {
			cmds = c
			break
		}
	}

	if cmds == nil {
		aptInstall := append([]string{"apt", "install", "-y"}, aptNames...)
		aptLayer.Commands = append(aptLayer.Commands,
			project.Command{aptLayerCmdInstall, aptInstall})
		cmds = &aptLayer.Commands[len(aptLayer.Commands)-1]
		ws.UpdateLayer(aptLayer)
	} else {
		if len(cmds.Args) < 4 && cmds.Args[0] != "apt" && cmds.Args[1] != "install" {
			return 0, errdefs.InternalError("malformed apt install command %v",
				aptLayerCmdInstall)
		}

		for i := 3; i < len(cmds.Args); i++ {
			for j, a := range aptNames {
				if cmds.Args[i] == a {
					aptNames = append(aptNames[:j], aptNames[j+1:]...)
				}
			}
		}
		if len(aptNames) > 0 {
			cmds.Args = append(cmds.Args, aptNames...)
			ws.UpdateLayer(aptLayer)
		}
	}

	// try to install the additional packages
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

	args := append([]string{"apt", "install", "-y"}, aptNames...)
	code, err := ctr.BuildExec(&user, stream, args)
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
