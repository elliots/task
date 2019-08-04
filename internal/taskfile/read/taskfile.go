package read

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-task/task/v2/internal/taskfile"

	"gopkg.in/yaml.v2"
)

var (
	// ErrIncludedTaskfilesCantHaveIncludes is returned when a included Taskfile contains includes
	ErrIncludedTaskfilesCantHaveIncludes = errors.New("task: Included Taskfiles can't have includes. Please, move the include to the main Taskfile")

	// ErrNoTaskfileFound is returned when Taskfile.yml is not found
	ErrNoTaskfileFound = errors.New(`task: No Taskfile.yml found. Use "task --init" to create a new one`)
)

// Taskfile reads a Taskfile for a given directory
func Taskfile(dir string) (*taskfile.Taskfile, error) {
	path, found, err := searchForFile(dir, "Taskfile.yml")
	if !found {
		return nil, ErrNoTaskfileFound
	}
	if err != nil {
		return nil, err
	}

	t, err := readTaskfile(path)
	if err != nil {
		return nil, err
	}

	// Use the dir where we found the taskfile as the base
	// otherwise the includes won't be relative to it
	dir = filepath.Dir(path)

	for namespace, path := range t.Includes {
		path = filepath.Join(dir, path)
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			path = filepath.Join(path, "Taskfile.yml")
		}
		var includedTaskfile *taskfile.Taskfile

		if strings.HasSuffix(path, "package.json") {
			includedTaskfile, err = readPackageJson(path)
			if err != nil {
				return nil, err
			}
		} else {
			includedTaskfile, err = readTaskfile(path)
			if err != nil {
				return nil, err
			}
		}

		if len(includedTaskfile.Includes) > 0 {
			return nil, ErrIncludedTaskfilesCantHaveIncludes
		}

		for _, task := range includedTaskfile.Tasks {
			if task.Dir != "" {
				// If the dir was specified on the task then join this with the project root
				task.Dir = filepath.Join(dir, task.Dir)
			} else {
				// Otherwise ensure the task runs from the same directory as its taskfile
				task.Dir = filepath.Dir(path)
			}
		}

		if err = taskfile.Merge(t, includedTaskfile, namespace); err != nil {
			return nil, err
		}
	}

	path = filepath.Join(dir, fmt.Sprintf("Taskfile_%s.yml", runtime.GOOS))
	if _, err = os.Stat(path); err == nil {
		osTaskfile, err := readTaskfile(path)
		if err != nil {
			return nil, err
		}
		if err = taskfile.Merge(t, osTaskfile); err != nil {
			return nil, err
		}
	}

	for name, task := range t.Tasks {
		task.Task = name
	}

	return t, nil
}

func readTaskfile(file string) (*taskfile.Taskfile, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	var t taskfile.Taskfile
	return &t, yaml.NewDecoder(f).Decode(&t)
}

type packageJson struct {
	Scripts map[string]string `json:"scripts"`
}

func readPackageJson(file string) (*taskfile.Taskfile, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	fd, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var p packageJson

	if err = json.Unmarshal(fd, &p); err != nil {
		return nil, err
	}

	t := taskfile.Taskfile{
		Version: "2",
		Tasks:   taskfile.Tasks{},
	}

	for name, cmd := range p.Scripts {
		t.Tasks[name] = &taskfile.Task{
			Desc: fmt.Sprintf("→ %s", file),
			Cmds: []*taskfile.Cmd{
				&taskfile.Cmd{
					Cmd: "yarn",
				},
				&taskfile.Cmd{
					Cmd: "yarn " + cmd,
				},
			},
		}
	}

	return &t, nil
}
