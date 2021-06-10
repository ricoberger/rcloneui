package view

import (
	"fmt"
	"strings"
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
