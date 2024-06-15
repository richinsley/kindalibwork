package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	kinda "github.com/richinsley/kinda/pkg"
	pylib "github.com/richinsley/kindalib/pkg"
)

/*
#include <stdio.h>
#include <stdint.h>

static void * pynone = NULL;
void * button_clicked(void* p1, void* p2) {
	printf("Button clicked! (C function called)\n");
	return pynone;
}

void set_pynone(void* p) {
	pynone = p;
}

void * get_cb() {
	return button_clicked;
}
*/
import "C"

var quote_str string = `
import example_module
example_module.button_clicked()
`

func cb(p1 uintptr, p2 uintptr) uintptr {
	fmt.Println("Button clicked!")
	return 0
}

func init() {
	// Run main on the startup thread to satisfy the requirement
	// that Main runs on that thread.
	runtime.LockOSThread()
}

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

	// test create a library
	lib, err := pylib.NewPythonLib(env)
	if err != nil {
		fmt.Printf("Error creating library: %v\n", err)
		return
	}

	lib.Init("my_program")

	// set the PyNone pointer
	// pynone is a global static PyObject* that is used to return None from C functions
	// returning just NULL is not enough, you need to return Py_None
	pynone := lib.GetPyNone()
	C.set_pynone(pynone)

	// create the method object
	meth := lib.NewPyMethodDefArray(1)
	fmt.Printf("Created method object with size: %d\n", meth.PyConfig.Size)

	// get the C function pointer for the callback
	goCallback := C.get_cb()

	// set the buttonClickPtr to meth.SetMethodDef
	meth.SetMethodDef(0, "button_clicked", uintptr(goCallback), pylib.METH_NOARGS)

	// Create the moduledef object and create the module from that def
	moduledef := lib.NewPyModuleDef("example_module", "Example module with Go callback", &meth)

	module := lib.Invoke("PyModule_Create2", moduledef.GetBuffer(), 3)
	fmt.Printf("Created module: %v\n", module)

	// Add the module to sys.modules
	sys_modules := lib.Invoke("PyImport_GetModuleDict")
	lib.Invoke("PyDict_SetItemString", sys_modules, pylib.StrToPtr("example_module"), module)
	lib.Invoke("Py_DecRef", module)

	// Run the Python code
	lib.Invoke("PyRun_SimpleString", pylib.StrToPtr(quote_str))
}

// funcPC returns the function pointer of the given function
// This is implemented in assembly to avoid cgo
// func funcPC(f interface{}) uintptr
