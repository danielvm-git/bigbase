package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

var DistFS fs.FS

func init() {
	var err error
	DistFS, err = fs.Sub(distFS, "dist")
	if err != nil {
		panic("admin embed: " + err.Error())
	}
}
