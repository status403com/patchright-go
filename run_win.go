//go:build windows

package patchright

import "syscall"

var defaultSysProcAttr = &syscall.SysProcAttr{HideWindow: true}

// for WritableStream.Copy
const defaultCopyBufSize = 64 * 1024
