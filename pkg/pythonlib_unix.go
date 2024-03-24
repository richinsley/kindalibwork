//go:build linux || darwin

package pkg

import (
	_ "embed"
	"fmt"
	"log"
	"strconv"
	"unsafe"

	kinda "github.com/richinsley/kinda/pkg"
)

/*
#cgo LDFLAGS: -ldl

#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h> // Include stdlib.h for C.free
#include <string.h>

void* openPythonLib(const char* name) {
    return dlopen(name, RTLD_LAZY);
}

void closePythonLib(void* handle) {
    dlclose(handle);
}

void loadPythonFunctions(char * libpath, char** functionNames, void** functionPointers, int count) {
    char *error;
    void* handle = dlopen(libpath, RTLD_NOW);
    if (handle == NULL) {
        fprintf(stderr, "%s\n", dlerror());
        return;
    }

    for (int i = 0; i < count; i++) {
        dlerror(); // Clear any existing error
        functionPointers[i] = dlsym(handle, functionNames[i]);
        if ((error = dlerror()) != NULL) {
            fprintf(stderr, "Error loading %s: %s\n", functionNames[i], error);
            functionPointers[i] = NULL;
        }
    }
}

static uint64_t Syscall0(void* addr) {
	return ((uint64_t(*)())addr)();
}

static uint64_t Syscall1(void* addr, void* p1) {
	return ((uint64_t(*)(void*))addr)(p1);
}

static uint64_t Syscall2(void* addr, void* p1, void* p2) {
	return ((uint64_t(*)(void*,void*))addr)(p1, p2);
}

static uint64_t Syscall3(void* addr, void* p1, void* p2, void* p3) {
	return ((uint64_t(*)(void*,void*,void*))addr)(p1, p2, p3);
}

static uint64_t Syscall4(void* addr, void* p1, void* p2, void* p3, void* p4) {
	return ((uint64_t(*)(void*,void*,void*,void*))addr)(p1, p2, p3, p4);
}

static uint64_t Syscall5(void* addr, void* p1, void* p2, void* p3, void* p4, void* p5) {
	return ((uint64_t(*)(void*,void*,void*,void*,void*))addr)(p1, p2, p3, p4, p5);
}

static uint64_t Syscall6(void* addr, void* p1, void* p2, void* p3, void* p4, void* p5, void* p6) {
	return ((uint64_t(*)(void*,void*,void*,void*,void*,void*))addr)(p1, p2, p3, p4, p5, p6);
}

static uint64_t Syscall7(void* addr, void* p1, void* p2, void* p3, void* p4, void* p5, void* p6, void *p7) {
	return ((uint64_t(*)(void*,void*,void*,void*,void*,void*, void*))addr)(p1, p2, p3, p4, p5, p6, p7);
}

static uint64_t Syscall8(void* addr, void* p1, void* p2, void* p3, void* p4, void* p5, void* p6, void *p7, void *p8) {
	return ((uint64_t(*)(void*,void*,void*,void*,void*,void*,void*,void*))addr)(p1, p2, p3, p4, p5, p6,p7,p8);
}

int run_python_script(const char* script, void * f) {
	int (*fun_ptr)(const char*) = f;
	return (*fun_ptr)(script);
}

*/
import "C"

type PythonLib struct {
	CTags         *PyCtags
	FTable        map[string]unsafe.Pointer
	FunctionNames []string
	Environment   *kinda.Environment
	PyConfig      unsafe.Pointer
}

func ToPtr(a uintptr) unsafe.Pointer {
	return unsafe.Pointer(a)
}

func CStrToPtr(s *C.char) unsafe.Pointer {
	return unsafe.Pointer(s)
}

func StrToPtr(s string) uintptr {
	return uintptr(CStrToPtr(C.CString(s)))
}

func FreeString(s uintptr) {
	C.free(unsafe.Pointer(s))
}

