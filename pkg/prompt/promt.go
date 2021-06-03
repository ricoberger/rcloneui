package prompt

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
)

var (
	actions = []string{"Copy", "Delete", "Cancel"}
)

func SelectRemote(remotes []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "→  {{ . | blue }}",
		Inactive: "   {{ . | blue }}",
		Selected: "   {{ . | red }}",
	}

	searcher := func(input string, index int) bool {
		s := remotes[index]
		name := strings.Replace(strings.ToLower(s), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     "Remotes",
		Items:     remotes,
		Templates: templates,
		Size:      10,
		Searcher:  searcher,
		Stdout:    &bellSkipper{},
	}

	i, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return remotes[i], nil
}

func SelectEntry(remote string, path []string, entries []Entry) (string, error) {
	funcMap := promptui.FuncMap
	funcMap["format"] = func(entry Entry) string {
		if entry.Description == ".." {
			return ".."
		}

		return fmt.Sprintf("%d %s %s", entry.Size, entry.Time, entry.Description)
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "→  {{ format . | blue }}",
		Inactive: "   {{ format . | blue }}",
		Selected: "   {{ format . | red }}",
		FuncMap:  funcMap,
	}

	searcher := func(input string, index int) bool {
		e := entries[index]
		name := strings.Replace(strings.ToLower(e.Description), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     fmt.Sprintf("%s:%s (%d)", remote, strings.Join(path, "/"), len(entries)-1),
		Items:     entries,
		Templates: templates,
		Size:      10,
		Searcher:  searcher,
		Stdout:    &bellSkipper{},
	}

	i, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return entries[i].Remote, nil
}

func SelectFileAction(remote string, path []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "→  {{ . | blue }}",
		Inactive: "   {{ . | blue }}",
		Selected: "   {{ . | red }}",
	}

	prompt := promptui.Select{
		Label:     fmt.Sprintf("%s:%s", remote, strings.Join(path, "/")),
		Items:     actions,
		Templates: templates,
		Size:      len(actions),
		Stdout:    &bellSkipper{},
	}

	i, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return actions[i], nil
}
