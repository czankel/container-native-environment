package support

import (
	"github.com/czankel/cne/project"
)

// Note that the commands are never executed by CNE for security reasons
func UbuntuCreateOSLayer(ws *project.Workspace, atIndex int) error {

	_, osLayer, err := ws.CreateLayer(project.LayerHandlerUbuntu, "")
	if err != nil {
		return err
	}

	osLayer.Handler = project.LayerHandlerUbuntu
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
