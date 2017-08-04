// +build windows

package lcow

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"bytes"

	"io/ioutil"

	"github.com/containerd/continuity/driver"
	"github.com/sirupsen/logrus"
)

// TODO: @gupta-ak. These constants are defined in the Microsoft/opengcs repo.
// Once docker switches to the using that repo for opengcs, we can remove these.
const remotefsCmd = "remotefs"

const (
	statCmd           = "stat"
	lstatCmd          = "lstat"
	readlinkCmd       = "readlink"
	mkdirCmd          = "mkdir"
	mkdirAllCmd       = "mkdirall"
	removeCmd         = "remove"
	removeAllCmd      = "removeall"
	linkCmd           = "link"
	symlinkCmd        = "symlink"
	lchmodCmd         = "lchmod"
	lchownCmd         = "lchown"
	mknodCmd          = "mknod"
	mkfifoCmd         = "mkfifo"
	openFileCmd       = "openfile"
	readFileCmd       = "readfile"
	writeFileCmd      = "writefile"
	readDirCmd        = "readdir"
	resolvePathCmd    = "resolvepath"
	extractArchiveCmd = "extractarchive"
	archivePathCmd    = "archivepath"
)

var _ driver.Driver = &lcowfs{}

func (l *lcowfs) Open(path string) (driver.File, error) {
	return l.OpenFile(path, os.O_RDONLY, 0)
}

func (l *lcowfs) OpenFile(path string, flag int, perm os.FileMode) (_ driver.File, err error) {
	flagStr := strconv.FormatInt(int64(flag), 10)
	permStr := strconv.FormatUint(uint64(perm), 8)

	cmd := fmt.Sprintf("%s %s %s %s %s", remotefsCmd, openFileCmd, path, flagStr, permStr)
	if err := l.runProcess(cmd, nil, nil, nil); err != nil {
		return nil, err
	}

	// Assume that we want to follow symlinks.
	fi, err := l.Stat(path)
	if err != nil {
		return nil, err
	}

	// After OpenFile, we know that the file exists in the VM.
	// If it's a regular file, retrieve it
	var file *os.File
	if !fi.IsDir() {
		file, err = ioutil.TempFile("", "remotefs")
		if err != nil {
			return nil, err
		}
		fileName := file.Name()

		cmd = fmt.Sprintf("%s %s %s", remotefsCmd, readFileCmd, path)
		if err := l.runProcess(cmd, nil, file, nil); err != nil {
			file.Close()
			os.RemoveAll(fileName)
			return nil, err
		}

		// We do this so that we can match the flags that were given to us.
		file.Close()
		file, err = os.OpenFile(fileName, flag, perm)
		if err != nil {
			os.RemoveAll(fileName)
			return nil, err
		}
	} else {
		tmpdir, err := ioutil.TempDir("", "remotefs")
		if err != nil {
			return nil, err
		}
		file, err = os.Open(tmpdir)
		if err != nil {
			os.RemoveAll(tmpdir)
			return nil, err
		}
	}

	return &lcowfile{
		File:         file,
		perm:         perm,
		flushOnClose: writeAllowed(flag),
		guestPath:    path,
	}, nil
}

func writeAllowed(flag int) bool {
	isSet := func(bits, mask int) bool {
		return bits&mask == mask
	}
	return isSet(flag, os.O_RDWR) || isSet(flag, os.O_WRONLY)
}

func (l *lcowfs) Readlink(p string) (string, error) {
	logrus.Debugf("removefs.readlink args: %s", p)

	result := &bytes.Buffer{}
	cmd := fmt.Sprintf("%s %s %s", remotefsCmd, readlinkCmd, p)
	if err := l.runProcess(cmd, nil, result, nil); err != nil {
		return "", err
	}
	return result.String(), nil
}

func (l *lcowfs) Mkdir(path string, mode os.FileMode) error {
	return l.mkdir(path, mode, mkdirCmd)
}

func (l *lcowfs) MkdirAll(path string, mode os.FileMode) error {
	return l.mkdir(path, mode, mkdirAllCmd)
}

func (l *lcowfs) mkdir(path string, mode os.FileMode, cmd string) error {
	modeStr := strconv.FormatUint(uint64(mode), 8)
	logrus.Debugf("removefs.%s args: %s %s", cmd, path, modeStr)
	procCommand := fmt.Sprintf("%s %s %s %s", remotefsCmd, cmd, path, modeStr)
	return l.runProcess(procCommand, nil, nil, nil)
}

func (l *lcowfs) Remove(path string) error {
	return l.remove(path, removeCmd)
}

