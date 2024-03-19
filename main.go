package main

// https://docs.python.org/3/extending/embedding.html#very-high-level-embedding

/*
func main() {
	py, _ := python.NewPython()

	// Initialize Python interpreter
	py.Invoke("Py_Initialize")

	script := `
import zlib

original_data = b"Hello, World! This is a test string for zlib compression."
compressed_data = zlib.compress(original_data)
print("Compressed data:", compressed_data)

decompressed_data = zlib.decompress(compressed_data)
print("Decompressed data:", decompressed_data.decode('utf-8'))
`
	sptr := python.StrToPtr(script)
	defer python.FreeString(sptr)

	py.Invoke("PyRun_SimpleString", sptr)

	// Finalize the Python interpreter
	py.Invoke("Py_Finalize")
}
*/

func main() {

}

/*
// https://docs.python.org/3/extending/embedding.html#pure-embedding
// https://stackoverflow.com/questions/17532371/pyimport-import-fails-returns-null
func main() {
	py, _ := python.NewPython()

	// Initialize Python interpreter
	py.Invoke("Py_Initialize")

	// depricated since 3.11, but works to enable loading module from paths relative to the program
	// https://stackoverflow.com/questions/13422764/why-does-pyimport-import-fail-to-load-a-module-from-the-current-directory
	// actual configuration should be done with Python Configuratuion:
	// https://docs.python.org/3/c-api/init_config.html#init-python-config
	// unfortunately, we'll have to get involved with a per-version c structure
	py.Invoke("PySys_SetArgv", 0, 0)

	// in the same folder as the executable, we'll have a python file "multiply.py".  We load it by it's
	// name "multiply".  We can do this becuase we called PySys_SetArgv.  The module name MUST be in a python string.
	// We create a python string with "PyUnicode_DecodeFSDefault"
	mpath := "multiply"
	sptr := python.StrToPtr(mpath)
	pName := py.Invoke("PyUnicode_DecodeFSDefault", sptr)

	// use "PyImport_Import" to load the module
	pModule := py.Invoke("PyImport_Import", pName)
	py.Invoke("Py_DecRef", pModule)
	if pModule != 0 {
		// find the multiply function in the module
		funcStr := "multiply"
		fsptr := python.StrToPtr(funcStr)
		pFunc := py.Invoke("PyObject_GetAttrString", pModule, fsptr)

		// make sure the returned function is callable
		if pFunc != 0 && py.Invoke("PyCallable_Check", pFunc) != 0 {
			// create a python Tuple that can hold n values
			tupleCount := 2
			pArgs := py.Invoke("PyTuple_New", uintptr(tupleCount))

			// fill the tuple with n values
			for i := 0; i < tupleCount; i++ {
				pValue := py.Invoke("PyLong_FromLong", uintptr(i+1))
				if pValue == 0 {
					py.Invoke("Py_DecRef", pArgs)
					py.Invoke("Py_DecRef", pModule)
					fmt.Println("Cannot convert argument")
				}
				// pValue reference stolen here:
				py.Invoke("PyTuple_SetItem", pArgs, uintptr(i), pValue)
			}

			// call the multiply function
			pValue := py.Invoke("PyObject_CallObject", pFunc, pArgs)
			py.Invoke("Py_DecRef", pArgs)
			if pValue != 0 {
				v := uint(py.Invoke("PyLong_AsLong", pValue))
				fmt.Printf("Returned value %d\n", v)
			} else {
				py.Invoke("Py_DecRef", pFunc)
				py.Invoke("Py_DecRef", pModule)
				py.Invoke("PyErr_Print")
				os.Exit(1)
			}
		} else {
			fmt.Println("Boo!")
		}
	} else {
		py.Invoke("PyErr_Print")
	}
}
*/
