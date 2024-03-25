package main

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	kinda "github.com/richinsley/kinda/pkg"
	pylib "github.com/richinsley/kindalib/pkg"
)

func main() {
	// Specify the binary folder to place micromamba in
	cwd, _ := os.Getwd()
	rootDirectory := filepath.Join(cwd, "..", "micromamba")
	fmt.Println("Creating Kinda repo at: ", rootDirectory)
	version := "3.12"
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

    PyConfig_InitPythonConfig(config);
	printf("Inited config\n");

    // Set the program name
    config->program_name = L"embedded_python";

    // Set the Python interpreter's home directory (path to the conda environment)
    config->home = L"/Users/richardinsley/Projects/comfycli/kindalib/tests/micromamba/envs/myenv3.10";

    // Initialize the Python interpreter with the specified configuration
	printf("Initializing Python\n");
    PyStatus status = Py_InitializeFromConfig(config);
    if (PyStatus_Exception(status)) {
        PyConfig_Clear(config);
        return 1;
    }
	printf("Inited Python\n");

	// Run the Python script using PyRun_SimpleFileExFlags
	// Set the Python script to run
    const char *script_path = "/Users/richardinsley/Projects/comfycli/kindalib/tests/runmain/quote_cli.py";
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
