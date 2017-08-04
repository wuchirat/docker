// +build windows

package lcow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/containerd/continuity/driver"
	"github.com/sirupsen/logrus"
)

var _ driver.File = &lcowfile{}

type lcowfile struct {
	*os.File
	perm         os.FileMode
	flushOnClose bool
	guestPath    string
	fs           *lcowfs
}

func (f *lcowfile) Close() error {
	var err error
	if f.flushOnClose {
		permStr := strconv.FormatUint(uint64(f.perm), 8)
		cmd := fmt.Sprintf("%s %s %s %s", remotefsCmd, writeFileCmd, f.guestPath, permStr)
		err = f.fs.runProcess(cmd, f.File, nil, nil)
		if err != nil {
			logrus.Warnf("lcowfile.Close %s %s failed: %s", f.guestPath, permStr, err)
		}
	}

	fileName := f.File.Name()
	if err1 := f.File.Close(); err1 != nil {
		logrus.Warnf("lcowfile.File.Close %s failed: %s", fileName, err1)
		if err == nil {
			err = err1
		}
	}

	if err1 := os.RemoveAll(fileName); err1 != nil {
		logrus.Warnf("lcowfile.RemoveAll %s failed: %s", fileName, err1)
		if err == nil {
			err = err1
		}
	}
	return err
}

func (f *lcowfile) Readdir(n int) ([]os.FileInfo, error) {
	nStr := strconv.FormatInt(int64(n), 10)
	cmd := fmt.Sprintf("%s %s %s %s", remotefsCmd, readDirCmd, f.guestPath, nStr)

	buf := &bytes.Buffer{}
	if err := f.fs.runProcess(cmd, nil, buf, nil); err != nil {
		return nil, err
	}

	var info []fileinfo
	if err := json.Unmarshal(buf.Bytes(), &info); err != nil {
		return nil, nil
	}

	osInfo := make([]os.FileInfo, len(info))
	for i := range info {
		osInfo[i] = &info[i]
	}
	return osInfo, nil
}
