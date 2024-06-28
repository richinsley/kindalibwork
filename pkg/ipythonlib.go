package pkg

import (
	_ "embed"
	"runtime"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

type IPythonLib interface {
	Invoke(f string, a ...uintptr) uintptr
	GetFTableCount() int
	AllocBuffer(size int) uintptr
	FreeBuffer(addr uintptr)
	Init(string) error
	GetPyNone() uintptr

	NewPyMethodDefArray(count int) PyMethodDefArray
	NewPyModuleDef(name string, doc string, methods *PyMethodDefArray) PyModuleDef
	StrToPtr(str string) uintptr
	PtrToStr(ptr uintptr) string
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

type PyConfigMember struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
}

type PyConfig struct {
	Name    string           `json:"name"`
	Size    int              `json:"size"`
	Members []PyConfigMember `json:"members"`
}

func (p PyConfig) GetMemberOffset(name string) int {
	for _, m := range p.Members {
		if m.Name == name {
			return m.Offset
		}
	}
	return -1
}

type PyStructs struct {
	PyConfig         PyConfig `json:"PyConfig"`
	PyPreConfig      PyConfig `json:"PyPreConfig"`
	PyWideStringList PyConfig `json:"PyWideStringList"`
	PyObject         PyConfig `json:"PyObject"`
	PyMethodDef      PyConfig `json:"PyMethodDef"`
	PyModuleDef_Base PyConfig `json:"PyModuleDef_Base"`
	PyModuleDef      PyConfig `json:"PyModuleDef"`
}

type PyCtags struct {
	Functions []PyFunction      `json:"PyFunctions"`
	PyStructs PyStructs         `json:"PyStructs"`
	PyData    map[string]string `json:"PyData"`
}

type WcharPtr uintptr

// StringToWcharPtr converts a Go string to a wchar_t*.
func StringToWcharPtr(s string) WcharPtr {
	if runtime.GOOS == "windows" {
		// On Windows, use UTF-16 encoding
		utf16Chars := utf16.Encode([]rune(s))
		ptr := make([]uint16, len(utf16Chars)+1)
		copy(ptr, utf16Chars)
		ptr[len(utf16Chars)] = 0
		return WcharPtr(unsafe.Pointer(&ptr[0]))
	}

	// On other platforms, use UTF-32 encoding
	utf32Chars := make([]rune, 0, utf8.RuneCountInString(s)+1)
	for _, r := range s {
		utf32Chars = append(utf32Chars, r)
	}
	utf32Chars = append(utf32Chars, 0)
	return WcharPtr(unsafe.Pointer(&utf32Chars[0]))
}

// WcharPtrToString converts a wchar_t* to a Go string.
func WcharPtrToString(ptr WcharPtr) string {
	if runtime.GOOS == "windows" {
		// On Windows, use UTF-16 encoding
		utf16Chars := make([]uint16, 0, 256)
		for {
			c := *(*uint16)(unsafe.Pointer(uintptr(ptr)))
			ptr += WcharPtr(unsafe.Sizeof(uint16(0)))
			if c == 0 {
				break
			}
			utf16Chars = append(utf16Chars, c)
		}
		return string(utf16.Decode(utf16Chars))
	}

	// On other platforms, use UTF-32 encoding
	utf32Chars := make([]rune, 0, 256)
	for {
		c := *(*rune)(unsafe.Pointer(uintptr(ptr)))
		ptr += WcharPtr(unsafe.Sizeof(rune(0)))
		if c == 0 {
			break
		}
		utf32Chars = append(utf32Chars, c)
	}
	return string(utf32Chars)
}
