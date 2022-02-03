// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF Testing Utility
 */

package test

import (
	"reflect"

	. "github.com/free5gc/openapi/models"
)

func CheckAuthorizedNetworkSliceInfo(target AuthorizedNetworkSliceInfo, expectList []AuthorizedNetworkSliceInfo) bool {
	for _, expectElement := range expectList {
		if reflect.DeepEqual(target, expectElement) {
			return true
		}
	}
	return false
}
