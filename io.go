// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// I/O utilities

package main

import (
	"io"
)

//
// Perform bidirectional data transfer between connections
//
func ioTransferData(env *Env, conn1, conn2 io.ReadWriteCloser) {
	pairs := []struct{ dst, src io.ReadWriteCloser }{{conn1, conn2}, {conn2, conn1}}
	for _, p := range pairs {
		go func(src, dst io.ReadWriteCloser) {
			io.Copy(dst, src)
			dst.Close()
		}(p.src, p.dst)
	}
}
