// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// http.FileSystem on a top of embedded pages

package pages

import (
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
)

var AssetFS = &assetfs.AssetFS{
	Asset:     Asset,
	AssetDir:  AssetDir,
	AssetInfo: AssetInfo,
}

var FileServer = http.FileServer(AssetFS)
