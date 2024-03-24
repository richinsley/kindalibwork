package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	kinda "github.com/richinsley/kinda/pkg"
	pkg "github.com/richinsley/kindalib/genctags/passtwo/pkg"
	kindalib "github.com/richinsley/kindalib/pkg"
)

func main() {

	cwd, err := os.Getwd()
	if err != nil {
		// Handle the error if there is one
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	// where are we going to create the ctags folder sturcture
	var outputFolder string
	if len(os.Args) != 2 {
		// default to the pkg folder
		outputFolder = path.Join(cwd, "..", "..", "pkg", "platform_ctags", runtime.GOOS)
	} else {
		outputFolder = path.Join(os.Args[1], "platform_ctags", runtime.GOOS)
	}

	// Specify the binary folder to place micromamba in
	rootDirectory := filepath.Join(cwd, "micromamba")
	fmt.Println("Creating Kinda repos at: ", rootDirectory)
	envs := make(map[string]*kinda.Environment)
	envs["39"], err = kinda.CreateEnvironment("myenv39", rootDirectory, "3.9", "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	envs["310"], err = kinda.CreateEnvironment("myenv310", rootDirectory, "3.10", "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	envs["311"], err = kinda.CreateEnvironment("myenv311", rootDirectory, "3.11", "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	envs["312"], err = kinda.CreateEnvironment("myenv312", rootDirectory, "3.12", "conda-forge")
	if err != nil {
		fmt.Printf("Error creating environment: %v\n", err)
		return
	}

	// deserialize the structs generated by the first pass
	for k, _ := range envs {
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
		pkg.GenStructOffsets(k, &PyCtags)
	}
}
