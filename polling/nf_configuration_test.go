// SPDX-FileCopyrightText: 2025 Canonical Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
/*
 * NF Polling Unit Tests
 *
 */

package polling

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/nfConfigApi"
)

func startTestPollingService(ctx context.Context, webuiURI string, plmnConfigChan chan<- []models.PlmnId) (context.CancelFunc, <-chan struct{}) {
	testCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		StartPollingService(testCtx, webuiURI, plmnConfigChan)
	}()
	return cancel, done
}

func waitForPollingServiceStop(t *testing.T, cancel context.CancelFunc, done <-chan struct{}) {
	t.Helper()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for polling service to stop")
	}
}

func waitForPollingCondition(t *testing.T, timeout time.Duration, condition func() bool, failureMessage string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !condition() {
		t.Fatal(failureMessage)
	}
}

func TestStartPollingService_Success(t *testing.T) {
	originalFetchPlmnConfig := fetchPlmnConfig
	originalFactoryConfig := factory.NssfConfig
	pollingChan := make(chan []models.PlmnId, 1)
	var cancel context.CancelFunc
	var done <-chan struct{}
	defer func() {
		waitForPollingServiceStop(t, cancel, done)
		fetchPlmnConfig = originalFetchPlmnConfig
		factory.NssfConfig = originalFactoryConfig
	}()

	factory.NssfConfig = factory.Config{
		Configuration: &factory.Configuration{
			SupportedNssaiInPlmnList: make(factory.SupportedNssaiInPlmn),
		},
	}

	fetchedConfig := []nfConfigApi.PlmnSnssai{
		{
			PlmnId:     nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
			SNssaiList: []nfConfigApi.Snssai{{Sst: 1, Sd: nil}},
		},
	}

	expectedPlmn := []models.PlmnId{{Mcc: "001", Mnc: "01"}}
	expectedSupportedNssai := factory.SupportedNssaiInPlmn{
		{Mcc: "001", Mnc: "01"}: {{Sst: 1, Sd: ""}: struct{}{}},
	}

	fetchPlmnConfig = func(poller *nfConfigPoller, pollingEndpoint string) ([]nfConfigApi.PlmnSnssai, error) {
		return fetchedConfig, nil
	}
	cancel, done = startTestPollingService(t.Context(), "http://dummy", pollingChan)

	select {
	case result := <-pollingChan:
		if !reflect.DeepEqual(result, expectedPlmn) {
			t.Errorf("Expected %+v, got %+v", expectedPlmn, result)
		}
	case <-time.After(initialPollingInterval + 200*time.Millisecond):
		t.Errorf("Timeout waiting for PLMN config")
	}

	waitForPollingServiceStop(t, cancel, done)
	cancel = func() {}

	if !reflect.DeepEqual(expectedSupportedNssai, factory.NssfConfig.Configuration.SupportedNssaiInPlmnList) {
		t.Errorf("Expected %+v, got %+v", expectedSupportedNssai, factory.NssfConfig.Configuration.SupportedNssaiInPlmnList)
	}
}

func TestStartPollingService_RetryAfterFailure(t *testing.T) {
	originalFetchPlmnConfig := fetchPlmnConfig
	originalFactoryConfig := factory.NssfConfig
	plmnChan := make(chan []models.PlmnId, 1)
	var cancel context.CancelFunc
	var done <-chan struct{}

	defer func() {
		waitForPollingServiceStop(t, cancel, done)
		fetchPlmnConfig = originalFetchPlmnConfig
		factory.NssfConfig = originalFactoryConfig
	}()
	factory.NssfConfig = factory.Config{
		Configuration: &factory.Configuration{
			SupportedNssaiInPlmnList: make(factory.SupportedNssaiInPlmn),
		},
	}

	var callCount atomic.Int32
	fetchPlmnConfig = func(poller *nfConfigPoller, pollingEndpoint string) ([]nfConfigApi.PlmnSnssai, error) {
		callCount.Add(1)
		return nil, errors.New("mock failure")
	}
	cancel, done = startTestPollingService(context.Background(), "http://dummy", plmnChan)

	waitForPollingCondition(t, 4*initialPollingInterval+time.Second, func() bool {
		return callCount.Load() >= 2
	}, "expected to retry after failure")
	waitForPollingServiceStop(t, cancel, done)
	cancel = func() {}

	if callCount.Load() < 2 {
		t.Error("Expected to retry after failure")
	}
	t.Logf("Tried %v times", callCount.Load())
}

