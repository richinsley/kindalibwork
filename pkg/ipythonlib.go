package pkg

import (
	_ "embed"
	"unsafe"
)

type IPythonLib interface {
	Invoke(f string, a ...uintptr) uintptr
	GetFTable() map[string]unsafe.Pointer
}

type PyFunctionParameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type PyFunction struct {
	Name       string                `json:"name"`
	ReturnType string                `json:"return_type"`
	Parameters []PyFunctionParameter `json:"parameters"`
}

type PythonLib struct {
	FTable        map[string]unsafe.Pointer
	FunctionDefs  []PyFunction
	FunctionNames []string
}
