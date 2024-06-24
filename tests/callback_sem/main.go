package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/jwijenbergh/purego"
	kinda "github.com/richinsley/kinda/pkg"
	pylib "github.com/richinsley/kindalib/pkg"
	sem "github.com/richinsley/kindalib/pkg/semaphore"
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
import ctypes
import sys

example_module.button_clicked("howdy!")

print("Hello from Python!")
print(example_module.semaphore_out)
print(example_module.semaphore_in)

# Get the semaphore handles from the example_module
semaphore_out_handle = example_module.semaphore_out
semaphore_in_handle = example_module.semaphore_in

# Signal the semaphore_out using platform-specific system calls
if sys.platform == "win32":
    # Windows
    ctypes.windll.kernel32.ReleaseSemaphore(semaphore_out_handle, 1, None)
elif sys.platform == "darwin":
    # macOS
    semaphore_ptr = ctypes.cast(semaphore_out_handle, ctypes.POINTER(ctypes.c_long))
    libsystem = ctypes.CDLL(None)
    print(ctypes.cast(semaphore_ptr, ctypes.c_void_p).value)
    libsystem.semaphore_signal(semaphore_ptr.contents.value)
else:
    # Unix-like systems (Linux, Android, etc.)
    libsystem = ctypes.CDLL(None)
    print(semaphore_out_handle)
    libsystem.sem_post(semaphore_out_handle)

# Wait for the semaphore_in to be signaled
if sys.platform == "win32":
    # Windows
    result = ctypes.windll.kernel32.WaitForSingleObject(semaphore_in_handle, 0xFFFFFFFF)
    if result == 0:
        print("Semaphore signaled!")
    else:
        print("Failed to wait for semaphore")
elif sys.platform == "darwin":
    # macOS
    semaphore_ptr = ctypes.cast(semaphore_in_handle, ctypes.POINTER(ctypes.c_long))
    libsystem = ctypes.CDLL(None)
    result = libsystem.semaphore_wait(semaphore_ptr.contents.value)
    if result == 0:
        print("Semaphore signaled!")
    else:
        print("Failed to wait for semaphore")
else:
    # Unix-like systems (Linux, Android, etc.)
    libsystem = ctypes.CDLL(None)
    result = libsystem.sem_wait(semaphore_in_handle)
    if result == 0:
        print("Semaphore signaled!")
    else:
        print("Failed to wait for semaphore")

example_module.button_clicked()
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

	// set the PyNone pointer
	// pynone is a global static PyObject* that is used to return None from C functions
	// returning just NULL is not enough, you need to return Py_None
	pynone = lib.GetPyNone()
	C.set_pynone(unsafe.Pointer(pynone))

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

	// create a semaphore out
	semaphore_out := sem.NewSemaphore()
	fmt.Printf("Created semaphore: %v\n", semaphore_out.GetHandle())

	// construct a python object from the semaphore handle
	semaphore_out_obj := lib.Invoke("PyLong_FromVoidPtr", semaphore_out.GetHandle())

	// add the semaphore object to the module
	lib.Invoke("PyModule_AddObject", module, lib.StrToPtr("semaphore_out"), semaphore_out_obj)

	// create a semaphore in
	semaphore_in := sem.NewSemaphore()
	fmt.Printf("Created semaphore: %v\n", semaphore_in.GetHandle())

	// construct a python object from the semaphore handle
	semaphore_in_obj := lib.Invoke("PyLong_FromVoidPtr", semaphore_in.GetHandle())

	// add the semaphore object to the module
	lib.Invoke("PyModule_AddObject", module, lib.StrToPtr("semaphore_in"), semaphore_in_obj)

	// Add the module to sys.modules
	sys_modules := lib.Invoke("PyImport_GetModuleDict")
	lib.Invoke("PyDict_SetItemString", sys_modules, lib.StrToPtr("example_module"), module)
	lib.Invoke("Py_DecRef", module)

	// Wait for the semaphore to be signaled
	go func() {
		fmt.Println("Waiting for semaphore...")
		semaphore_out.Wait()
		fmt.Println("Semaphore signaled!")

		// Signal the semaphore in
		semaphore_in.Post()
	}()

	// Run the Python code
	lib.Invoke("PyRun_SimpleString", lib.StrToPtr(quote_str))

	purego.UnrefCallbackFnPtr(&f)
}
