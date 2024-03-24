package main

import (
	"fmt"
	"os"
	"path/filepath"

	kinda "github.com/richinsley/kinda/pkg"
	pylib "github.com/richinsley/kindalib/pkg"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		// Handle the error if there is one
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	// Specify the binary folder to place micromamba in
	rootDirectory := filepath.Join(cwd, "..", "micromamba")
	fmt.Println("Creating Kinda repo at: ", rootDirectory)
	version := "3.10"
	env, err := kinda.CreateEnvironment("myenv"+version, rootDirectory, version, "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}
	fmt.Printf("Created environment: %s\n", env.Name)

	// test create a library
	lib, err := pylib.NewPythonLib(env.PythonLibPath, env.EnvPath, env.SitePackagesPath, env.PythonVersion.MinorString())
	if err != nil {
		fmt.Printf("Error creating library: %v\n", err)
		return
	}
	fmt.Printf("Created library with : %d functions\n", lib.GetFTableCount())

	// Initialize Python interpreter
	os.Setenv("PYTHONHOME", env.EnvPath)
	os.Setenv("PYTHONPATH", env.EnvLibPath)
	lib.Invoke("Py_Initialize")

	// depricated since 3.11, but works to enable loading module from paths relative to the program
	// https://stackoverflow.com/questions/13422764/why-does-pyimport-import-fail-to-load-a-module-from-the-current-directory
	// actual configuration should be done with Python Configuratuion:
	// https://docs.python.org/3/c-api/init_config.html#init-python-config
	// unfortunately, we'll have to get involved with a per-version c structure
	lib.Invoke("PySys_SetArgv", 0, 0)

	// in the same folder as the executable, we'll have a python file "multiply.py".  We load it by it's
	// name "multiply".  We can do this becuase we called PySys_SetArgv.  The module name MUST be in a python string.
	// We create a python string with "PyUnicode_DecodeFSDefault"
	mpath := "multiply"
	sptr := pylib.StrToPtr(mpath)
	pName := lib.Invoke("PyUnicode_DecodeFSDefault", sptr)

	// use "PyImport_Import" to load the module
	pModule := lib.Invoke("PyImport_Import", pName)
	lib.Invoke("Py_DecRef", pModule)
	if pModule != 0 {
		// find the multiply function in the module
		funcStr := "multiply"
		fsptr := pylib.StrToPtr(funcStr)
		pFunc := lib.Invoke("PyObject_GetAttrString", pModule, fsptr)

		// make sure the returned function is callable
		if pFunc != 0 && lib.Invoke("PyCallable_Check", pFunc) != 0 {
			// create a python Tuple that can hold n values
			tupleCount := 2
			pArgs := lib.Invoke("PyTuple_New", uintptr(tupleCount))

			// fill the tuple with n values
			for i := 0; i < tupleCount; i++ {
				pValue := lib.Invoke("PyLong_FromLong", uintptr(i+1))
				if pValue == 0 {
					lib.Invoke("Py_DecRef", pArgs)
					lib.Invoke("Py_DecRef", pModule)
					fmt.Println("Cannot convert argument")
				}
				// pValue reference stolen here:
				lib.Invoke("PyTuple_SetItem", pArgs, uintptr(i), pValue)
			}

			// call the multiply function
			pValue := lib.Invoke("PyObject_CallObject", pFunc, pArgs)
			lib.Invoke("Py_DecRef", pArgs)
			if pValue != 0 {
				v := uint(lib.Invoke("PyLong_AsLong", pValue))
				fmt.Printf("Returned value %d\n", v)
			} else {
				lib.Invoke("Py_DecRef", pFunc)
				lib.Invoke("Py_DecRef", pModule)
				lib.Invoke("PyErr_Print")
				os.Exit(1)
			}
		} else {
			fmt.Println("Boo!")
		}
	} else {
		lib.Invoke("PyErr_Print")
	}
}
