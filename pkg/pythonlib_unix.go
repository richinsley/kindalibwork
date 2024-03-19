//go:build linux || darwin

package pkg

import (
	_ "embed"
	"encoding/json"
	"log"
	"strconv"
	"unsafe"
)

/*
#cgo LDFLAGS: -ldl

#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h> // Include stdlib.h for C.free

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

// func (env *Environment) NewPythonLib() (*PythonLib, error) {
// 	return NewPythonLib(env.PythonLibPath, env.EnvPath, env.SitePackagesPath, env.PythonVersion.MinorString())
// }

func NewPythonLib(libpath string, pyhome string, pypkg string, version string) (IPythonLib, error) {
	// os.Setenv("PYTHONHOME", "/Users/richardinsley/miniconda3/envs/py39")
	// os.Setenv("PYTHONPATH", "/Users/richardinsley/miniconda3/envs/py39/lib/python3.9/site-packages")

	// os.Setenv("PYTHONHOME", pyhome)
	// os.Setenv("PYTHONPATH", pypkg)
	retv := &PythonLib{
		FTable: make(map[string]unsafe.Pointer),
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

	// Prepare C arrays
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
		if ptr == nil {
			log.Printf("Function %s failed to load.", retv.FunctionNames[i])
		} else {
			log.Printf("Function %s loaded.", retv.FunctionNames[i])
		}
	}

	return retv, nil
}

func (p *PythonLib) GetFTable() map[string]unsafe.Pointer {
	return p.FTable
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
