package pkg

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"unsafe"

	"github.com/ebitengine/purego"
	kinda "github.com/richinsley/kinda/pkg"
)

type InvokeFunc func(args ...uintptr) uintptr
type InvokeFuncVoid func(args ...uintptr)

type InvokeFunc0 func() uintptr
type InvokeFunc1 func(uintptr) uintptr
type InvokeFunc2 func(uintptr, uintptr) uintptr
type InvokeFunc3 func(uintptr, uintptr, uintptr) uintptr
type InvokeFunc4 func(uintptr, uintptr, uintptr, uintptr) uintptr
type InvokeFunc5 func(uintptr, uintptr, uintptr, uintptr, uintptr) uintptr
type InvokeFunc6 func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) uintptr
type InvokeFunc7 func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) uintptr
type InvokeFunc8 func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) uintptr
type InvokeFunc9 func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) uintptr
type InvokeFunc10 func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) uintptr

type PythonLib struct {
	CTags         *PyCtags
	FTable        map[string]interface{}
	FunctionDefs  map[string]PyFunction
	FunctionNames []string
	Environment   *kinda.Environment
	PyConfig      unsafe.Pointer
	DLL           uintptr
	Malloc        InvokeFunc
	Calloc        InvokeFunc
	Realloc       InvokeFunc
	Free          InvokeFunc
	PyNone        uintptr
}

func getFunction(functiondef PyFunction, dll uintptr) interface{} {
	pcount := len(functiondef.Parameters)
	if pcount == 1 && functiondef.Parameters[0].Type == "void" {
		// some function defs indicate a single void parameter when they have no parameters
		pcount = 0
	}

	fptr, err := OpenSymbol(dll, functiondef.Name)
	if err != nil {
		return nil
	}

	switch pcount {
	case 0:
		var ff InvokeFunc0
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 1:
		var ff InvokeFunc1
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 2:
		var ff InvokeFunc2
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 3:
		var ff InvokeFunc3
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 4:
		var ff InvokeFunc4
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 5:
		var ff InvokeFunc5
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 6:
		var ff InvokeFunc6
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 7:
		var ff InvokeFunc7
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 8:
		var ff InvokeFunc8
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 9:
		var ff InvokeFunc9
		purego.RegisterFunc(&ff, fptr)
		return ff
	case 10:
		var ff InvokeFunc10
		purego.RegisterFunc(&ff, fptr)
		return ff
	default:
		return nil
	}
}

func loadPythonFunctions(libpath string, functionDefs map[string]PyFunction, functionPointers map[string]interface{}) (uintptr, error) {
	// store the current diectory
	cwd, err := os.Getwd()
	if err != nil {
		return 0, err
	}

	// set current directory to path with the windows dll's
	// TODO, find how to set the correct dll search paths.
	goFriendlyPath := filepath.ToSlash(libpath)
	basepath, _ := path.Split(goFriendlyPath)

	err = os.Chdir(basepath)
	if err != nil {
		fmt.Println("nope")
	}
	defer os.Chdir(cwd)

	dll, err := OpenLibrary(libpath)
	if err != nil {
		log.Fatalf("Failed to load library: %v", err)
		return 0, err
	}

	for _, f := range functionDefs {
		if f.Name[0] == '_' {
			// skip private functions
			continue
		}

		proc := getFunction(f, dll)
		if proc == nil {
			log.Printf("Error loading %s: %v", f.Name, err)
			functionPointers[f.Name] = nil
		} else {
			functionPointers[f.Name] = proc
		}
	}

	return dll, nil
}

func (p *PythonLib) StrToPtr(str string) uintptr {
	// Convert the Go string to a null-terminated byte slice
	bytes := append([]byte(str), 0)

	// Allocate memory for the string including the null terminator
	addr := p.Invoke("PyMem_Malloc", uintptr(len(bytes)))
	if addr == 0 {
		return 0
	}

	// Copy the bytes to the allocated memory
	for i, b := range bytes {
		*(*byte)(unsafe.Pointer(addr + uintptr(i))) = b
	}

	return addr
}

func (p *PythonLib) FreeString(s uintptr) {
	p.Invoke("PyMem_Free", s)
}

// func (p *PythonLib) LoadSymbol(name string) (uintptr, error) {
// 	return purego.Dlsym(p.DLL, name)
// }

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
		FTable: make(map[string]interface{}),
	}

	// extract function names
	retv.FunctionNames = make([]string, len(retv.CTags.Functions))
	retv.FunctionDefs = make(map[string]PyFunction)
	for i, v := range retv.CTags.Functions {
		retv.FunctionNames[i] = v.Name
		retv.FunctionDefs[v.Name] = v
	}

	procs := make(map[string]interface{}, len(retv.FunctionNames))
	dll, err := loadPythonFunctions(libpath, retv.FunctionDefs, procs)
	if err != nil {
		return nil, err
	}

	// save the DLL and the procs for later use
	// remember, a proc in this case is not a function pointer.  We get the proc's fptr with .Addr
	retv.DLL = dll

	// Check for NULL pointers
	for k, ptr := range procs {
		retv.FTable[k] = ptr
	}

	// py_none is a global static PyObject* that is used to return None from C functions
	// it is available in the python library as "_Py_NoneStruct" and marked as "PyAPI_DATA(PyObject) _Py_NoneStruct;"
	pynone, err := OpenSymbol(dll, "_Py_NoneStruct")
	if err != nil {
		fmt.Printf("Error loading Py_None: %s\n", err.Error())
	} else {
		retv.PyNone = pynone
	}

	return retv, nil
}

func (p *PythonLib) GetFTableCount() int {
	return len(p.FTable)
}

func (p *PythonLib) Invoke(f string, a ...uintptr) uintptr {
	pcount := len(a)
	switch pcount {
	case 0:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc0)
		retv := ff()
		return retv
	case 1:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc1)
		retv := ff(a[0])
		return retv
	case 2:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc2)
		retv := ff(a[0], a[1])
		return retv
	case 3:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc3)
		retv := ff(a[0], a[1], a[2])
		return retv
	case 4:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc4)
		retv := ff(a[0], a[1], a[2], a[3])
		return retv
	case 5:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc5)
		retv := ff(a[0], a[1], a[2], a[3], a[4])
		return retv
	case 6:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc6)
		retv := ff(a[0], a[1], a[2], a[3], a[4], a[5])
		return retv
	case 7:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc7)
		retv := ff(a[0], a[1], a[2], a[3], a[4], a[5], a[6])
		return retv
	case 8:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc8)
		retv := ff(a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7])
		return retv
	case 9:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc9)
		retv := ff(a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[8])
		return retv
	case 10:
		fn := p.FTable[f]
		ff := fn.(InvokeFunc10)
		retv := ff(a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[8], a[9])
		return retv
	default:
		return 0
	}
}

func (p *PythonLib) AllocBuffer(size int) uintptr {
	return p.Malloc(uintptr(size))
}

func (p *PythonLib) FreeBuffer(addr uintptr) {
	p.Free(addr)
}

func (p *PythonLib) Init(program_name string) error {
	// Doesn't work >= 3.11 !!!
	// we need to tell python where it's env is at
	envpathchar := p.StrToPtr(p.Environment.EnvPath)
	envpath := p.Invoke("Py_DecodeLocale", envpathchar, 0)
	p.Invoke("Py_SetPythonHome", envpath)

	// Initialize Python interpreter
	p.Invoke("Py_Initialize")

	return nil
}

func (p *PythonLib) GetPyNone() uintptr {
	// return uintptr(unsafe.Pointer(C.our_Py_NoneStruct))
	return p.PyNone
}