// func (env *Environment) NewPythonLib() (*PythonLib, error) {
// 	return NewPythonLib(env.PythonLibPath, env.EnvPath, env.SitePackagesPath, env.PythonVersion.MinorString())
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
	// os.Setenv("PYTHONHOME", "/Users/richardinsley/miniconda3/envs/py39")
	// os.Setenv("PYTHONPATH", "/Users/richardinsley/miniconda3/envs/py39/lib/python3.9/site-packages")

	// os.Setenv("PYTHONHOME", pyhome)
	// os.Setenv("PYTHONPATH", pypkg)

	ctags, err := GetPlatformCtags(version)
	if err != nil {
		return nil, err
	}

	retv := &PythonLib{
		CTags:  ctags,
		FTable: make(map[string]unsafe.Pointer),
	}

	// extract function names
	retv.FunctionNames = make([]string, len(retv.CTags.Functions))
	for i, v := range retv.CTags.Functions {
		retv.FunctionNames[i] = v.Name
	}

	// Prepare C function names and pointers
	cFunctionNames := make([]*C.char, len(retv.FunctionNames))
	for i, name := range retv.FunctionNames {
		cFunctionNames[i] = C.CString(name)
		defer C.free(unsafe.Pointer(cFunctionNames[i]))
	}

	cFunctionPointers := make([]unsafe.Pointer, len(retv.FunctionNames))
	lp := C.CString(libpath)
	defer C.free(unsafe.Pointer(lp))
	C.loadPythonFunctions(lp, &cFunctionNames[0], &cFunctionPointers[0], C.int(len(retv.FunctionNames)))

	// Check for NULL pointers and use the functions...
	for i, ptr := range cFunctionPointers {
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

func (p *PythonLib) invoke0(f string) uintptr {
	return uintptr(C.Syscall0(p.FTable[f]))
}

func (p *PythonLib) invoke1(f string, p1 unsafe.Pointer) uintptr {
	return uintptr(C.Syscall1(p.FTable[f], p1))
}

func (p *PythonLib) invoke2(f string, p1 unsafe.Pointer, p2 unsafe.Pointer) uintptr {
	return uintptr(C.Syscall2(p.FTable[f], p1, p2))
}

func (p *PythonLib) invoke3(f string, p1 unsafe.Pointer, p2 unsafe.Pointer, p3 unsafe.Pointer) uintptr {
	return uintptr(C.Syscall3(p.FTable[f], p1, p2, p3))
}

func (p *PythonLib) invoke4(f string, p1 unsafe.Pointer, p2 unsafe.Pointer, p3 unsafe.Pointer, p4 unsafe.Pointer) uintptr {
	return uintptr(C.Syscall4(p.FTable[f], p1, p2, p3, p4))
}

func (p *PythonLib) invoke5(f string, p1 unsafe.Pointer, p2 unsafe.Pointer, p3 unsafe.Pointer, p4 unsafe.Pointer, p5 unsafe.Pointer) uintptr {
	return uintptr(C.Syscall5(p.FTable[f], p1, p2, p3, p4, p5))
}

func (p *PythonLib) invoke6(f string, p1 unsafe.Pointer, p2 unsafe.Pointer, p3 unsafe.Pointer, p4 unsafe.Pointer, p5 unsafe.Pointer, p6 unsafe.Pointer) uintptr {
	return uintptr(C.Syscall6(p.FTable[f], p1, p2, p3, p4, p5, p6))
}

func (p *PythonLib) invoke7(f string, p1 unsafe.Pointer, p2 unsafe.Pointer, p3 unsafe.Pointer, p4 unsafe.Pointer, p5 unsafe.Pointer, p6 unsafe.Pointer, p7 unsafe.Pointer) uintptr {
	return uintptr(C.Syscall7(p.FTable[f], p1, p2, p3, p4, p5, p6, p7))
}

func (p *PythonLib) invoke8(f string, p1 unsafe.Pointer, p2 unsafe.Pointer, p3 unsafe.Pointer, p4 unsafe.Pointer, p5 unsafe.Pointer, p6 unsafe.Pointer, p7 unsafe.Pointer, p8 unsafe.Pointer) uintptr {
	return uintptr(C.Syscall8(p.FTable[f], p1, p2, p3, p4, p5, p6, p7, p8))
}

func (p *PythonLib) Invoke(f string, a ...uintptr) uintptr {
	switch len(a) {
	case 0:
		return uintptr(p.invoke0(f))
	case 1:
		return uintptr(p.invoke1(f, ToPtr(a[0])))
	case 2:
		return uintptr(p.invoke2(f, ToPtr(a[0]), ToPtr(a[1])))
	case 3:
		return uintptr(p.invoke3(f, ToPtr(a[0]), ToPtr(a[1]), ToPtr(a[2])))
	case 4:
		return uintptr(p.invoke4(f, ToPtr(a[0]), ToPtr(a[1]), ToPtr(a[2]), ToPtr(a[3])))
	case 5:
		return uintptr(p.invoke5(f, ToPtr(a[0]), ToPtr(a[1]), ToPtr(a[2]), ToPtr(a[3]), ToPtr(a[4])))
	case 6:
		return uintptr(p.invoke6(f, ToPtr(a[0]), ToPtr(a[1]), ToPtr(a[2]), ToPtr(a[3]), ToPtr(a[4]), ToPtr(a[5])))
	case 7:
		return uintptr(p.invoke7(f, ToPtr(a[0]), ToPtr(a[1]), ToPtr(a[2]), ToPtr(a[3]), ToPtr(a[4]), ToPtr(a[5]), ToPtr(a[6])))
	case 8:
		return uintptr(p.invoke8(f, ToPtr(a[0]), ToPtr(a[1]), ToPtr(a[2]), ToPtr(a[3]), ToPtr(a[4]), ToPtr(a[5]), ToPtr(a[6]), ToPtr(a[7])))
	default:
		panic("invoke " + f + " with too many arguments " + strconv.Itoa(len(a)) + ".")
	}
}

func (p *PythonLib) AllocBuffer(size int) uintptr {
	rptr := C.malloc(C.size_t(size))
	// zero the buffer
	C.memset(rptr, 0, C.size_t(size))

	nptr := uintptr(rptr)
	return nptr
}

func (p *PythonLib) FreeBuffer(addr uintptr) {
	C.free(unsafe.Pointer(addr))
}

func (p *PythonLib) GetPyConfigPointer(member string) unsafe.Pointer {
	// get the offset for the member
	var offset int
	gotit := false
	for _, m := range p.CTags.PyConfigs.PyConfig.Members {
		if m.Name == member {
			gotit = true
			offset = m.Offset
			break
		}
	}

	if !gotit {
		log.Printf("Member %s not found in PyConfig struct.", member)
		return unsafe.Pointer(nil)
	}

	// get the pointer 5411727168 + 248 = 5411727416 (0x142906838)
	fmt.Printf("PyConfig: %p Offset P %p, Offset V %d \n", p.PyConfig, unsafe.Pointer(uintptr(offset)), offset)
	uptr := unsafe.Pointer(uintptr(p.PyConfig) + uintptr(offset))
	return uptr
}

func (p *PythonLib) SetPyConfigPointer(member string, ptr uintptr) {
	// get the offset for the member
	var offset int
	gotit := false
	for _, m := range p.CTags.PyConfigs.PyConfig.Members {
		if m.Name == member {
			gotit = true
			offset = m.Offset
			break
		}
	}

	if !gotit {
		log.Printf("Member %s not found in PyConfig struct.", member)
		return
	}

	// set the pointer
	*(*uintptr)(unsafe.Pointer(uintptr(p.PyConfig) + uintptr(offset))) = ptr
}

func (p *PythonLib) Init(program_name string) error {
	return nil
	/*
		fmt.Printf("There are %d members in the PyConfig struct.\n", len(p.CTags.PyConfigs.PyConfig.Members))
		var PyConfig uintptr
		if p.Environment != nil {
			// create a PyConfig struct and initialize it as an Environment
			// 368 should be 392

			// truesize := 0
			// for _, m := range p.CTags.PyConfigs.PyConfig.Members {
			// 	truesize += m.Size
			// }

			// if truesize != p.CTags.PyConfigs.PyConfig.Size {
			// 	log.Printf("PyConfig struct size mismatch: %d != %d\n", truesize, p.CTags.PyConfigs.PyConfig.Size)
			// }

			PyConfig = p.AllocBuffer(p.CTags.PyConfigs.PyConfig.Size + 256)
			p.PyConfig = ToPtr(PyConfig)
			p.Invoke("PyConfig_InitPythonConfig", uintptr(PyConfig))

			// set the home and program name in the PyConfig struct
			envpathchar := StrToPtr(p.Environment.EnvPath)
			envpath := p.Invoke("Py_DecodeLocale", envpathchar, 0)
			p.SetPyConfigPointer("home", envpath)

			pnamechar := StrToPtr(program_name)
			pname := p.Invoke("Py_DecodeLocale", pnamechar, 0)
			p.SetPyConfigPointer("program_name", pname)

			status := p.Invoke("Py_InitializeFromConfig", uintptr(PyConfig))
			if p.Invoke("PyStatus_Exception", status) != 0 {
				return fmt.Errorf("Py_InitializeFromConfig failed with status %d", status)
			}

			// Read the configuration based on the current settings (including command line arguments)
			// status := p.Invoke("PyConfig_Read", uintptr(PyConfig))
			// if status != 0 {
			// 	return errors.New("PyConfig_Read failed")
			// }

			// Set the Python home in the PyConfig struct (depricated in 3.10)
			// p.Invoke("Py_SetPythonHome", envpath)
		}

		// Initialize Python interpreter
		p.Invoke("Py_Initialize")

		return nil
	*/
}
