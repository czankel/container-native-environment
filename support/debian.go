package support

import (
	"github.com/czankel/cne/project"
)

// Note that the commands are never executed by CNE for security reasons
func DebianOSLayerInit(layer *project.Layer) error {

	layer.Commands = []project.Command{{
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
