package view

import (
	"fmt"
	"strings"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/filter"
)

// fsPath checks the remote and if it is the special local "remote" we just return the path instead of the remote:path
// notation.
func fsPath(remote string, path []string) string {
	if remote == Local {
		return fmt.Sprintf("%s", strings.Join(path, "/"))
	}

	return fmt.Sprintf("%s:%s", remote, strings.Join(path, "/"))
}

// fsPathFilename returns the path and filename from a given path. This is mainly used after the check if a path is a
// file so that we can remove the filename from the path for the rclone operations.
func fsPathFilename(path []string) ([]string, string) {
	return path[:len(path)-1], path[len(path)-1]
}

// CreateFilter returns a filter with the given min/max age and size.
func CreateFilter(minAge, maxAge, minSize, maxSize string) (*filter.Filter, error) {
	minAgeParsed, err := parseDuration(minAge)
	if err != nil {
		return nil, err
	}

	maxAgeParsed, err := parseDuration(maxAge)
	if err != nil {
		return nil, err
	}

	minSizeParsed, err := parseSize(minSize)
	if err != nil {
		return nil, err
	}

	maxSizeParsed, err := parseSize(maxSize)
	if err != nil {
		return nil, err
	}

	opts := filter.Opt{
		MinAge:  minAgeParsed,
		MaxAge:  maxAgeParsed,
		MinSize: minSizeParsed,
		MaxSize: maxSizeParsed,
	}

	return filter.NewFilter(&opts)
}

// parseDuration parses the given duration string and returns it in the fs.Duration format, which can be used in the
// filter options.
func parseDuration(duration string) (fs.Duration, error) {
	if duration == "off" {
		return fs.DurationOff, nil
	}

	parsed, err := fs.ParseDuration(duration)
	if err != nil {
		return fs.DurationOff, err
	}

	return fs.Duration(parsed), nil
}

// parseSize parses the given size and returns it in the fs.SizeSuffix format, which can be used in the filter options.
func parseSize(size string) (fs.SizeSuffix, error) {
	if size == "off" {
		return fs.SizeSuffix(-1), nil
	}

	sizeSuffix := fs.SizeSuffix(-1)
	err := sizeSuffix.Set(size)
	if err != nil {
		return sizeSuffix, err
	}

	return sizeSuffix, nil
}
