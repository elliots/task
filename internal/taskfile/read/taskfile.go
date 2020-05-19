package read

import (
	"bufio"
	"bytes"
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

	// Store the absolute path to the project root as an environment variable
	absdir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for directory %s: %s", dir, err)
	}
	if t.Vars == nil {
		t.Vars = taskfile.Vars{}
	}
	t.Vars["PROJECT_ROOT"] = taskfile.Var{
		Static: absdir,
	}

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
			includedTaskfile, err = readPackageJson(absdir, path)
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

func readPackageJson(projectRoot, file string) (*taskfile.Taskfile, error) {
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

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	relFile, err := filepath.Rel(wd, file)
	if err != nil {
		relFile = file
	}

	for name := range p.Scripts {
		t.Tasks[name] = &taskfile.Task{
			Desc: fmt.Sprintf("→ %s%s", relFile, findLineNumber(fd, name)),
			Cmds: []*taskfile.Cmd{
				&taskfile.Cmd{
					Cmd: "pnpm run " + name,
				},
			},
		}
	}

	return &t, nil
}

func findLineNumber(f []byte, scriptName string) string {
	// Splits on newlines by default.
	scanner := bufio.NewScanner(bytes.NewReader(f))

	line := 1
	// https://golang.org/pkg/bufio/#Scanner.Scan
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), `"`+scriptName+`":`) {
			return fmt.Sprintf(":%d", line)
		}

		line++
	}

	if err := scanner.Err(); err != nil {
		panic(err) // Probably shouldn't be possible?
	}

	return ""
}
