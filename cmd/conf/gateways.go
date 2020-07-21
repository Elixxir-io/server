///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conf

// Contains Gateways config params
type Gateway struct {
	Paths        Paths
	useNodeIp    bool
	advertisedIP string
}
