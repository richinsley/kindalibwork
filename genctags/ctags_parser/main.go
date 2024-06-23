package main

/*
	Intended to be run from a macos system.
	Install universal ctags:
	brew install universal-ctags
*/

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path"
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
	// we need two parameters, the min version and the max version and an optional path to the outputFolder
	if len(os.Args) < 3 {
		fmt.Println("Usage: passone <min_version> <max_version> [outputFolder]")
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		// Handle the error if there is one
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	min := os.Args[1]
	max := os.Args[2]
	minversion, err := kinda.ParseVersion(min)
	if err != nil {
		fmt.Printf("Invalid min version: %v\n", err)
		return
	}
	maxversion, err := kinda.ParseVersion(max)
	if err != nil {
		fmt.Printf("Invalid max version: %v\n", err)
		return
	}

	// get the current working directory if provided
	var outputFolder string
	if len(os.Args) > 3 {
		outputFolder = os.Args[3]
	} else {
		// where are we going to create the ctags folder sturcture
		outputFolder = path.Join(cwd, "..", "..", "pkg", "platform_ctags")
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

	// loop over range from minversion.Minor to maxversion.Minor
	for i := minversion.Minor; i <= maxversion.Minor; i++ {
		// create the environment
		envkey := fmt.Sprintf("3%d", i)
		envname := fmt.Sprintf("myenv%s", envkey)
		version := fmt.Sprintf("3.%d", i)
		envs[envkey], err = kinda.CreateEnvironment(envname, rootDirectory, version, "conda-forge", kinda.ShowVerbose)
		if err != nil {
			fmt.Printf("Error creating environment: %v\n", err)
			return
		}
		envs[envkey].PipInstallPackage("pycparser", "", "", false, kinda.ShowNothing)
	}

	// patch the pyport.h file for each environment
	for k, env := range envs {
		pyportPath := filepath.Join(env.PythonHeadersPath, "pyport.h")
		err = patchpyport(k, pyportPath)
		if err != nil {
			fmt.Printf("Error patching pyport.h: %v\n", err)
			return
		}
	}

	// target platforms
	platforms := []string{"darwin", "linux", "windows"}

	// where ctags universal is installed on macos
	ctag_path := "/opt/homebrew/bin/ctags"

	// process the ctags for each environment and each python version
	for _, platform := range platforms {
		for i := minversion.Minor; i <= maxversion.Minor; i++ {
			envkey := fmt.Sprintf("3%d", i)
			env := envs[envkey]
			fakeheaders := filepath.Join(pycparserFolder, "utils", "fake_libc_include")
			// "/Users/richardinsley/Projects/comfycli/kindalib/genctags/passone/micromamba/envs/myenv311/include/python3.11",
			// "/opt/homebrew/bin/ctags",
			// "/Users/richardinsley/Projects/comfycli/kindalib/genctags/passone/pycparser/utils/fake_libc_include",
			// "darwin",
			// "/Users/richardinsley/Projects/comfycli/kindalib/pkg/platform_ctags/darwin/ctags-311.json",
			fmt.Printf("Running ctags for python version %s on %s\n", envkey, platform)
			output, err := env.RunPythonReadCombined("nctags.py", env.PythonHeadersPath, ctag_path, fakeheaders, platform, filepath.Join(outputFolder, platform, fmt.Sprintf("ctags-%s.json", envkey)))
			if err != nil {
				fmt.Println(output)
				fmt.Printf("Error running nctags.py: %v\n", err)
				continue
			}
		}
	}
}
