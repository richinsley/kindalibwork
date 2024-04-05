package main

/*
	Intended to be run from a macos system.
	Install universal ctags:
	brew install universal-ctags
*/

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-git/go-git/v5"
	cp "github.com/otiai10/copy"
	kinda "github.com/richinsley/kinda/pkg"
	kindalib "github.com/richinsley/kindalib/pkg"
)

//go:embed genmain.txt
var genmain string

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

func genPlatformGoFile(platformfolder string, platformpkgfolder, env string, structs *kindalib.PyCtags, environmnet *kinda.Environment) {
	// create the main file
	mainfilepath := path.Join(platformfolder, "main.go")
	// erase the file if it exists
	os.Remove(mainfilepath)
	file, err := os.Create(mainfilepath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	// write the package declaration
	file.WriteString("package main\n\n")

	// write the imports
	file.WriteString("import (\n")
	file.WriteString("\t\"encoding/json\"\n")
	file.WriteString("\t\"fmt\"\n")
	file.WriteString("\t\"os\"\n")
	file.WriteString("\t\"path\"\n")
	file.WriteString("\t\"runtime\"\n")

	file.WriteString(fmt.Sprintf("\tpkg \"github.com/richinsley/kindalib/genctags/passtwo/pkg/platformpy%s/pkg\"\n", env))
	file.WriteString("\tkindalib \"github.com/richinsley/kindalib/pkg\"\n")
	file.WriteString(")\n\n")

	// write the main function
	file.WriteString(genmain)

	// create the pkg folder
	err = os.MkdirAll(platformpkgfolder, 0755)
	if err != nil {
		fmt.Printf("Error creating output folder: %v\n", err)
		return
	}

	// create the platformpy.go file
	platformpyfilepath := path.Join(platformpkgfolder, "platformpy.go")
	// erase the file if it exists
	os.Remove(platformpyfilepath)
	platformpyfile, err := os.Create(platformpyfilepath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer platformpyfile.Close()

	platformpyfile.WriteString("package pkg\n\n")
	platformpyfile.WriteString("import (\n")
	platformpyfile.WriteString("\t\"fmt\"\n\n")
	platformpyfile.WriteString("\tkindalib \"github.com/richinsley/kindalib/pkg\"\n")
	platformpyfile.WriteString(")\n\n")
	platformpyfile.WriteString("/*\n")
	pyhpath := filepath.Join(environmnet.PythonHeadersPath, "Python.h")
	platformpyfile.WriteString(fmt.Sprintf("	#include <%s>\n", pyhpath))
	platformpyfile.WriteString("PyConfig * config = NULL;\n")
	platformpyfile.WriteString("void initStruct() { config = (PyConfig *)malloc(sizeof(PyConfig)); }\n")
	platformpyfile.WriteString("void deinitStruct() { free(config); }\n\n")

	for _, v := range structs.PyConfigs.PyConfig.Members {
		// int _config_init() { return (int)((void *)&config->_config_init - (void *)config);}
		platformpyfile.WriteString(fmt.Sprintf("int %s() { return (int)((void *)&config->%s - (void *)config);}\n", v.Name, v.Name))
	}
	platformpyfile.WriteString("*/\n")
	platformpyfile.WriteString("import \"C\"\n")
	platformpyfile.WriteString("func PopulateStructs(PyCtags *kindalib.PyCtags) {\n")
	platformpyfile.WriteString("\t\tC.initStruct()\n\n")
	platformpyfile.WriteString("\t\tfor i, v := range PyCtags.PyConfigs.PyConfig.Members {\n")
	platformpyfile.WriteString("\t\t\tswitch v.Name {\n")
	for _, v := range structs.PyConfigs.PyConfig.Members {
		platformpyfile.WriteString(fmt.Sprintf("\t\t\tcase \"%s\":\n", v.Name))
		platformpyfile.WriteString(fmt.Sprintf("\t\t\t\tPyCtags.PyConfigs.PyConfig.Members[i].Offset = int(C.%s())\n", v.Name))
	}
	platformpyfile.WriteString("\t\t\tdefault:\n")
	platformpyfile.WriteString("\t\t\t\tPyCtags.PyConfigs.PyConfig.Members[i].Offset = -1\n")
	platformpyfile.WriteString("\t\t\t\tfmt.Printf(\"Unknown member: %s\\n\", v.Name)\n")
	platformpyfile.WriteString("\t\t\t}\n")
	platformpyfile.WriteString("\t\t}\n")
	platformpyfile.WriteString("\t\tC.deinitStruct()\n")
	platformpyfile.WriteString("}\n")
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		// Handle the error if there is one
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	// where are we going to create the ctags folder sturcture
	outputFolder := path.Join(cwd, "..", "..", "pkg", "platform_ctags")

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
	envs["39"], err = kinda.CreateEnvironment("myenv39", rootDirectory, "3.9", "conda-forge", kinda.ShowVerbose)
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}
	envs["39"].PipInstallPackage("pycparser", "", "", false)

	envs["310"], err = kinda.CreateEnvironment("myenv310", rootDirectory, "3.10", "conda-forge", kinda.ShowVerbose)
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}
	envs["310"].PipInstallPackage("pycparser", "", "", false)

	envs["311"], err = kinda.CreateEnvironment("myenv311", rootDirectory, "3.11", "conda-forge", kinda.ShowVerbose)
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}
	envs["311"].PipInstallPackage("pycparser", "", "", false)

	envs["312"], err = kinda.CreateEnvironment("myenv312", rootDirectory, "3.12", "conda-forge", kinda.ShowVerbose)
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}
	envs["312"].PipInstallPackage("pycparser", "", "", false)

	// recursively copy (not rename) the entire micromamba folder to the second pass folder
	// this will allow us to run the second pass without having to recreate the environments
	secondPassFolder := filepath.Join(cwd, "..", "passtwo", "micromamba")
	err = os.RemoveAll(secondPassFolder)
	if err != nil {
		fmt.Printf("Error removing second pass folder: %v\n", err)
		return
	}
	err = cp.Copy(rootDirectory, secondPassFolder)
	if err != nil {
		fmt.Printf("Error copying micromamba folder: %v\n", err)
		return
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

	var platforms []string
	if len(os.Args) != 2 {
		platforms = []string{runtime.GOOS}
	} else {
		platforms = []string{os.Args[1]}
	}
	platform_ctags := make(map[string]*kindalib.PyCtags)

	for _, platform := range platforms {
		// create the output folder
		platformfolder := filepath.Join(outputFolder, platform)
		err = os.MkdirAll(platformfolder, 0755)
		if err != nil {
			fmt.Printf("Error creating output folder: %v\n", err)
			return
		}

		ctag_path := "ctags"
		if platform == "darwin" {
			// we want the brew specific universal-ctags
			ctag_path = "/opt/homebrew/bin/ctags"
		} else if platform == "windows" {
			// for now, we'll generate ctags for windows on macos
			ctag_path = "/opt/homebrew/bin/ctags"
		}

		for k, env := range envs {
			fakeheaders := filepath.Join(pycparserFolder, "utils", "fake_libc_include")
			output, err := env.RunPythonReadCombined("ctags.py", env.PythonHeadersPath, ctag_path, fakeheaders)
			if err != nil {
				fmt.Println(output)
				fmt.Printf("Error running ctags.py: %v\n", err)
				continue
			}

			// write the output to a file in the platform folder
			err = os.WriteFile(filepath.Join(platformfolder, fmt.Sprintf("ctags-%s.json", k)), []byte(output), 0644)
			if err != nil {
				fmt.Printf("Error writing ctags output: %v\n", err)
				continue
			}

			// create and store the PyCtags struct
			platform_ctags[k] = &kindalib.PyCtags{}
			err = json.Unmarshal([]byte(output), platform_ctags[k])
			if err != nil {
				fmt.Printf("Error unmarshalling json: %v\n", err)
				continue
			}
		}
	}

	// generate the go files for the second pass
	for k, e := range envs {
		goplatformFolder := path.Join(cwd, "..", "passtwo", "pkg", fmt.Sprintf("platformpy%s", k))
		goplatformPKGFolder := path.Join(cwd, "..", "passtwo", "pkg", fmt.Sprintf("platformpy%s", k), "pkg")
		// erase the folder if it exists
		os.RemoveAll(goplatformFolder)
		err = os.MkdirAll(goplatformFolder, 0755)
		err = os.MkdirAll(goplatformPKGFolder, 0755)
		genPlatformGoFile(goplatformFolder, goplatformPKGFolder, k, platform_ctags[k], e)
	}
}