func TestHandlePolledPlmnSnssaiConfig_ExpectChannelNotToBeUpdated(t *testing.T) {
	plmn1 := nfConfigApi.PlmnSnssai{
		PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
		SNssaiList: []nfConfigApi.Snssai{
			{Sst: 1, Sd: openapi.PtrString("010203")},
		},
	}
	plmn2 := nfConfigApi.PlmnSnssai{
		PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
		SNssaiList: []nfConfigApi.Snssai{
			{Sst: 2, Sd: openapi.PtrString("112233")},
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
			poller := nfConfigPoller{
				currentPlmnSnssaiConfig: tc.initialPlmnSnssaiConfig,
				currentPlmnConfig:       tc.initialPlmnConfig,
				plmnConfigChan:          ch,
			}

			poller.handlePolledPlmnSnssaiConfig(tc.input)

			select {
			case updated := <-ch:
				t.Errorf("Unexpected channel send: %+v", updated)
			case <-time.After(100 * time.Millisecond):
				// Expected
			}

			if !reflect.DeepEqual(poller.currentPlmnConfig, tc.expectedCurrentPlmnConfig) {
				t.Errorf("Expected current PLMN config: %+v, got: %+v",
					tc.expectedCurrentPlmnConfig, poller.currentPlmnConfig)
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
			{Sst: 1, Sd: openapi.PtrString("010203")},
		},
	}
	plmn2 := nfConfigApi.PlmnSnssai{
		PlmnId: nfConfigApi.PlmnId{Mcc: "002", Mnc: "02"},
		SNssaiList: []nfConfigApi.Snssai{
			{Sst: 2, Sd: openapi.PtrString("112233")},
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

			poller := nfConfigPoller{
				currentPlmnSnssaiConfig: tc.initialPlmnSnssaiConfig,
				currentPlmnConfig:       tc.initialPlmnConfig,
				plmnConfigChan:          ch,
			}

			poller.handlePolledPlmnSnssaiConfig(tc.input)

			select {
			case updated := <-ch:
				if !reflect.DeepEqual(updated, tc.expectedCurrentPlmnConfig) {
					t.Errorf("Wrong config sent on channel.\nExpected: %+v\nGot: %+v", tc.expectedCurrentPlmnConfig, updated)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("Expected update to be sent to channel but none received")
			}

			if !reflect.DeepEqual(poller.currentPlmnConfig, tc.expectedCurrentPlmnConfig) {
				t.Errorf("Expected current PLMN config: %+v, got: %+v",
					tc.expectedCurrentPlmnConfig, poller.currentPlmnConfig)
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
	sdPtr := openapi.PtrString("010203")

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
						{Sst: 3, Sd: openapi.PtrString("112233")},
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

func TestFetchPlmnConfig(t *testing.T) {
	validPlmnList := []nfConfigApi.PlmnSnssai{
		{
			PlmnId:     nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
			SNssaiList: []nfConfigApi.Snssai{{Sst: 1}},
		},
	}
	validJson, err := json.Marshal(validPlmnList)
	if err != nil {
		t.Fail()
	}

	var expectedPlmnSnssai []nfConfigApi.PlmnSnssai
	err = json.Unmarshal(validJson, &expectedPlmnSnssai)
	if err != nil {
		t.Fatalf("failed to unmarshal expectedPlmnSnssai: %v", err)
	}

	tests := []struct {
		name           string
		statusCode     int
		contentType    string
		responseBody   string
		expectedError  string
		expectedResult []nfConfigApi.PlmnSnssai
	}{
		{
			name:           "200 OK with valid JSON",
			statusCode:     http.StatusOK,
			contentType:    "application/json",
			responseBody:   string(validJson),
			expectedError:  "",
			expectedResult: expectedPlmnSnssai,
		},
		{
			name:          "200 OK with invalid Content-Type",
			statusCode:    http.StatusOK,
			contentType:   "text/plain",
			responseBody:  string(validJson),
			expectedError: "unexpected Content-Type: got text/plain, want application/json",
		},
		{
			name:          "400 Bad Request",
			statusCode:    http.StatusBadRequest,
			contentType:   "application/json",
			responseBody:  "",
			expectedError: "server returned 400 error code",
		},
		{
			name:          "500 Internal Server Error",
			statusCode:    http.StatusInternalServerError,
			contentType:   "application/json",
			responseBody:  "",
			expectedError: "server returned 500 error code",
		},
		{
			name:          "Unexpected Status Code 418",
			statusCode:    http.StatusTeapot,
			contentType:   "application/json",
			responseBody:  "",
			expectedError: "unexpected status code: 418",
		},
		{
			name:          "200 OK with invalid JSON",
			statusCode:    http.StatusOK,
			contentType:   "application/json",
			responseBody:  "{invalid-json}",
			expectedError: "failed to parse JSON response:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				accept := r.Header.Get("Accept")
				if accept != "application/json" {
					t.Errorf("Accept header mismatch. got = %q, want = %q", accept, "application/json")
				}
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.statusCode)
				_, err = w.Write([]byte(tc.responseBody))
				if err != nil {
					t.Fail()
				}
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			ch := make(chan []models.PlmnId, 1)
			poller := nfConfigPoller{
				currentPlmnSnssaiConfig: []nfConfigApi.PlmnSnssai{},
				currentPlmnConfig:       []models.PlmnId{{Mcc: "001", Mnc: "01"}},
				plmnConfigChan:          ch,
				client:                  &http.Client{},
			}
			defer server.Close()

			fetchedConfig, err := fetchPlmnConfig(&poller, server.URL)

			if tc.expectedError == "" {
				if err != nil {
					t.Errorf("expected no error, got `%v`", err)
				}
				if !reflect.DeepEqual(tc.expectedResult, fetchedConfig) {
					t.Errorf("error in fetched config: expected `%v`, got `%v`", tc.expectedResult, fetchedConfig)
				}
			} else {
				if err == nil {
					t.Errorf("expected error `%v`, got nil", tc.expectedError)
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("expected error `%v`, got `%v`", tc.expectedError, err)
				}
			}
		})
	}
}
