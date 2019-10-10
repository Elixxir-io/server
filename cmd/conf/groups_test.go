////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/utils"
	"gopkg.in/yaml.v2"
	"reflect"
	"testing"
)

var prime = large.NewInt(int64(17))
var generator = large.NewInt(int64(4))

var ExpectedGroup = cyclic.NewGroup(prime, generator)

var ExpectedGroups = Groups{
	CMix: map[string]string{
		"prime":      "0x11",
		"smallprime": "0x0B",
		"generator":  "0x04",
	},
	E2E: map[string]string{
		"prime":      "0x11",
		"smallprime": "0x0B",
		"generator":  "0x04",
	},
}

// This test checks that unmarshalling the groups.yaml file
// is equal to the expected groups object.
func TestGroups_UnmarshallingFileEqualsExpected(t *testing.T) {

	actual := Params{}
	buf, _ := utils.ReadFile("./params.yaml")

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(ExpectedGroups, actual.Groups) {
		t.Errorf("Groups object did not match expected value")
	}

}

// This test checks that the CMIX fingerprint
// matches the actualy cyclic group object
func TestGroup_GetCMixValidFingerprint(t *testing.T) {
	actual := Params{}
	buf, _ := utils.ReadFile("./params.yaml")

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	fp := actual.Groups.GetCMix().GetFingerprint()
	if fp != ExpectedGroup.GetFingerprint() {
		t.Errorf("CMix finger print did not match expected value")
	}
}

// This test checks that the E2E fingerprint
// matches the actualy cyclic group object
func TestGroup_GetE2EValidFingerprint(t *testing.T) {
	actual := Params{}
	buf, _ := utils.ReadFile("./params.yaml")

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	fp := actual.Groups.GetE2E().GetFingerprint()
	if fp != ExpectedGroup.GetFingerprint() {
		t.Errorf("E2E finger print did not match expected value")
	}
}
