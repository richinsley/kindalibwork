package pkg

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"unsafe"

	kinda "github.com/richinsley/kinda/pkg"
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
	CTags         *PyCtags
	FTable        map[string]*syscall.Proc
	FunctionDefs  []PyFunction
	FunctionNames []string
	Environment   *kinda.Environment
	PyConfig      unsafe.Pointer
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

func NewPythonLib(env *kinda.Environment) (IPythonLib, error) {
	retv, err := NewPythonLibFromPaths(env.PythonLibPath, env.EnvPath, env.SitePackagesPath, env.PythonVersion.MinorString())
	if err != nil {
		return nil, err
	}

	myenv := retv.(*PythonLib)
	myenv.Environment = env

	return retv, nil
}

func NewPythonLibFromPaths(libpath string, pyhome string, pypkg string, version string) (IPythonLib, error) {
	ctags, err := GetPlatformCtags(version)
	if err != nil {
		return nil, err
	}

	retv := &PythonLib{
		CTags:  ctags,
		FTable: make(map[string]*syscall.Proc),
	}

	// extract function names
	retv.FunctionNames = make([]string, len(retv.CTags.Functions))
	for i, v := range retv.CTags.Functions {
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
		// if ptr == nil {
		// 	log.Printf("Function %s failed to load.", retv.FunctionNames[i])
		// } else {
		// 	log.Printf("Function %s loaded.", retv.FunctionNames[i])
		// }
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

func (p *PythonLib) AllocBuffer(size int) uintptr {
	retv, _ := VirtualAlloc(uintptr(size))
	return retv
}

func (p *PythonLib) FreeBuffer(addr uintptr) {
	VirtualFree(addr)
}

func (p *PythonLib) Init(program_name string) error {
	// we need to tell python where it's env is at
	envpathchar := StrToPtr(p.Environment.EnvPath)
	envpath := p.Invoke("Py_DecodeLocale", envpathchar, 0)
	p.Invoke("Py_SetPythonHome", envpath)

	// Initialize Python interpreter
	p.Invoke("Py_Initialize")

	return nil
}
