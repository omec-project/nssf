// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2025 Canonical Ltd.
// SPDX-License-Identifier: Apache-2.0

package polling

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/nfConfigApi"
)

func startPoller(t *testing.T, handler http.HandlerFunc) <-chan []models.PlmnId {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	ch := make(chan []models.PlmnId, 4)
	poller, err := newPoller(srv.URL, ch, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); poller.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return ch
}

func withFactoryConfig(t *testing.T) {
	t.Helper()
	orig := factory.NssfConfig
	t.Cleanup(func() { factory.NssfConfig = orig })
	factory.ConfigLock.Lock()
	factory.NssfConfig = factory.Config{Configuration: &factory.Configuration{SupportedNssaiInPlmnList: make(factory.SupportedNssaiInPlmn)}}
	factory.ConfigLock.Unlock()
}

func TestPolling_DeliversPlmnListAndUpdatesSupportedNssai(t *testing.T) {
	withFactoryConfig(t)
	ch := startPoller(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pollingPath {
			t.Errorf("polled %s, want %s", r.URL.Path, pollingPath)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`[{"plmnId":{"mcc":"208","mnc":"93"},"sNssaiList":[{"sst":1,"sd":"010203"}]}]`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	})

	select {
	case got := <-ch:
		if len(got) != 1 || got[0].Mcc != "208" || got[0].Mnc != "93" {
			t.Errorf("got %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no config delivered")
	}
	select {
	case got := <-ch:
		t.Errorf("unchanged config delivered again: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
	factory.ConfigLock.Lock()
	defer factory.ConfigLock.Unlock()
	if n := len(factory.NssfConfig.Configuration.SupportedNssaiInPlmnList); n != 1 {
		t.Errorf("SupportedNssaiInPlmnList has %d entries, want 1", n)
	}
}

func TestPolling_EmptyInitialConfigIsNotDelivered(t *testing.T) {
	withFactoryConfig(t)
	var polls atomic.Int32
	ch := startPoller(t, func(w http.ResponseWriter, r *http.Request) {
		polls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`[]`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	})

	deadline := time.Now().Add(2 * time.Second)
	for polls.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if got := polls.Load(); got < 3 {
		t.Fatalf("the poller made %d requests, want at least 3: the empty first config was never put to the test", got)
	}
	select {
	case got := <-ch:
		t.Errorf("empty config delivered before any PLMN was configured: %+v", got)
	default:
	}
}

func TestHandlePolledPlmnSnssaiConfig_SendCancelled_ThenNotRecordedAndRetryable(t *testing.T) {
	withFactoryConfig(t)
	plmn := nfConfigApi.PlmnSnssai{PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"}}
	ch := make(chan []models.PlmnId) // unbuffered and nobody receiving
	handler := plmnSnssaiHandler{currentPlmnConfig: []models.PlmnId{}, plmnConfigChan: ch}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := handler.handlePolledPlmnSnssaiConfig(ctx, []nfConfigApi.PlmnSnssai{plmn}); err == nil {
		t.Fatal("expected an error when the send is cancelled")
	}
	if len(handler.currentPlmnConfig) != 0 {
		t.Errorf("PLMN config recorded before delivery succeeded: %+v", handler.currentPlmnConfig)
	}
	if n := len(factory.NssfConfig.Configuration.SupportedNssaiInPlmnList); n != 0 {
		t.Errorf("SupportedNssaiInPlmnList updated although delivery failed: %d entries", n)
	}
}

func TestHandlePolledPlmnSnssaiConfig_ExpectChannelNotToBeUpdated(t *testing.T) {
	plmn1 := nfConfigApi.PlmnSnssai{
		PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
		SNssaiList: []nfConfigApi.Snssai{
			{Sst: 1, Sd: new("010203")},
		},
	}
	plmn2 := nfConfigApi.PlmnSnssai{
		PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
		SNssaiList: []nfConfigApi.Snssai{
			{Sst: 2, Sd: new("112233")},
		},
	}

	supportedNssai1 := factory.SupportedNssaiInPlmn{
		{Mcc: "001", Mnc: "01"}: {{Sst: 1, Sd: "010203"}: struct{}{}},
	}

	tests := []struct {
		name                        string
		initialPlmnSnssaiConfig     []nfConfigApi.PlmnSnssai
		initialPlmnConfig           []models.PlmnId
		initialSupportedNssaiConfig factory.SupportedNssaiInPlmn
		input                       []nfConfigApi.PlmnSnssai
		expectedCurrentPlmnConfig   []models.PlmnId
		expectedSupportedNssaiCount int
	}{
		{
			name:                        "Same config, factory config not to be updated",
			initialPlmnSnssaiConfig:     []nfConfigApi.PlmnSnssai{plmn1},
			initialPlmnConfig:           []models.PlmnId{{Mcc: "001", Mnc: "01"}},
			initialSupportedNssaiConfig: supportedNssai1,
			input:                       []nfConfigApi.PlmnSnssai{plmn1},
			expectedCurrentPlmnConfig:   []models.PlmnId{{Mcc: "001", Mnc: "01"}},
			expectedSupportedNssaiCount: 1,
		},
		{
			name:                        "Initial config is empty, new config empty, expect factory config not to be updated",
			initialPlmnSnssaiConfig:     []nfConfigApi.PlmnSnssai{},
			initialPlmnConfig:           []models.PlmnId{},
			input:                       []nfConfigApi.PlmnSnssai{},
			expectedCurrentPlmnConfig:   []models.PlmnId{},
			expectedSupportedNssaiCount: 0,
		},
		{
			name:                        "New config, same PLMN, different S-NSSAI, expect factory config updated",
			initialPlmnSnssaiConfig:     []nfConfigApi.PlmnSnssai{plmn1},
			initialPlmnConfig:           []models.PlmnId{{Mcc: "001", Mnc: "01"}},
			input:                       []nfConfigApi.PlmnSnssai{plmn2},
			expectedCurrentPlmnConfig:   []models.PlmnId{{Mcc: "001", Mnc: "01"}},
			expectedSupportedNssaiCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFactoryConfig := factory.NssfConfig
			defer func() {
				factory.NssfConfig = originalFactoryConfig
			}()
			factory.NssfConfig = factory.Config{
				Configuration: &factory.Configuration{
					SupportedNssaiInPlmnList: tc.initialSupportedNssaiConfig,
				},
			}

			ch := make(chan []models.PlmnId, 1)
			handler := plmnSnssaiHandler{currentPlmnConfig: tc.initialPlmnConfig, plmnConfigChan: ch}

			if err := handler.handlePolledPlmnSnssaiConfig(context.Background(), tc.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			select {
			case updated := <-ch:
				t.Errorf("Unexpected channel send: %+v", updated)
			case <-time.After(100 * time.Millisecond):
				// Expected
			}

			if !reflect.DeepEqual(handler.currentPlmnConfig, tc.expectedCurrentPlmnConfig) {
				t.Errorf("Expected current PLMN config: %+v, got: %+v",
					tc.expectedCurrentPlmnConfig, handler.currentPlmnConfig)
			}

			if len(factory.NssfConfig.Configuration.SupportedNssaiInPlmnList) != tc.expectedSupportedNssaiCount {
				t.Errorf("Expected SupportedNssaiInPlmnList to have %d entries, got %d",
					tc.expectedSupportedNssaiCount,
					len(factory.NssfConfig.Configuration.SupportedNssaiInPlmnList))
			}
		})
	}
}

func TestHandlePolledPlmnSnssaiConfig_ExpectChannelUpdate(t *testing.T) {
	plmn1 := nfConfigApi.PlmnSnssai{
		PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
		SNssaiList: []nfConfigApi.Snssai{
			{Sst: 1, Sd: new("010203")},
		},
	}
	plmn2 := nfConfigApi.PlmnSnssai{
		PlmnId: nfConfigApi.PlmnId{Mcc: "002", Mnc: "02"},
		SNssaiList: []nfConfigApi.Snssai{
			{Sst: 2, Sd: new("112233")},
		},
	}

	tests := []struct {
		name                        string
		initialPlmnSnssaiConfig     []nfConfigApi.PlmnSnssai
		initialPlmnConfig           []models.PlmnId
		input                       []nfConfigApi.PlmnSnssai
		expectedCurrentPlmnConfig   []models.PlmnId
		expectedSupportedNssaiCount int
	}{
		{
			name:                        "Initial config is empty, new config, expect channel and factory config update",
			initialPlmnSnssaiConfig:     []nfConfigApi.PlmnSnssai{},
			initialPlmnConfig:           []models.PlmnId{},
			input:                       []nfConfigApi.PlmnSnssai{plmn1, plmn2},
			expectedCurrentPlmnConfig:   []models.PlmnId{{Mcc: "001", Mnc: "01"}, {Mcc: "002", Mnc: "02"}},
			expectedSupportedNssaiCount: 2,
		},
		{
			name:                        "New config, change in PLMN and S-NSSAI, expect channel and factory config update",
			initialPlmnSnssaiConfig:     []nfConfigApi.PlmnSnssai{plmn1},
			initialPlmnConfig:           []models.PlmnId{{Mcc: "001", Mnc: "01"}},
			input:                       []nfConfigApi.PlmnSnssai{plmn2},
			expectedCurrentPlmnConfig:   []models.PlmnId{{Mcc: "002", Mnc: "02"}},
			expectedSupportedNssaiCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan []models.PlmnId, 1)

			originalFactoryConfig := factory.NssfConfig
			defer func() {
				factory.NssfConfig = originalFactoryConfig
			}()
			factory.NssfConfig = factory.Config{
				Configuration: &factory.Configuration{
					SupportedNssaiInPlmnList: make(factory.SupportedNssaiInPlmn),
				},
			}

			handler := plmnSnssaiHandler{currentPlmnConfig: tc.initialPlmnConfig, plmnConfigChan: ch}

			if err := handler.handlePolledPlmnSnssaiConfig(context.Background(), tc.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			select {
			case updated := <-ch:
				if !reflect.DeepEqual(updated, tc.expectedCurrentPlmnConfig) {
					t.Errorf("Wrong config sent on channel.\nExpected: %+v\nGot: %+v", tc.expectedCurrentPlmnConfig, updated)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("Expected update to be sent to channel but none received")
			}

			if !reflect.DeepEqual(handler.currentPlmnConfig, tc.expectedCurrentPlmnConfig) {
				t.Errorf("Expected current PLMN config: %+v, got: %+v",
					tc.expectedCurrentPlmnConfig, handler.currentPlmnConfig)
			}

			if len(factory.NssfConfig.Configuration.SupportedNssaiInPlmnList) != tc.expectedSupportedNssaiCount {
				t.Errorf("Expected SupportedNssaiInPlmnList to have %d entries, got %d",
					tc.expectedSupportedNssaiCount,
					len(factory.NssfConfig.Configuration.SupportedNssaiInPlmnList))
			}
		})
	}
}

func TestConvertPlmnSnssaiList(t *testing.T) {
	sdPtr := new("010203")

	tests := []struct {
		name                   string
		input                  []nfConfigApi.PlmnSnssai
		expectedPlmnList       []models.PlmnId
		expectedSupportedNssai factory.SupportedNssaiInPlmn
	}{
		{
			name:                   "Empty input",
			input:                  []nfConfigApi.PlmnSnssai{},
			expectedPlmnList:       []models.PlmnId{},
			expectedSupportedNssai: factory.SupportedNssaiInPlmn{},
		},
		{
			name: "Single PLMN with one SNSSAI (with SD)",
			input: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					SNssaiList: []nfConfigApi.Snssai{
						{Sst: 1, Sd: sdPtr},
					},
				},
			},
			expectedPlmnList: []models.PlmnId{
				{Mcc: "001", Mnc: "01"},
			},
			expectedSupportedNssai: factory.SupportedNssaiInPlmn{
				{Mcc: "001", Mnc: "01"}: {
					{Sst: 1, Sd: "010203"}: struct{}{},
				},
			},
		},
		{
			name: "Single PLMN with one SNSSAI (nil SD)",
			input: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					SNssaiList: []nfConfigApi.Snssai{
						{Sst: 2, Sd: nil},
					},
				},
			},
			expectedPlmnList: []models.PlmnId{
				{Mcc: "001", Mnc: "01"},
			},
			expectedSupportedNssai: factory.SupportedNssaiInPlmn{
				{Mcc: "001", Mnc: "01"}: {
					{Sst: 2, Sd: ""}: struct{}{},
				},
			},
		},
		{
			name: "Multiple PLMNs and SNSSAIs (mixed SD presence)",
			input: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					SNssaiList: []nfConfigApi.Snssai{
						{Sst: 1, Sd: sdPtr},
						{Sst: 2, Sd: nil},
					},
				},
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "002", Mnc: "02"},
					SNssaiList: []nfConfigApi.Snssai{
						{Sst: 3, Sd: new("112233")},
					},
				},
			},
			expectedPlmnList: []models.PlmnId{
				{Mcc: "001", Mnc: "01"},
				{Mcc: "002", Mnc: "02"},
			},
			expectedSupportedNssai: factory.SupportedNssaiInPlmn{
				{Mcc: "001", Mnc: "01"}: {
					{Sst: 1, Sd: "010203"}: struct{}{},
					{Sst: 2, Sd: ""}:       struct{}{},
				},
				{Mcc: "002", Mnc: "02"}: {
					{Sst: 3, Sd: "112233"}: struct{}{},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPlmnList, gotSupported := convertPlmnSnssaiList(tc.input)

			if !reflect.DeepEqual(gotPlmnList, tc.expectedPlmnList) {
				t.Errorf("Expected PLMN list: %+v, got: %+v", tc.expectedPlmnList, gotPlmnList)
			}
			if !reflect.DeepEqual(gotSupported, tc.expectedSupportedNssai) {
				t.Errorf("Expected Supported NSSAI map: %+v, got: %+v", tc.expectedSupportedNssai, gotSupported)
			}
		})
	}
}
