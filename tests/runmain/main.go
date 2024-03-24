package main

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	kinda "github.com/richinsley/kinda/pkg"
	pylib "github.com/richinsley/kindalib/pkg"
)

// #cgo LDFLAGS: -L/Users/richardinsley/Projects/comfycli/kindalib/tests/micromamba/envs/myenv3.10/lib -lpython3.10 -Wl,-rpath,/Users/richardinsley/Projects/comfycli/kindalib/tests/micromamba/envs/myenv3.10/lib
// #cgo CFLAGS: -I../micromamba/envs/myenv3.10/include/python3.10
/*
#include <Python.h>
#include <stdio.h>
#include <unistd.h> // For sleep() and pause()
#include <signal.h> // For signal handling

int keepRunning = 1;
void sigintHandler(int dummy) {
    keepRunning = 0;
}


// /Users/richardinsley/Projects/comfycli/kindalib/tests/micromamba/envs/myenv3.10
uint8_t * buffer = NULL;

void * get_buffer() {
	return buffer;
}

void * get_home() {
	PyConfig* config = (PyConfig*)buffer;
	return (void *)&config->home;
}

int test2() {
	// Register the SIGINT handler (Ctrl+C)
    signal(SIGINT, sigintHandler);

	// dump some runtime sizes
	printf("Struct size :%d\n", sizeof(PyConfig));
	printf("PyWideStringList size :%d\n", sizeof(PyWideStringList));
	printf("Py_ssize_t size :%d\n", sizeof(Py_ssize_t));
	printf("int size :%d\n", sizeof(int));
	printf("unsigned long size :%d\n", sizeof(unsigned long));


	// PyConfig config;
	buffer = malloc(sizeof(PyConfig));
	PyConfig* config = (PyConfig*)buffer;
	PyConfig* pconfig = config;

	uint64_t pos = (uint64_t)pconfig;

	uint64_t _config_init = (uint64_t)&pconfig->_config_init - pos;
	printf("_config_init_offset :%d\n", _config_init);

	uint64_t isolated = (uint64_t)&pconfig->isolated - pos;
	printf("isolated :%d\n", isolated);

	uint64_t use_environment = (uint64_t)&pconfig->use_environment - pos;
	printf("use_environment :%d\n", use_environment);

	uint64_t dev_mode = (uint64_t)&pconfig->dev_mode - pos;
	printf("dev_mode :%d\n", dev_mode);

	uint64_t hash_seed = (uint64_t)&pconfig->hash_seed - pos;
	printf("hash_seed :%d\n", hash_seed);

	uint64_t faulthandler = (uint64_t)&pconfig->faulthandler - pos;
	printf("faulthandler :%d\n", faulthandler);

	uint64_t tracemalloc = (uint64_t)&pconfig->tracemalloc - pos;
	printf("tracemalloc :%d\n", tracemalloc);

	uint64_t import_time = (uint64_t)&pconfig->import_time - pos;
	printf("import_time :%d\n", import_time);

	uint64_t show_ref_count = (uint64_t)&pconfig->show_ref_count - pos;
	printf("show_ref_count :%d\n", show_ref_count);

	uint64_t dump_refs = (uint64_t)&pconfig->dump_refs - pos;
	printf("dump_refs :%d\n", dump_refs);

	uint64_t malloc_stats = (uint64_t)&pconfig->malloc_stats - pos;
	printf("malloc_stats :%d\n", malloc_stats);

	uint64_t filesystem_encoding = (uint64_t)&pconfig->filesystem_encoding - pos;
	printf("filesystem_encoding :%d\n", filesystem_encoding);

	uint64_t filesystem_errors = (uint64_t)&pconfig->filesystem_errors - pos;
	printf("filesystem_errors :%d\n", filesystem_errors);

	uint64_t pycache_prefix = (uint64_t)&pconfig->pycache_prefix - pos;
	printf("pycache_prefix :%d\n", pycache_prefix);

	uint64_t parse_argv = (uint64_t)&pconfig->parse_argv - pos;
	printf("parse_argv :%d\n", parse_argv);

	uint64_t orig_argv = (uint64_t)&pconfig->orig_argv - pos;
	printf("orig_argv :%d\n", orig_argv);

	uint64_t argv = (uint64_t)&pconfig->argv - pos;
	printf("argv :%d\n", argv);

	uint64_t xoptions = (uint64_t)&pconfig->xoptions - pos;
	printf("xoptions :%d\n", xoptions);

	uint64_t home = (uint64_t)&pconfig->home - pos;
	printf("home :%d\n", home);

	uint64_t _isolated_interpreter = (uint64_t)&pconfig->_isolated_interpreter - pos;
	printf("_isolated_interpreter :%d\n", _isolated_interpreter);
	printf("actual size :%d\n", _isolated_interpreter + sizeof(pconfig->_isolated_interpreter));

    PyConfig_InitPythonConfig(config);
	printf("Inited config\n");

    // Set the program name
    config->program_name = L"embedded_python";

    // Set the Python interpreter's home directory (path to the conda environment)
    config->home = L"/Users/richardinsley/Projects/comfycli/kindalib/tests/micromamba/envs/myenv3.10";

    // Set the Python script to run
    const char *script_path = "/Users/richardinsley/Projects/comfycli/kindalib/tests/runmain/quote_cli.py";

    // Initialize the Python interpreter with the specified configuration
	printf("Initializing Python\n");
    PyStatus status = Py_InitializeFromConfig(config);
    if (PyStatus_Exception(status)) {
        PyConfig_Clear(config);
        return 1;
    }
	printf("Inited Python\n");

	// Run the Python script using PyRun_SimpleFileExFlags
    FILE *file = fopen(script_path, "r");
    if (file == NULL) {
        fprintf(stderr, "Failed to open the script file.\n");
        Py_Finalize();
        PyConfig_Clear(config);
        return 1;
    }
    int ret = PyRun_SimpleFileExFlags(file, script_path, 1, NULL);
    fclose(file);

	printf("Py_Finalize\n");
    // Finalize the Python interpreter
    // Py_Finalize();
	printf("fin\n");

    // while (keepRunning) {
    //     sleep(5); // Sleep for 1 second
    // }

	// print the contents of the config->program_name
	printf("Program name: %ls\n", config->program_name);

	// print the contents of the config->home
	printf("Program name: %ls\n", config->home);

	// print the address of home
	void * hadr = (void*)&config->home;
	printf("Home address: %p\n", hadr);

    printf("Gracefully exiting...\n");
}

*/
import "C"

