////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package internal

// firstTimeSignal.go contains the logic for a channel
// that can only be sent to once

import (
	"sync"
)

type FirstTime struct {
	c chan struct{}
	sync.Once
}

// NewFirstTime is a constructor of the FirstTime object
func NewFirstTime() *FirstTime {
	return &FirstTime{
		c:    make(chan struct{}, 1),
		Once: sync.Once{},
	}
}

// Send sends to the structs channel explicitly once
func (ft *FirstTime) Send() {
	ft.Once.Do(func() {
		ft.c <- struct{}{}
	})
}

// Receive either receives from the channel.
func (ft *FirstTime) Receive()  {
	<-ft.c
}
