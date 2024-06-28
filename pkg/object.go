package pkg

// PyObject is a pointer to a Python object
type PyObject uintptr

// These flags are used to specify attributes and behaviors of Python types. They are
// used in the tp_flags field of the type object.
const (
	// Placement of dict (and values) pointers are managed by the VM, not by the type.
	// The VM will automatically set tp_dictoffset. Should not be used for variable sized
	// classes, such as classes that extend tuple.
	Py_TPFLAGS_MANAGED_DICT uintptr = 1 << 4

	// Set if instances of the type object are treated as sequences for pattern matching
	Py_TPFLAGS_SEQUENCE uintptr = 1 << 5

	// Set if instances of the type object are treated as mappings for pattern matching
	Py_TPFLAGS_MAPPING uintptr = 1 << 6

	// Disallow creating instances of the type: set tp_new to NULL and don't create
	// the "__new__" key in the type dictionary.
	Py_TPFLAGS_DISALLOW_INSTANTIATION uintptr = 1 << 7

	// Set if the type object is immutable: type attributes cannot be set nor deleted
	Py_TPFLAGS_IMMUTABLETYPE uintptr = 1 << 8

	// Set if the type object is dynamically allocated
	Py_TPFLAGS_HEAPTYPE uintptr = 1 << 9

	// Set if the type allows subclassing
	Py_TPFLAGS_BASETYPE uintptr = 1 << 10

	// Set if the type implements the vectorcall protocol (PEP 590)
	Py_TPFLAGS_HAVE_VECTORCALL uintptr = 1 << 11

	// Set if the type is 'ready' -- fully initialized
	Py_TPFLAGS_READY uintptr = 1 << 12

	// Set while the type is being 'readied', to prevent recursive ready calls
	Py_TPFLAGS_READYING uintptr = 1 << 13

	// Objects support garbage collection (see objimpl.h)
	Py_TPFLAGS_HAVE_GC uintptr = 1 << 14

	// Objects behave like an unbound method
	Py_TPFLAGS_METHOD_DESCRIPTOR uintptr = 1 << 17

	// Object has up-to-date type attribute cache
	Py_TPFLAGS_VALID_VERSION_TAG uintptr = 1 << 19

	// Type is abstract and cannot be instantiated
	Py_TPFLAGS_IS_ABSTRACT uintptr = 1 << 20

	// This undocumented flag gives certain built-ins their unique pattern-matching
	// behavior, which allows a single positional subpattern to match against the
	// subject itself (rather than a mapped attribute on it):
	Py_TPFLAGS_MATCH_SELF uintptr = 1 << 22

	// These flags are used to determine if a type is a subclass.
	Py_TPFLAGS_LONG_SUBCLASS     uintptr = 1 << 24
	Py_TPFLAGS_LIST_SUBCLASS     uintptr = 1 << 25
	Py_TPFLAGS_TUPLE_SUBCLASS    uintptr = 1 << 26
	Py_TPFLAGS_BYTES_SUBCLASS    uintptr = 1 << 27
	Py_TPFLAGS_UNICODE_SUBCLASS  uintptr = 1 << 28
	Py_TPFLAGS_DICT_SUBCLASS     uintptr = 1 << 29
	Py_TPFLAGS_BASE_EXC_SUBCLASS uintptr = 1 << 30
	Py_TPFLAGS_TYPE_SUBCLASS     uintptr = 1 << 31
)

// func (p *PythonLib) GetTypeString(obj uintptr) string {
// 	pcfg := p.CTags.PyStructs["PyObject"]
// }
