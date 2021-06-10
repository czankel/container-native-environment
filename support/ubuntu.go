package support

import (
	"github.com/czankel/cne/project"
)

const OSLayerCmdUser = "ubuntu-user"

// Note that the commands are never executed by CNE for security reasons
func UbuntuCreateOSLayer(ws *project.Workspace, atIndex int) error {

	osLayer, err := ws.CreateLayer(true /* system */, project.LayerTypeUbuntu, 0)
	if err != nil {
		return err
	}

	osLayer.Commands = []project.Command{{
		Name: "ubuntu-user",
		Args: []string{
			"adduser",
			"--system",
			"--home", "{{.User.HomeDir}}",
			"--shell", "{{.User.Shell}}",
			"--uid", "{{.User.UID}}",
			"--group",
			"{{.User.Username}}",
		}},
	}

	return nil
}
