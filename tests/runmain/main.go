package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	kinda "github.com/richinsley/kinda/pkg"
	pylib "github.com/richinsley/kindalib/pkg"
)

//go:embed quote_cli.py
var quote_str string

func main() {
	// Specify the binary folder to place micromamba in
	cwd, _ := os.Getwd()
	rootDirectory := filepath.Join(cwd, "..", "micromamba")
	fmt.Println("Creating Kinda repo at: ", rootDirectory)
	version := "3.10"
	env, err := kinda.CreateEnvironment("myenv"+version, rootDirectory, version, "conda-forge", kinda.ShowVerbose)
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}
	fmt.Printf("Created environment: %s\n", env.Name)

	// installing quote with pip
	err = env.PipInstallPackage("quote", "", "", true)
	if err != nil {
		fmt.Printf("Error installing quote package: %v\n", err)
		os.Exit(1)
	}

	// test create a library
	lib, err := pylib.NewPythonLib(env)
	if err != nil {
		fmt.Printf("Error creating library: %v\n", err)
		return
	}
	err = lib.Init("myprogram")
	if err != nil {
		fmt.Printf("Error initializing library: %v\n", err)
		return
	}
	fmt.Printf("Created library with : %d functions\n", lib.GetFTableCount())

	// run the quote_cli.py script
	qstr := pylib.StrToPtr(quote_str)
	status := lib.Invoke("PyRun_SimpleString", qstr)
	if status != 0 {
		fmt.Printf("Error running Py_Main: %d\n", status)
	}

	// using Py_Main seems to bonk any configuration we've done
	/*
		// create a slice of uintptrs to hold the arguments
		args := make([]uintptr, len(os.Args)+1)

		// the first argument is the program name
		args[0] = lib.Invoke("Py_DecodeLocale", pylib.StrToPtr(os.Args[0]), 0)

		// the second argument is the script name
		args[1] = lib.Invoke("Py_DecodeLocale", pylib.StrToPtr("quote_cli.py"), 0)

		// convert the remainder of the arguments to wchar_t*
		for i, arg := range os.Args[1:] {
			args[i+2] = lib.Invoke("Py_DecodeLocale", pylib.StrToPtr(arg), 0)
		}

		// we need to make the slice available to the C code as a pointer
		// we'll use the address of the first element
		argsPtr := uintptr(unsafe.Pointer(&args[0]))

		// call Py_Main
		status := lib.Invoke("Py_Main", uintptr(len(os.Args)+1), argsPtr)
		if status != 0 {
			fmt.Printf("Error running Py_Main: %d\n", status)
		}
	*/
}
