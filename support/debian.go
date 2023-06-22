package support

import (
	"github.com/czankel/cne/project"
)

// Note that the commands are never executed by CNE for security reasons
func DebianCreateOSLayer(ws *project.Workspace, atIndex int) error {

	_, osLayer, err := ws.CreateLayer(project.LayerHandlerDebian, "")
	if err != nil {
		return err
	}

	osLayer.Handler = project.LayerHandlerDebian
	osLayer.Commands = []project.Command{{
		Name: "debian-user",
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
