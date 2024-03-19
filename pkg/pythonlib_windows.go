package pkg

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	loadLibraryW     = kernel32.NewProc("LoadLibraryW")
	getProcAddress   = kernel32.NewProc("GetProcAddress")
	freeLibrary      = kernel32.NewProc("FreeLibrary")
	procVirtualAlloc = kernel32.NewProc("VirtualAlloc")
	procVirtualFree  = kernel32.NewProc("VirtualFree")
)

type PythonLib struct {
	FTable        map[string]*syscall.Proc
	FunctionDefs  []PyFunction
	FunctionNames []string
	DLL           *syscall.DLL
}

func loadPythonFunctions(libpath string, functionNames []string, functionPointers []*syscall.Proc) (*syscall.DLL, error) {
	// store the current diectory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// set current directory to path with the windows dll's
	// TODO, find how to set the correct dll search paths.
	goFriendlyPath := filepath.ToSlash(libpath)
	basepath, _ := path.Split(goFriendlyPath)
	// err = os.Chdir("C:/Users/johnn/comfycli_forge/kindalibwork/micromamba/envs/myenv3.10")
	err = os.Chdir(basepath)
	if err != nil {
		fmt.Println("nope")
	}
	defer os.Chdir(cwd)

	dll, err := syscall.LoadDLL(libpath)
	if err != nil {
		log.Fatalf("Failed to load library: %v", err)
		return nil, err
	}
	// defer dll.Release()

	for i, name := range functionNames {
		proc, err := dll.FindProc(name)
		if err != nil {
			log.Printf("Error loading %s: %v", name, err)
			functionPointers[i] = nil
		} else {
			functionPointers[i] = proc
		}
	}

	return dll, nil
}

//go:embed ctags/ctags-39.json
var functionsJson39 []byte

//go:embed ctags/ctags-310.json
var functionsJson310 []byte

//go:embed ctags/ctags-311.json
var functionsJson311 []byte

//go:embed ctags/ctags-312.json
var functionsJson312 []byte

var pythonCtags = map[string][]byte{
	"3.9":  functionsJson39,
	"3.10": functionsJson310,
	"3.11": functionsJson311,
	"3.12": functionsJson312,
}

const (
	MEM_COMMIT     = 0x1000
	MEM_RELEASE    = 0x8000
	MEM_RESERVE    = 0x2000
	PAGE_READWRITE = 0x04
)

func VirtualAlloc(size uintptr) (uintptr, error) {
	addr, _, err := procVirtualAlloc.Call(
		0, // Let the system determine where to allocate memory
		size,
		MEM_COMMIT|MEM_RESERVE,
		PAGE_READWRITE,
	)
	if addr == 0 {
		return 0, err
	}
	return addr, nil
}

func VirtualFree(addr uintptr) error {
	ret, _, err := procVirtualFree.Call(
		addr,
		0, // Size must be 0 if MEM_RELEASE is used
		MEM_RELEASE,
	)
	if ret == 0 {
		return err
	}
	return nil
}

func StrToPtr(str string) uintptr {
	// Convert the Go string to a null-terminated byte slice
	bytes := append([]byte(str), 0)

	// Allocate memory for the string including the null terminator
	addr, err := VirtualAlloc(uintptr(len(bytes)))
	if err != nil {
		return 0
	}

	// Copy the bytes to the allocated memory
	for i, b := range bytes {
		*(*byte)(unsafe.Pointer(addr + uintptr(i))) = b
	}

	return addr
}

func FreeString(s uintptr) {
	VirtualFree(s)
}

func NewPythonLib(libpath string, pyhome string, pypkg string, version string) (IPythonLib, error) {
	retv := &PythonLib{
		FTable: make(map[string]*syscall.Proc),
	}

	// Parse the JSON data into the FunctionDefs
	err := json.Unmarshal(pythonCtags[version], &retv.FunctionDefs)
	if err != nil {
		return nil, err
	}

	// extract function names
	retv.FunctionNames = make([]string, len(retv.FunctionDefs))
	for i, v := range retv.FunctionDefs {
		retv.FunctionNames[i] = v.Name
	}

	procs := make([]*syscall.Proc, len(retv.FunctionNames))
	dll, err := loadPythonFunctions(libpath, retv.FunctionNames, procs)
	if err != nil {
		return nil, err
	}

	// save the DLL and the procs for later use
	// remember, a proc in this case is not a function pointer.  We get the proc's fptr with .Addr
	retv.DLL = dll

	// Check for NULL pointers and use the functions...
	for i, ptr := range procs {
		retv.FTable[retv.FunctionNames[i]] = ptr
		if ptr == nil {
			log.Printf("Function %s failed to load.", retv.FunctionNames[i])
		} else {
			log.Printf("Function %s loaded.", retv.FunctionNames[i])
		}
	}

	return retv, nil
}

func (p *PythonLib) GetFTableCount() int {
	return len(p.FTable)
}

func (p *PythonLib) Invoke(f string, a ...uintptr) uintptr {
	fn := p.FTable[f]

	// Call executes procedure p with arguments a.
	//
	// The returned error is always non-nil, constructed from the result of GetLastError.
	// Callers must inspect the primary return value to decide whether an error occurred
	// (according to the semantics of the specific function being called) before consulting
	// the error. The error always has type syscall.Errno.
	//
	// On amd64, Call can pass and return floating-point values. To pass
	// an argument x with C type "float", use
	// uintptr(math.Float32bits(x)). To pass an argument with C type
	// "double", use uintptr(math.Float64bits(x)). Floating-point return
	// values are returned in r2. The return value for C type "float" is
	// math.Float32frombits(uint32(r2)). For C type "double", it is
	// math.Float64frombits(uint64(r2)).
	retv, _, _ := fn.Call(a...)
	return retv
}
