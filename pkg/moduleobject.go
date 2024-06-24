package pkg

import (
	"unsafe"
)

type PyModuleDef struct {
	Buffer    []byte
	PyConfig  *PyConfig
	PythonLib *PythonLib
}

// GetBuffer returns the address of the buffer as a uintptr
func (p PyModuleDef) GetBuffer() uintptr {
	return uintptr(unsafe.Pointer(&p.Buffer[0]))
}

func (p *PythonLib) NewPyModuleDef(name string, doc string, methods *PyMethodDefArray) PyModuleDef {
	pconf := &p.CTags.PyStructs.PyModuleDef
	baseconf := &p.CTags.PyStructs.PyModuleDef_Base
	pyobjconf := &p.CTags.PyStructs.PyObject

	retv := PyModuleDef{
		Buffer:    make([]byte, pconf.Size),
		PyConfig:  pconf,
		PythonLib: p,
	}

	// #define PyModuleDef_HEAD_INIT {  \
	// 	PyObject_HEAD_INIT(_Py_NULL) \
	// 	_Py_NULL, /* m_init */       \
	// 	0,        /* m_index */      \
	// 	_Py_NULL, /* m_copy */       \
	//   }

	// #define PyObject_HEAD_INIT(type) \
	// {                            \
	//     _PyObject_EXTRA_INIT     \
	//     { 1 },                   \
	//     (type)                   \
	// },

	// define _PyObject_EXTRA_INIT

	// define _Py_NULL nullptr

	// get the struct offset for m_base (PyModuleDef_Base)
	baseoffset := pconf.GetMemberOffset("m_base")

	// get the struct offset for ob_base (PyObject)
	pyobjectoffset := baseconf.GetMemberOffset("ob_base")
	ob_refcntoffset := pyobjconf.GetMemberOffset("ob_refcnt")
	ob_typeoffset := pyobjconf.GetMemberOffset("ob_type")

	// set the pyobject ob_refcnt to 1
	*(*int64)(unsafe.Pointer(&retv.Buffer[baseoffset+pyobjectoffset+ob_refcntoffset])) = 1

	// set the pyobject ob_type to the null type
	*(*uintptr)(unsafe.Pointer(&retv.Buffer[baseoffset+pyobjectoffset+ob_typeoffset])) = 0

	// set the name field to the struct offset
	nameoffset := pconf.GetMemberOffset("m_name")
	n := p.StrToPtr(name)
	*(*uintptr)(unsafe.Pointer(&retv.Buffer[nameoffset])) = n

	// set the doc field to the struct offset
	docoffset := pconf.GetMemberOffset("m_doc")
	d := p.StrToPtr(doc)
	*(*uintptr)(unsafe.Pointer(&retv.Buffer[docoffset])) = d

	// set the size field to the struct offset to -1 (size is an size_t)
	sizeoffset := pconf.GetMemberOffset("m_size")
	*(*int64)(unsafe.Pointer(&retv.Buffer[sizeoffset])) = -1

	// set the methods field to the struct offset
	methodsoffset := pconf.GetMemberOffset("m_methods")
	if methods == nil {
		*(*uintptr)(unsafe.Pointer(&retv.Buffer[methodsoffset])) = 0
	} else {
		*(*uintptr)(unsafe.Pointer(&retv.Buffer[methodsoffset])) = methods.GetBuffer()
	}

	return retv
}
