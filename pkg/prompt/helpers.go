package prompt

import (
	"context"

	"github.com/rclone/rclone/fs"
)

type Entry struct {
	Description string
	Remote      string
	Size        int64
	Time        string
}

func ConvertEntries(entries fs.DirEntries) []Entry {
	convertedEntries := []Entry{{Description: "..", Remote: ".."}}

	for _, entry := range entries {
		convertedEntries = append(convertedEntries, Entry{
			Description: entry.String(),
			Remote:      entry.Remote(),
			Size:        entry.Size(),
			Time:        entry.ModTime(context.Background()).Format("2006-01-02 15:04:05"),
		})
	}

	return convertedEntries
}
