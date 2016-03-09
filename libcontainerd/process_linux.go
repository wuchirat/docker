package libcontainerd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Sirupsen/logrus"
	containerd "github.com/docker/containerd/api/grpc/types"
	"github.com/docker/docker/pkg/ioutils"
	"golang.org/x/net/context"
)

var fdNames = map[int]string{
	syscall.Stdin:  "stdin",
	syscall.Stdout: "stdout",
	syscall.Stderr: "stderr",
}

func (p *process) openFifos(terminal bool) (*IOPipe, error) {
	bundleDir := p.dir
	if err := os.MkdirAll(bundleDir, 0700); err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		f := p.fifo(i)
		if err := syscall.Mkfifo(f, 0700); err != nil && !os.IsExist(err) {
			return nil, fmt.Errorf("mkfifo: %s %v", f, err)
		}
	}

	io := &IOPipe{}
	// FIXME: O_RDWR? open one-sided in goroutines?
	stdinf, err := os.OpenFile(p.fifo(syscall.Stdin), syscall.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	io.Stdout = openReaderFromFifo(p.fifo(syscall.Stdout))
	if !terminal {
		io.Stderr = openReaderFromFifo(p.fifo(syscall.Stderr))
	} else {
		io.Stderr = emptyReader{}
	}

	io.Stdin = ioutils.NewWriteCloserWrapper(stdinf, func() error {
		stdinf.Close()
		_, err := p.client.remote.apiClient.UpdateProcess(context.Background(), &containerd.UpdateProcessRequest{
			Id:         p.id,
			Pid:        p.processID,
			CloseStdin: true,
		})
		return err
	})

	return io, nil
}

type emptyReader struct{}

func (r emptyReader) Read(b []byte) (int, error) {
	return 0, io.EOF
}

func openReaderFromFifo(fn string) io.Reader {
	r, w := io.Pipe()
	go func() {
		logrus.Debugf(">open", fn)
		stdoutf, err := os.OpenFile(fn, syscall.O_RDONLY, 0)
		logrus.Debugf("<open", fn)
		if err != nil {
			r.CloseWithError(err)
		}
		if _, err := io.Copy(w, stdoutf); err != nil {
			r.CloseWithError(err)
		}
		w.Close()
	}()
	return r
}

func (p *process) fifo(index int) string {
	return filepath.Join(p.dir, p.processID+"-"+fdNames[index])
}
