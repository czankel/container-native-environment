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
		aptLayerCmdUpdate, []string{}, []string{"apt", "update"},
	}, {
		aptLayerCmdUpgrade,
		[]string{"DEBIAN_FRONTEND=noninteractive"},
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

func AptDeleteLayer(ws *project.Workspace) error {
	return ws.DeleteLayer(project.LayerTypeApt)
}

// helper function to return the command args and index
func getAptInstallCommand(aptLayer *project.Layer) (int, *project.Command) {

	for i := 0; i < len(aptLayer.Commands); i++ {
		c := &aptLayer.Commands[i]
		if c.Name == aptLayerCmdInstall {
			return i, c
		}
	}
	return -1, nil
}

// AptInstall attempts to install the specified app and adds it to the apt layer if successful.
func AptInstall(ws *project.Workspace, aptLayerIdx int, user config.User,
	ctr container.ContainerInterface,
	stream runtime.Stream, aptUpdate bool, aptNames []string) (int, error) {

	aptLayer := &ws.Environment.Layers[aptLayerIdx]

	_, cmds := getAptInstallCommand(aptLayer)
	if cmds == nil {
		aptInstall := append([]string{"apt", "install", "-y"}, aptNames...)

		aptLayer.Commands = append(aptLayer.Commands,
			project.Command{
				aptLayerCmdInstall,
				[]string{"DEBIAN_FRONTEND=noninteractive"},
				aptInstall})
		cmds = &aptLayer.Commands[len(aptLayer.Commands)-1]
	} else {
		n := aptNames
		for i := 3; i < len(cmds.Args); i++ {
			for j, apt := range n {
				if cmds.Args[i] == apt {
					aptNames = append(aptNames[:j], aptNames[j+1:]...)
					break
				}
			}
		}

		if len(aptNames) == 0 {
			return 0, nil
		}
		cmds.Args = append(cmds.Args, aptNames...)
	}

	// try to install the additional packages
	if aptUpdate {
		aptUpd := []string{"apt", "update"}
		code, err := ctr.BuildExec(&user, stream, aptUpd, []string{})
		if err != nil {
			return 0, err
		}
		if code != 0 {
			return int(code), nil
		}
	}

	args := append([]string{"apt", "install", "-y"}, aptNames...)
	code, err := ctr.BuildExec(&user, stream, args, []string{"DEBIAN_FRONTEND=noninteractive"})

	if err != nil {
		return 0, err
	}
	if code != 0 {
		return int(code), nil
	}

	ws.UpdateLayer(aptLayer)

	return 0, nil
}

// AptRemove removes a package from the container and layer.
// The implementation uses apt-purge instead of rebuilding the layer.
// It currently only removes packages that were added in this layer and are not part of the image
// or other layer.
// This function returns the return code of the apt command and any error
func AptRemove(ws *project.Workspace, aptLayerIdx int, user config.User,
	ctr container.ContainerInterface, stream runtime.Stream, aptNames []string) (int, error) {

	aptLayer := &ws.Environment.Layers[aptLayerIdx]

	cmdIdx, cmd := getAptInstallCommand(aptLayer)
	if cmd == nil {
		return 0, nil
	}

	if len(cmd.Args) < 4 && cmd.Args[0] != "apt" && cmd.Args[1] != "install" {
		return 0, errdefs.InternalError("malformed command group %v",
			aptLayerCmdInstall)
	}

	var delNames []string
	for _, a := range aptNames {
		for j := 3; j < len(cmd.Args); j++ {
			if a == cmd.Args[j] {
				delNames = append(delNames, a)
				cmd.Args = append(cmd.Args[:j], cmd.Args[j+1:]...)
			}
		}
	}

	if len(delNames) == 0 {
		return 0, nil
	}

	args := append([]string{"apt", "purge", "-y"}, delNames...)
	code, err := ctr.BuildExec(&user, stream, args, []string{"DEBIAN_FRONTEND=noninteractive"})
	if err != nil {
		return 0, err
	}
	if code != 0 {
		return int(code), nil
	}

	if len(delNames) > 0 {
		if len(cmd.Args) <= 3 {
			aptLayer.Commands = append(aptLayer.Commands[:cmdIdx],
				aptLayer.Commands[cmdIdx+1:]...)
		}

		ws.UpdateLayer(aptLayer)
	}

	return 0, nil
}
