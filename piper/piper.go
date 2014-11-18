package piper

import (
	"errors"
	"io"
	"net"

	"github.com/ORBAT/proxinator/logging"
)

const (
	ReadBufSize = 64 * 1024
)

var proxyLog = logging.New("ConnProxy", nil)

// A Conn is a network connection piped through one or more hosts
type Conn struct {
}

// copy copies from src to dst until either EOF is reached
// on src or an error occurs, and in both cases dst will be closed.
// It returns the number of bytes copied and the first error encountered
// while copying, if any.
//
// A successful copy returns err == nil, not err == EOF.
// Because copy is defined to read from src until EOF, it does
// not treat an EOF from Read as an error to be reported.
func copy(dst io.WriteCloser, src io.Reader) (written int64, err error) {
	buf := make([]byte, ReadBufSize)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}

	if ec := dst.Close(); ec != nil {
		err = ec
	}

	return written, err
}

func Proxy(fst, snd net.Conn) (*Proxy, error) {
	return nil, errors.New("wip")
}
