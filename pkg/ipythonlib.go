package pkg

import (
	_ "embed"
)

type IPythonLib interface {
	Invoke(f string, a ...uintptr) uintptr
	GetFTableCount() int
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