func (l *lcowfs) RemoveAll(path string) error {
	return l.remove(path, removeAllCmd)
}

func (l *lcowfs) remove(path string, cmd string) error {
	logrus.Debugf("removefs.%s args: %s", cmd, path)
	procCommand := fmt.Sprintf("%s %s %s", remotefsCmd, cmd, path)
	return l.runProcess(procCommand, nil, nil, nil)
}

func (l *lcowfs) Link(oldname, newname string) error {
	return l.link(oldname, newname, linkCmd)
}

func (l *lcowfs) Symlink(oldname, newname string) error {
	return l.link(oldname, newname, symlinkCmd)
}

func (l *lcowfs) link(oldname, newname, cmd string) error {
	logrus.Debugf("removefs.%s args: %s %s", cmd, oldname, newname)
	procCommand := fmt.Sprintf("%s %s %s %s", remotefsCmd, cmd, oldname, newname)
	return l.runProcess(procCommand, nil, nil, nil)
}

func (l *lcowfs) Lchown(name string, uid, gid int64) error {
	uidStr := strconv.FormatInt(uid, 10)
	gidStr := strconv.FormatInt(gid, 10)

	logrus.Debugf("removefs.lchown args: %s %s %s", name, uidStr, gidStr)
	cmd := fmt.Sprintf("%s %s %s %s %s", remotefsCmd, lchownCmd, name, uidStr, gidStr)
	return l.runProcess(cmd, nil, nil, nil)
}

// Lchmod changes the mode of an file not following symlinks.
func (l *lcowfs) Lchmod(path string, mode os.FileMode) error {
	modeStr := strconv.FormatUint(uint64(mode), 8)
	logrus.Debugf("removefs.lchmod args: %s %s", path, modeStr)
	cmd := fmt.Sprintf("%s %s %s %s", remotefsCmd, lchmodCmd, path, modeStr)
	return l.runProcess(cmd, nil, nil, nil)
}

func (l *lcowfs) Mknod(path string, mode os.FileMode, major, minor int) error {
	modeStr := strconv.FormatUint(uint64(mode), 8)
	majorStr := strconv.FormatUint(uint64(major), 10)
	minorStr := strconv.FormatUint(uint64(minor), 10)

	logrus.Debugf("removefs.mknod args: %s %s %s %s", path, modeStr, majorStr, minorStr)
	cmd := fmt.Sprintf("%s %s %s %s %s %s", remotefsCmd, mknodCmd, path, modeStr, majorStr, minorStr)
	return l.runProcess(cmd, nil, nil, nil)
}

func (l *lcowfs) Mkfifo(path string, mode os.FileMode) error {
	modeStr := strconv.FormatUint(uint64(mode), 8)
	logrus.Debugf("removefs.mkfifo args: %s %s", path, modeStr)
	cmd := fmt.Sprintf("%s %s %s %s", remotefsCmd, mkfifoCmd, path, modeStr)
	return l.runProcess(cmd, nil, nil, nil)
}

func (l *lcowfs) Stat(p string) (os.FileInfo, error) {
	return l.stat(p, "stat")
}

func (l *lcowfs) Lstat(p string) (os.FileInfo, error) {
	return l.stat(p, "lstat")
}

func (l *lcowfs) stat(path string, cmd string) (os.FileInfo, error) {
	logrus.Debugf("remotefs.stat inputs: %s %s", cmd, path)

	output := &bytes.Buffer{}
	err := l.runProcess(fmt.Sprintf("remotefs %s %s", cmd, path), nil, output, nil)
	if err != nil {
		return nil, err
	}

	var fi fileinfo
	if err := json.Unmarshal(output.Bytes(), &fi); err != nil {
		return nil, err
	}

	logrus.Debugf("remotefs.stat success. got: %v\n", fi)
	return &fi, nil
}

// TODO: @gupta-ak. Remove this struct once the switch to Mircrosoft/opengcs is done.
// This truct is defined there.
type fileinfo struct {
	NameVar    string
	SizeVar    int64
	ModeVar    os.FileMode
	ModTimeVar int64
	IsDirVar   bool
}

func (f *fileinfo) Name() string       { return f.NameVar }
func (f *fileinfo) Size() int64        { return f.SizeVar }
func (f *fileinfo) Mode() os.FileMode  { return f.ModeVar }
func (f *fileinfo) ModTime() time.Time { return time.Unix(0, f.ModTimeVar) }
func (f *fileinfo) IsDir() bool        { return f.IsDirVar }
func (f *fileinfo) Sys() interface{}   { return nil }
