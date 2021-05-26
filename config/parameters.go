package config

const (
	UpgradeNone  = ""
	UpgradeImage = "image"
	UpgradeApt   = "apt"
)

type Parameters struct {
	Upgrade string // upgrade the listed components during container rebuilt
}
