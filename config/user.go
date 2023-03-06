package config

import (
	"os"
	"os/user"
	"strconv"
)

// The User is technically its own first-class object similar to
// Project and Config. However, since the configuration can define
// the User, it includes a function for returning the User
// credentials rather than being part of it.

const defaultShell = "/bin/bash"

type User struct {
	Username  string
	Groupname string
	Shell     string
	IsSudo    bool
	IsSuid    bool
	UID       uint32
	EUID      uint32
	GID       uint32
	HomeDir   string
	Pwd       string
	BuildUID  uint32
	BuildGID  uint32
}

// CurrentUser returns information about the current user.
// Note that for IsSudo, the CNE binary must run as root and SUDO_ envvars must be set.
func CurrentUser() (User, error) {

	usr, err := user.Current()
	if err != nil {
		return User{}, err
	}

	euid := os.Geteuid()
	uid := os.Getuid()
	gid := os.Getgid()

	isSuid := uid != euid
	isSudo := false
	username := usr.Username
	sudo_user := os.Getenv("SUDO_USER")
	if uid == 0 && sudo_user != "" {
		isSudo = true
		username = sudo_user
		uid, _ = strconv.Atoi(os.Getenv("SUDO_UID"))
		gid, _ = strconv.Atoi(os.Getenv("SUDO_GID"))
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = defaultShell
	}

	groupname := username
	group, err := user.LookupGroupId(strconv.Itoa(gid))
	if err != nil {
		groupname = group.Name
	}

	pwd, err := os.Getwd()
	if err != nil {
		pwd = usr.HomeDir
	}

	user := User{
		Username:  username,
		Groupname: groupname,
		IsSudo:    isSudo,
		IsSuid:    isSuid,
		HomeDir:   usr.HomeDir,
		Pwd:       pwd,
		Shell:     shell,
		UID:       uint32(uid),
		GID:       uint32(gid),
		EUID:      uint32(euid),

		// TODO: --------------------------------------------------
		// TODO: running as root inside the container during build!
		// TODO: --------------------------------------------------

		BuildUID: 0,
		BuildGID: 0,
	}

	return user, nil
}

/*
// DeploymentUser returns information about a deployment user....
func DeploymentUser() (User, error) {

	// Defaults:
	//  Username: 	ProjectName
	//  Groupname:	Username
	//  Homedir: 	none? or /home/...
	//  Pwd:	none? or /Projectname?
	//  Shell:      bash ??
	//  UID:        101
	//  GID:        101
	// FIXME: logging??

	user := User{
		Username:  prj.Deployment.Username,
		Groupname: prj.Deployment.Groupname,
		IsSudo:    false,
		IsSuid:    false,
		HomeDir:   prj.Homedir,
		Pwd:       prj.Deployment.Pwd,
		Shell:     prj.Deployment.Shell,
		UID:       prj.Deployment.UID,
		GID:       prj.Deployment.GID,
		BuildID:   0,
		BuildGID:  0,
	}

	return user, nil
}
*/
