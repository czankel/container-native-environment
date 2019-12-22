package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/runtime"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List resources",
	Aliases: []string{"l"},
	Args:    cobra.MinimumNArgs(1),
}

var listRuntimeCmd = &cobra.Command{
	Use:     "runtimes",
	Aliases: []string{"runtime", "r"},
	Short:   "list runtimes",
	Args:    cobra.NoArgs,
	RunE:    listRuntimeRunE,
}

func listRuntimeRunE(cmd *cobra.Command, args []string) error {
	return nil
}

var listImageCmd = &cobra.Command{
	Use:     "images",
	Aliases: []string{"image", "i"},
	Short:   "list images",
	Args:    cobra.NoArgs,
	RunE:    listImageRunE,
}

const displayHashLength = 8

// sizeToSIString converts the provide integer value to a SI size string from the 10^3x exponent
func sizeToSIString(sz int64) string {
	const unit = 1000
	b := sz
	if b < 0 {
		b = -b
	}
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%cB", float64(sz)/float64(div), "kMGTPE"[exp])
}

// timeToAgoString converts the timespan from the provided time to the current time to a string
// in the formwat "T {year|month|hour}[s] ago". Future dates will return 'future'
func timeToAgoString(t time.Time) string {

	now := time.Now()
	if now.Before(t) {
		return "future"
	}

	diff := now.Sub(t)
	hours := diff.Hours()
	years := int(hours / 365 / 24)

	if years == 1 {
		return "one year ago"
	} else if years > 1 {
		return strconv.Itoa(years) + " years ago"
	}

	months := int(hours / 30.5)
	if months == 1 {
		return "one month ago"
	} else if months >= 1 {
		return strconv.Itoa(months) + " months ago"
	}

	if int(hours) == 1 {
		return "one hour ago"
	} else if hours > 1 {
		return strconv.Itoa(int(hours)) + " hours ago"
	}

	mins := diff.Minutes()
	if int(mins) == 1 {
		return "one minute ago"
	} else if mins > 1 {
		return strconv.Itoa(int(mins)) + " minutes ago"
	}

	return "seconds ago"
}

// splitRepoNameTag splits the provided full name to the image name and tag
// and resolves any respository aliases from the registered repositories.
// The default repository is omitted in the name.
func splitRepoNameTag(name string) (string, string) {

	conf := config.Load()

	tPos := strings.Index(name, ":")
	dispName := name[:tPos]

	for rn, r := range conf.Registry {
		p := r.Domain + "/" + r.RepoName
		if strings.HasPrefix(dispName, p) {
			dispName = name[len(p)+1 : tPos]
			if rn != config.DefaultRegistryName {
				dispName = rn + "/" + dispName
			}
		}
	}

	return dispName, name[tPos+1:]
}

func listImageRunE(cmd *cobra.Command, args []string) error {

	conf := config.Load()

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	images, err := run.Images()
	if err != nil {
		return err
	}

	imgList := make([]struct {
		Name      string
		Tag       string
		ID        string
		CreatedAt string
		Size      string
	}, len(images), len(images))

	for i, img := range images {
		name, tag := splitRepoNameTag(img.Name())
		imgList[i].Name = name
		imgList[i].Tag = tag
		digest := img.Digest().String()
		dPos := strings.Index(digest, ":")
		imgList[i].ID = digest[dPos+1 : dPos+1+displayHashLength]
		imgList[i].CreatedAt = timeToAgoString(img.CreatedAt())
		imgList[i].Size = sizeToSIString(img.Size())
	}
	printList(imgList)

	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listRuntimeCmd)
	listCmd.AddCommand(listImageCmd)
}
