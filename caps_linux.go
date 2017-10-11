// +build linux
// Copyright (C) 2017 Adam Hose adis@blad.is
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"syscall"
)

/*
#cgo LDFLAGS: -lcap
#include <sys/capability.h>
#include <errno.h>

static int dropAllCaps(void) {
    cap_t state;

    state = cap_init();
    if (!state) {
        cap_free(state);
    }


    if (cap_clear(state) < 0) {
        cap_free(state);
        return errno;
    }

    if (cap_set_proc(state) == -1) {
        cap_free(state);
        return errno;
    }

    cap_free(state);
    return 0;
}
*/
import "C"

// DropAllCaps - Drop all posix capabilities
func DropAllCaps() (err error) {
	errno := C.dropAllCaps()
	if errno != 0 {
		return syscall.Errno(errno)
	}
	return
}
