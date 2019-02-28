//
// I/O utilities
//

package main

import (
	"io"
)

//
// Perform bidirectional data transfer between connections
//
func ioTransferData(conn1, conn2 io.ReadWriteCloser) {
	pairs := []struct{ dst, src io.ReadWriteCloser }{{conn1, conn2}, {conn2, conn1}}
	for _, p := range pairs {
		go func(src, dst io.ReadWriteCloser) {
			io.Copy(dst, src)
			dst.Close()
			src.Close()
		}(p.src, p.dst)
	}
}
