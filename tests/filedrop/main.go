package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ebitengine/purego"
	kinda "github.com/richinsley/kinda/pkg"
	pylib "github.com/richinsley/kindalib/pkg"
)

// example_module.file_dropped(wx.ICON_INFORMATION)
var quote_str string = `
import example_module
import sys
from PyQt5.QtWidgets import QApplication, QWidget, QVBoxLayout, QLabel
from PyQt5.QtCore import Qt

class DropArea(QWidget):
    def __init__(self):
        super().__init__()
        self.initUI()

    def initUI(self):
        layout = QVBoxLayout()
        self.label = QLabel('Drop files here')
        self.label.setAlignment(Qt.AlignCenter)
        layout.addWidget(self.label)
        self.setLayout(layout)

        self.setAcceptDrops(True)
        self.setWindowTitle('Drag and Drop Files')
        self.setGeometry(300, 300, 300, 200)

    def dragEnterEvent(self, event):
        if event.mimeData().hasUrls():
            event.accept()
        else:
            event.ignore()

    def dropEvent(self, event):
        files = [u.toLocalFile() for u in event.mimeData().urls()]
        for f in files:
            print(f'Dropped file: {f}')
            # self.label.setText(f'Dropped: {f}')
            example_module.file_dropped(f)

app = QApplication(sys.argv)
ex = DropArea()
ex.show()
app.exec_()
`
var lib pylib.IPythonLib

func file_dropped_cb(p1 uintptr, p2 uintptr) uintptr {
	fmt.Println("File Dropped!")
	// Ensure the result is a Python string
	// PyUnicode_Check is a MACRO.  We need to create these manually
	if lib.Invoke("PyUnicode_Check", p2) == 0 {
		fmt.Println("Not a string")
		return lib.GetPyNone()
	}

	s := lib.Invoke("PyUnicode_AsUTF8", p2)
	fmt.Printf("File dropped: %s\n", lib.PtrToStr(s))
	return lib.GetPyNone()
}

// func init() {
// 	// Run main on the startup thread to satisfy the requirement
// 	// that Main runs on that thread.
// 	runtime.LockOSThread()
// }

func main() {
	// Specify the binary folder to place micromamba in
	cwd, _ := os.Getwd()
	rootDirectory := filepath.Join(cwd, "..", "micromamba")
	fmt.Println("Creating Kinda repo at: ", rootDirectory)
	version := "3.10"
	env, err := kinda.CreateEnvironment("myenv"+version, rootDirectory, version, "conda-forge", kinda.ShowVerbose)
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	// install wxpython into the environment
	// pip install PyQt5
	env.PipInstallPackage("PyQt5", "", "", true, kinda.ShowNothing)

	fmt.Printf("Created environment: %s\n", env.Name)

	// test create a library
	lib, err = pylib.NewPythonLib(env)
	if err != nil {
		fmt.Printf("Error creating library: %v\n", err)
		return
	}

	lib.Init("file_drop")

	// create the method object
	meth := lib.NewPyMethodDefArray(1)
	fmt.Printf("Created method object with size: %d\n", meth.PyConfig.Size)

	// create a callback function
	var f func(uintptr, uintptr) uintptr = file_dropped_cb
	goCallback := purego.NewCallback(f)

	// set the buttonClickPtr to meth.SetMethodDef
	meth.SetMethodDef(0, "file_dropped", uintptr(goCallback), pylib.METH_VARARGS)

	// Create the moduledef object and create the module from that def
	moduledef := lib.NewPyModuleDef("example_module", "Example module with Go callback", &meth)

	module := lib.Invoke("PyModule_Create2", moduledef.GetBuffer(), 3)
	fmt.Printf("Created module: %v\n", module)

	// Add the module to sys.modules
	sys_modules := lib.Invoke("PyImport_GetModuleDict")
	lib.Invoke("PyDict_SetItemString", sys_modules, lib.StrToPtr("example_module"), module)
	lib.Invoke("Py_DecRef", module)

	// Run the Python code
	lib.Invoke("PyRun_SimpleString", lib.StrToPtr(quote_str))

	// purego.UnrefCallbackFnPtr(&f)

	// free the memory

	fmt.Println("Done")
}
