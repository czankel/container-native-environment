package remote

import (
	"fmt"
	"time"

	"github.com/czankel/cne/errdefs"
)

type snapshot struct {
}

func (snap *snapshot) Name() string {
	fmt.Println("snapshot.Name")
	return ""
}

func (snap *snapshot) Parent() string {
	fmt.Println("snapshot.Parent")
	return ""
}

func (snap *snapshot) CreatedAt() time.Time {
	fmt.Println("snapshot.CreatedAt")
	return time.Now()
}

func (snap *snapshot) Size() (int64, error) {
	fmt.Println("snapshot.Size")
	return 0, errdefs.NotImplemented()
}

func (snap *snapshot) Inodes() (int64, error) {
	fmt.Println("snapshot.Inodes")
	return 0, errdefs.NotImplemented()
}
