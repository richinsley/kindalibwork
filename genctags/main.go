package main

/*
	Intended to be run from a macos system.
	Install universal ctags:
	brew install universal-ctags
*/

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	kinda "github.com/richinsley/kinda/pkg"
)

func patchpyport(version string, path string) error {
	// patch pyport.h to force define some macros
	// this will render the headers unusable for building python, but
	// it will allow us to generate the ctags json output with pycparser

	var appends string
	if version == "39" || version == "310" || version == "311" {
		appends = `
#define PyAPI_FUNC(RTYPE) RTYPE
#define PyAPI_DATA(RTYPE) RTYPE
#define PyMODINIT_FUNC PyObject*
#define _Py_NO_RETURN
#define Py_GCC_ATTRIBUTE(x)
#define Py_DEPRECATED(x)`
	} else {
		// 3.12 (and greater) requires a different patch
		appends = `
#define PyAPI_FUNC(RTYPE) RTYPE
#define Py_ALWAYS_INLINE
#define PyAPI_DATA(RTYPE) RTYPE
#define _Py_NO_RETURN
#define PyMODINIT_FUNC PyObject*
#define Py_GCC_ATTRIBUTE(x)
#define Py_DEPRECATED(x)
#define Py_UNUSED(x) x`
	}

	// read the file into memory
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// check if data is already patched
	if !bytes.Contains(data, []byte(appends)) {
		// split into lines and append the appends patch
		lines := strings.Split(string(data), "\n")
		lines = append(lines, appends)
		data = []byte(strings.Join(lines, "\n"))

		// write the data back to the file
		err = os.WriteFile(path, data, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		// Handle the error if there is one
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	// we need to clone the pycparser repo and point to the fake headers directory
	pycparserFolder := filepath.Join(cwd, "pycparser")
	git.PlainClone(pycparserFolder, false, &git.CloneOptions{
		URL:      "https://github.com/eliben/pycparser.git",
		Progress: os.Stdout,
	})

	// Specify the binary folder to place micromamba in
	rootDirectory := filepath.Join(cwd, "micromamba")
	fmt.Println("Creating Kinda repos at: ", rootDirectory)
	envs := make(map[string]*kinda.Environment)
	envs["39"], err = kinda.CreateEnvironment("myenv39", rootDirectory, "3.9", "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	envs["310"], err = kinda.CreateEnvironment("myenv310", rootDirectory, "3.10", "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	envs["311"], err = kinda.CreateEnvironment("myenv311", rootDirectory, "3.11", "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	envs["312"], err = kinda.CreateEnvironment("myenv312", rootDirectory, "3.12", "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	// we'll use the 39 env to generate the ctags json output for all the python versions
	envs["39"].PipInstallPackage("pycparser")

	// patch the pyport.h file for each environment
	for k, env := range envs {
		pyportPath := filepath.Join(env.PythonHeadersPath, "pyport.h")
		err = patchpyport(k, pyportPath)
		if err != nil {
			fmt.Printf("Error patching pyport.h: %v\n", err)
			return
		}
	}

	for k, env := range envs {
		output, err := envs["39"].RunPythonReadCombined("ctags.py", env.PythonHeadersPath, "/opt/homebrew/bin/ctags")
		if err != nil {
			fmt.Println(output)
			fmt.Printf("Error running ctags.py: %v\n", err)
			continue
		}

		// write the output to a file
		err = os.WriteFile(fmt.Sprintf("ctags-%s.json", k), []byte(output), 0644)
		if err != nil {
			fmt.Printf("Error writing ctags output: %v\n", err)
			continue
		}
		fmt.Println(k, output)
	}
}
