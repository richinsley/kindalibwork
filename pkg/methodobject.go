package pkg

import (
	"unsafe"
)

const (
	METH_VARARGS  = 0x0001
	METH_KEYWORDS = 0x0002
	METH_NOARGS   = 0x0004
	METH_O        = 0x0008
	METH_CLASS    = 0x0010
	METH_STATIC   = 0x0020
	METH_COEXIST  = 0x0040
	METH_FASTCALL = 0x0080
	METH_METHOD   = 0x0200
)

// PyMethodDef is a structure used to define a method in a Python module.
// It is used to define the methods that are available in a module.
// The last entry in the array must be a NULL entry.
type PyMethodDef struct {
	Name  string
	Meth  uintptr
	Flags int
}

type PyMethodDefArray struct {
	Buffer    []byte
	PyConfig  *PyConfig
	PythonLib *PythonLib
}

// GetBuffer returns the address of the buffer as a uintptr
func (p PyMethodDefArray) GetBuffer() uintptr {
	return uintptr(unsafe.Pointer(&p.Buffer[0]))
}

func (p *PythonLib) NewPyMethodDefArray(count int) PyMethodDefArray {
	retv := PyMethodDefArray{
		PyConfig:  &p.CTags.PyStructs.PyMethodDef,
		PythonLib: p,
	}
	retv.Buffer = make([]byte, (count+1)*p.CTags.PyStructs.PyMethodDef.Size)
	return retv
}

func (p PyMethodDefArray) SetMethodDef(index int, name string, meth uintptr, flags int) {
	indexoffset := index * p.PyConfig.Size
	// set the name field to the struct offset
	nameoffset := p.PyConfig.GetMemberOffset("ml_name")

	// write the n uintptr directly to the buffer at the nameoffset
	n := p.PythonLib.StrToPtr(name)
	*(*uintptr)(unsafe.Pointer(&p.Buffer[indexoffset+nameoffset])) = n

	// set the meth field to the struct offset
	methoffset := p.PyConfig.GetMemberOffset("ml_meth")
	*(*uintptr)(unsafe.Pointer(&p.Buffer[indexoffset+methoffset])) = meth

	// set the flags field to the struct offset (flags is an int)
	flagsoffset := p.PyConfig.GetMemberOffset("ml_flags")
	*(*int)(unsafe.Pointer(&p.Buffer[indexoffset+flagsoffset])) = flags

	// set the ml_doc field to NULL
	docoffset := p.PyConfig.GetMemberOffset("ml_doc")
	*(*uintptr)(unsafe.Pointer(&p.Buffer[indexoffset+docoffset])) = 0
}
