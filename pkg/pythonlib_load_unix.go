//go:build linux || darwin

package pkg

import (
	"github.com/jwijenbergh/purego"
)

func loadLibrary(libpath string) (uintptr, error) {
	dll, err := purego.Dlopen(libpath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0, err
	}
	return dll, nil
}
