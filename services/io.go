///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package services

import (
	"sync/atomic"
)

type IO_Notify chan Chunk

type moduleInput struct {
	input  IO_Notify
	isOpen *uint32
}

func (mi *moduleInput) closeInput() {
	iClose := atomic.CompareAndSwapUint32(mi.isOpen, 1, 0)
	if iClose {
		close(mi.input)
	}
}

func (mi *moduleInput) open(size uint32) {
	open := uint32(1)
	mi.isOpen = &open
	mi.input = make(IO_Notify, size)
}
