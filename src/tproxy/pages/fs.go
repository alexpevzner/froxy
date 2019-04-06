//
// http.FileSystem on a top of embedded pages
//
package pages

import (
	"net/http"
	"os"

	assetfs "github.com/elazarl/go-bindata-assetfs"
)

var AssetFS = &assetfs.AssetFS{
	Asset:     Asset,
	AssetDir:  AssetDir,
	AssetInfo: AssetInfo,
}

var FileServer = http.FileServer(&assetFSWrapper{AssetFS})

//
// AssetFS wrapper that substitutes "Page not found" page
// instead of any missed file
//
type assetFSWrapper struct {
	*assetfs.AssetFS
}

func (fs *assetFSWrapper) Open(name string) (http.File, error) {
	file, err := fs.AssetFS.Open(name)
	if err == os.ErrNotExist {
		file, err = fs.AssetFS.Open("/404/index.html")
	}
	return file, err
}
