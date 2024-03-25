package pkg

import (
	"embed"
	"encoding/json"
	"path"
	"runtime"

	kinda "github.com/richinsley/kinda/pkg"
)

//go:embed platform_ctags/darwin/* platform_ctags/windows/* platform_ctags/linux/*
var EmbeddedCtags embed.FS

func GetPlatformCtags(pyversion string) (*PyCtags, error) {
	v, err := kinda.ParseVersion(pyversion)
	if err != nil {
		return nil, err
	}
	name := "ctags-" + v.MinorStringCompact() + ".json"
	rpath := path.Join("platform_ctags", runtime.GOOS, name)
	rfile, err := EmbeddedCtags.ReadFile(rpath)
	if err != nil {
		return nil, err
	}

	// parse the ctags json
	var ctags PyCtags
	err = json.Unmarshal(rfile, &ctags)
	if err != nil {
		return nil, err
	}

	return &ctags, nil
}
