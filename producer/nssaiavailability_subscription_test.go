// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"testing"

	"github.com/omec-project/nssf/factory"
)

func TestGetUnusedSubscriptionIDLockedNoDuplicates(t *testing.T) {
	factory.ConfigLock.Lock()
	defer factory.ConfigLock.Unlock()
	saved := factory.NssfConfig.Subscriptions
	defer func() { factory.NssfConfig.Subscriptions = saved }()

	// Unsorted list with a gap at the front, as left behind by a delete of ID "1"
	factory.NssfConfig.Subscriptions = []factory.Subscription{
		{SubscriptionId: "2"}, {SubscriptionId: "3"},
	}
	seen := map[string]struct{}{"2": {}, "3": {}}
	for range 5 {
		id, err := getUnusedSubscriptionIDLocked()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate subscription ID %s", id)
		}
		seen[id] = struct{}{}
		factory.NssfConfig.Subscriptions = append(factory.NssfConfig.Subscriptions, factory.Subscription{SubscriptionId: id})
	}
}

func TestGetUnusedSubscriptionIDLockedIgnoresNonNumeric(t *testing.T) {
	factory.ConfigLock.Lock()
	defer factory.ConfigLock.Unlock()
	saved := factory.NssfConfig.Subscriptions
	defer func() { factory.NssfConfig.Subscriptions = saved }()

	factory.NssfConfig.Subscriptions = []factory.Subscription{{SubscriptionId: "abc"}}
	id, err := getUnusedSubscriptionIDLocked()
	if err != nil || id != "1" {
		t.Fatalf("got (%q, %v), want (\"1\", nil)", id, err)
	}
}
