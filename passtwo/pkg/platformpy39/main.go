package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	pkg "github.com/richinsley/kindalib/genctags/passtwo/pkg/platformpy39/pkg"
	kindalib "github.com/richinsley/kindalib/pkg"
)

func main() {
	k := os.Args[1]

	cwd, err := os.Getwd()
	if err != nil {
		// Handle the error if there is one
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	// where are we going to create the ctags folder sturcture
	// default to the pkg folder
	outputFolder := path.Join(cwd, "..", "..", "..", "..", "pkg", "platform_ctags", runtime.GOOS)

	envpath := path.Join(outputFolder, fmt.Sprintf("ctags-%s.json", k))
	PyCtags := kindalib.PyCtags{}
	// create a new decoder from the file and decode the json
	file, err := os.Open(envpath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&PyCtags)
	if err != nil {
		fmt.Printf("Error decoding json: %v\n", err)
		return
	}
	pkg.PopulateStructs(&PyCtags)

	// remove the original file
	os.Remove(envpath)

	// write the json back to the file
	file, err = os.Create(envpath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(PyCtags)
	if err != nil {
		fmt.Printf("Error encoding json: %v\n", err)
		return
	}
}
