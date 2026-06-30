//go:build !windows

package patchright

import "syscall"

var defaultSysProcAttr = &syscall.SysProcAttr{}

// for WritableStream.Copy
const defaultCopyBufSize = 1024 * 1024
