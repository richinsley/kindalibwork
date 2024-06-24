package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jwijenbergh/purego"
	kinda "github.com/richinsley/kinda/pkg"
	pylib "github.com/richinsley/kindalib/pkg"
)

var quote_str string = `
import tkinter as tk
import example_module

def button_click():
    example_module.button_clicked()
    print("Button clicked! (Python function called)")

window = tk.Tk()
window.title("Tkinter Example with C++ Callback")

button = tk.Button(window, text="Click Me", command=button_click)
button.pack()

window.mainloop()
`
var pynone uintptr

func button_cb(p1 uintptr, p2 uintptr) uintptr {
	fmt.Println("Button clicked!")
	return pynone
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

	// pylong tests
	pylong := lib.Invoke("PyLong_FromLong", 123)
	rval := int(lib.Invoke("PyLong_AsLong", pylong))
	fmt.Printf("PyLong_AsLong: %d\n", rval)
	lib.Invoke("Py_DecRef", pylong)

	// set the PyNone pointer
	// pynone is a global static PyObject* that is used to return None from C functions
	// returning just NULL is not enough, you need to return Py_None
	pynone = lib.GetPyNone()

	// create the method object
	meth := lib.NewPyMethodDefArray(1)
	fmt.Printf("Created method object with size: %d\n", meth.PyConfig.Size)

	// create a callback function
	var f func(uintptr, uintptr) uintptr = button_cb
	goCallback := purego.NewCallbackFnPtr(&f)

	// set the buttonClickPtr to meth.SetMethodDef
	meth.SetMethodDef(0, "button_clicked", uintptr(goCallback), pylib.METH_VARARGS)

	// Create the moduledef object and create the module from that def
	moduledef := lib.NewPyModuleDef("example_module", "Example module with Go callback", &meth)

	module := lib.Invoke("PyModule_Create2", moduledef.GetBuffer(), 3)
	fmt.Printf("Created module: %v\n", module)

	// Add the module to sys.modules
	sys_modules := lib.Invoke("PyImport_GetModuleDict")
	lib.Invoke("PyDict_SetItemString", sys_modules, lib.StrToPtr("example_module"), module)
	lib.Invoke("Py_DecRef", module)

	// Run the Python code
	lib.Invoke("PyRun_SimpleString", lib.StrToPtr(quote_str))

	purego.UnrefCallbackFnPtr(&f)

	select {}
}