func main() {
	C.test2()

	// get the buffer
	buffer := C.get_buffer()

	// decode the home member

	// Specify the binary folder to place micromamba in
	cwd, err := os.Getwd()
	rootDirectory := filepath.Join(cwd, "..", "micromamba")
	fmt.Println("Creating Kinda repo at: ", rootDirectory)
	version := "3.10"
	env, err := kinda.CreateEnvironment("myenv"+version, rootDirectory, version, "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}
	fmt.Printf("Created environment: %s\n", env.Name)

	// installing quote with pip
	err = env.PipInstallPackage("quote", "", "")
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

	// print the home in the pyconfig
	// get the buffer
	lbuffer := C.get_buffer()
	fmt.Println(lbuffer, buffer)
	p := lib.(*pylib.PythonLib)
	p.PyConfig = unsafe.Pointer(buffer)

	// get the true home
	thome := C.get_home()

	// get the home member / off by 8!
	home := p.GetPyConfigPointer("home")

	// dereference the home member pointer 0x12f8042e8 0x12f8042f0 / 5091902184 5091902192
	thome = *(*unsafe.Pointer)(thome)
	homeStr2 := pylib.WcharPtrToString((pylib.WcharPtr)(thome))

	// dereference the home member pointer
	home = *(*unsafe.Pointer)(home)

	homeStr := pylib.WcharPtrToString(pylib.WcharPtr(home))
	fmt.Println("Home:", homeStr2, "MyHome: ", homeStr, "My Home: ", home, "True Home: ", thome)

	// call runmain with the quote package
	/*
			    {
		        "name": "Py_DecodeLocale",
		        "return_type": "wchar_t*",
		        "parameters": [
		            {
		                "name": "arg",
		                "type": "char*"
		            },
		            {
		                "name": "size",
		                "type": "size_t*"
		            }
		        ]
		    },
	*/

	/*
		// we need to tell python where it's env is at
		// wchar_t *python_home = Py_DecodeLocale("/path/to/your/conda/env", NULL);
		// Py_SetPythonHome(python_home);
		envpathchar := pylib.StrToPtr(env.EnvPath)
		envpath := lib.Invoke("Py_DecodeLocale", envpathchar, 0)
		lib.Invoke("Py_SetPythonHome", envpath)

		// Initialize Python interpreter
		lib.Invoke("Py_Initialize")
	*/

	// yes, the offset calculation is correct
	// 248 + 5044715680 = 5044715928

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
}

/*
#include <Python.h>

int main() {
    PyStatus status;
    PyConfig config;

    PyConfig_InitPythonConfig(&config);

    // Set the Python home directory
    wchar_t *python_home = Py_DecodeLocale("/Users/richardinsley/Projects/comfycli/kindalib/tests/micromamba/envs/myenv3.10", NULL);
    if (!python_home) {
        fprintf(stderr, "Fatal error: cannot decode Python home\n");
        exit(1);
    }
    config.home = python_home;

    // Read the configuration based on the current settings (including command line arguments)
    status = PyConfig_Read(&config);
    if (PyStatus_Exception(status)) {
        PyConfig_Clear(&config);
        Py_ExitStatusException(status);
    }

    // Initialize Python with the given configuration
    status = Py_InitializeFromConfig(&config);
    if (PyStatus_Exception(status)) {
        PyConfig_Clear(&config);
        Py_ExitStatusException(status);
    }

    // Your code here. For example, execute a Python script or interact with Python objects

    // Finalize the Python interpreter
    Py_Finalize();

    // Clean up
    PyConfig_Clear(&config);
    PyMem_RawFree(python_home);

    return 0;
}
*/
