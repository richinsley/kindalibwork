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

type PuConfigMember struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
}

type PyConfig struct {
	Name    string           `json:"name"`
	Size    int              `json:"size"`
	Members []PuConfigMember `json:"members"`
}

type PyConfigStruct struct {
	PyConfig    PyConfig `json:"PyConfig"`
	PyPreConfig PyConfig `json:"PyPreConfig"`
}

type PyCtags struct {
	Functions []PyFunction   `json:"PyFunctions"`
	PyConfigs PyConfigStruct `json:"PyStructs"`
}

type WcharPtr uintptr

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
