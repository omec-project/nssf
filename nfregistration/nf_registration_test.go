// SPDX-FileCopyrightText: 2025 Canonical Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
/*
 * NRF Registration Unit Testcases
 *
 */
package nfregistration

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omec-project/nssf/consumer"
	"github.com/omec-project/openapi/v2/models"
)

func startRegistrationServiceForTest(t *testing.T, ch <-chan []models.PlmnId) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		StartNfRegistrationService(ctx, ch)
	}()
	return cancel, done
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, errMessage string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(errMessage)
}

func withKeepAliveTimerLock(f func()) {
	keepAliveTimerMutex.Lock()
	defer keepAliveTimerMutex.Unlock()
	f()
}

func TestNfRegistrationService_WhenEmptyConfig_ThenDeregisterNFAndStopTimer(t *testing.T) {
	testCases := []struct {
		name                         string
		sendDeregisterNFInstanceMock func(called chan<- struct{}) func() error
	}{
		{
			name: "Success",
			sendDeregisterNFInstanceMock: func(called chan<- struct{}) func() error {
				return func() error {
					select {
					case called <- struct{}{}:
					default:
					}
					return nil
				}
			},
		},
		{
			name: "ErrorInDeregisterNFInstance",
			sendDeregisterNFInstanceMock: func(called chan<- struct{}) func() error {
				return func() error {
					select {
					case called <- struct{}{}:
					default:
					}
					return errors.New("mock error")
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withKeepAliveTimerLock(func() {
				stopKeepAliveTimer()
				keepAliveTimer = time.NewTimer(60 * time.Second)
			})

			registerCalled := make(chan struct{}, 1)
			deregisterCalled := make(chan struct{}, 1)
			originalDeregisterNF := consumer.SendDeregisterNFInstance
			originalRegisterNF := registerNF

			ch := make(chan []models.PlmnId, 1)
			cancel, done := startRegistrationServiceForTest(t, ch)
			defer func() {
				cancel()
				<-done
				consumer.SendDeregisterNFInstance = originalDeregisterNF
				registerNF = originalRegisterNF
				withKeepAliveTimerLock(func() {
					stopKeepAliveTimer()
				})
			}()

			consumer.SendDeregisterNFInstance = tc.sendDeregisterNFInstanceMock(deregisterCalled)
			registerNF = func(ctx context.Context, newPlmnConfig []models.PlmnId) {
				select {
				case registerCalled <- struct{}{}:
				default:
				}
			}

			ch <- []models.PlmnId{}

			select {
			case <-deregisterCalled:
			case <-time.After(500 * time.Millisecond):
				t.Fatal("expected SendDeregisterNFInstance to be called")
			}

			waitForCondition(t, 500*time.Millisecond, func() bool {
				isNil := false
				withKeepAliveTimerLock(func() {
					isNil = keepAliveTimer == nil
				})
				return isNil
			}, "expected keepAliveTimer to be nil after stopKeepAliveTimer")

			select {
			case <-registerCalled:
				t.Errorf("expected registerNF not to be called")
			default:
			}
		})
	}
}

func TestNfRegistrationService_WhenConfigChanged_ThenRegisterNFSuccessAndStartTimer(t *testing.T) {
	withKeepAliveTimerLock(func() {
		stopKeepAliveTimer()
	})
	originalSendRegisterNFInstance := consumer.SendRegisterNFInstance
	ch := make(chan []models.PlmnId, 1)
	cancel, done := startRegistrationServiceForTest(t, ch)
	defer func() {
		cancel()
		<-done
		consumer.SendRegisterNFInstance = originalSendRegisterNFInstance
		withKeepAliveTimerLock(func() {
			stopKeepAliveTimer()
		})
	}()

	registrationMu := sync.Mutex{}
	registrations := []models.PlmnId{}
	registerCalled := make(chan struct{}, 1)
	consumer.SendRegisterNFInstance = func(plmnConfig []models.PlmnId) (*models.NFProfile, string, error) {
		profile := models.NewNFProfileWithDefaults()
		profile.SetHeartBeatTimer(60)
		registrationMu.Lock()
		registrations = append(registrations, plmnConfig...)
		registrationMu.Unlock()
		select {
		case registerCalled <- struct{}{}:
		default:
		}
		return profile, "", nil
	}

	newConfig := []models.PlmnId{{Mcc: "001", Mnc: "01"}}
	ch <- newConfig

	select {
	case <-registerCalled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected SendRegisterNFInstance to be called")
	}

	waitForCondition(t, 500*time.Millisecond, func() bool {
		isSet := false
		withKeepAliveTimerLock(func() {
			isSet = keepAliveTimer != nil
		})
		return isSet
	}, "expected keepAliveTimer to be initialized by startKeepAliveTimer")

	registrationMu.Lock()
	registered := append([]models.PlmnId(nil), registrations...)
	registrationMu.Unlock()
	if !reflect.DeepEqual(registered, newConfig) {
		t.Errorf("expected %+v config, received %+v", newConfig, registered)
	}
}

func TestNfRegistrationService_WhenEmptyConfig_ThenContinuesListeningForUpdates(t *testing.T) {
	originalDeregisterNF := consumer.SendDeregisterNFInstance
	originalRegisterNF := registerNF
	ch := make(chan []models.PlmnId, 2)
	cancel, done := startRegistrationServiceForTest(t, ch)
	defer func() {
		cancel()
		<-done
		consumer.SendDeregisterNFInstance = originalDeregisterNF
		registerNF = originalRegisterNF
		withKeepAliveTimerLock(func() {
			stopKeepAliveTimer()
		})
	}()

	withKeepAliveTimerLock(func() {
		stopKeepAliveTimer()
	})
	var deregisterCalls atomic.Int32
	consumer.SendDeregisterNFInstance = func() error {
		deregisterCalls.Add(1)
		return nil
	}

	registered := make(chan []models.PlmnId, 1)
	registerNF = func(ctx context.Context, newPlmnConfig []models.PlmnId) {
		registered <- newPlmnConfig
	}

	ch <- []models.PlmnId{}
	ch <- []models.PlmnId{{Mcc: "001", Mnc: "01"}}

	select {
	case config := <-registered:
		if !reflect.DeepEqual(config, []models.PlmnId{{Mcc: "001", Mnc: "01"}}) {
			t.Fatalf("unexpected registration config: %+v", config)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected registration service to continue after empty config")
	}

	if deregisterCalls.Load() != 1 {
		t.Fatalf("expected one deregistration call, got %d", deregisterCalls.Load())
	}

	withKeepAliveTimerLock(func() {
		stopKeepAliveTimer()
	})
}

func TestNfRegistrationService_ConfigChanged_RetryIfRegisterNFFails(t *testing.T) {
	originalSendRegisterNFInstance := consumer.SendRegisterNFInstance
	ch := make(chan []models.PlmnId, 1)
	cancel, done := startRegistrationServiceForTest(t, ch)
	defer func() {
		cancel()
		<-done
		consumer.SendRegisterNFInstance = originalSendRegisterNFInstance
		withKeepAliveTimerLock(func() {
			stopKeepAliveTimer()
		})
	}()

	var called atomic.Int32
	consumer.SendRegisterNFInstance = func(plmnConfig []models.PlmnId) (*models.NFProfile, string, error) {
		profile := models.NewNFProfileWithDefaults()
		profile.SetHeartBeatTimer(60)
		called.Add(1)
		return profile, "", errors.New("mock error")
	}

	ch <- []models.PlmnId{{Mcc: "001", Mnc: "01"}}

	waitForCondition(t, retryTime+3*time.Second, func() bool {
		return called.Load() >= 2
	}, "expected to retry register to NRF")

	if called.Load() < 2 {
		t.Error("expected to retry register to NRF")
	}
	t.Logf("Tried %v times", called.Load())
}

func TestNfRegistrationService_WhenConfigChanged_ThenPreviousRegistrationIsCancelled(t *testing.T) {
	originalRegisterNf := registerNF
	ch := make(chan []models.PlmnId, 1)
	cancel, done := startRegistrationServiceForTest(t, ch)
	defer func() {
		cancel()
		<-done
		registerNF = originalRegisterNf
		withKeepAliveTimerLock(func() {
			stopKeepAliveTimer()
		})
	}()

	type registrationCall struct {
		ctx    context.Context
		config []models.PlmnId
	}
	registrations := make(chan registrationCall, 2)
	registerNF = func(registerCtx context.Context, newPlmnConfig []models.PlmnId) {
		registrations <- registrationCall{ctx: registerCtx, config: newPlmnConfig}
		<-registerCtx.Done() // Wait until registration is cancelled
	}

	firstConfig := []models.PlmnId{{Mcc: "001", Mnc: "01"}}
	ch <- firstConfig

	var firstRegistration registrationCall
	select {
	case firstRegistration = <-registrations:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected one registration to the NRF")
	}

	secondConfig := []models.PlmnId{{Mcc: "002", Mnc: "02"}}
	ch <- secondConfig
	var secondRegistration registrationCall
	select {
	case secondRegistration = <-registrations:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected 2 registrations to the NRF")
	}

	select {
	case <-firstRegistration.ctx.Done():
		// expected
	case <-time.After(500 * time.Millisecond):
		t.Error("expected first registration context to be cancelled")
	}

	select {
	case <-secondRegistration.ctx.Done():
		t.Error("second registration context should not be cancelled")
	default:
		// expected
	}

	if !reflect.DeepEqual(firstRegistration.config, firstConfig) {
		t.Errorf("expected %+v config, received %+v", firstConfig, firstRegistration.config)
	}
	if !reflect.DeepEqual(secondRegistration.config, secondConfig) {
		t.Errorf("expected %+v config, received %+v", secondConfig, secondRegistration.config)
	}
}

func TestHeartbeatNF_Success(t *testing.T) {
	keepAliveTimer = time.NewTimer(60 * time.Second)
	calledRegister := false
	originalSendRegisterNFInstance := consumer.SendRegisterNFInstance
	originalSendUpdateNFInstance := consumer.SendUpdateNFInstance
	defer func() {
		consumer.SendRegisterNFInstance = originalSendRegisterNFInstance
		consumer.SendUpdateNFInstance = originalSendUpdateNFInstance
		if keepAliveTimer != nil {
			keepAliveTimer.Stop()
		}
	}()

	consumer.SendUpdateNFInstance = func(patchItem []models.PatchItem) (*models.NFProfile, *models.ProblemDetails, error) {
		return &models.NFProfile{}, nil, nil
	}
	consumer.SendRegisterNFInstance = func(plmnConfig []models.PlmnId) (*models.NFProfile, string, error) {
		calledRegister = true
		profile := models.NewNFProfileWithDefaults()
		profile.SetHeartBeatTimer(60)
		return profile, "", nil
	}
	plmnConfig := []models.PlmnId{}
	heartbeatNF(plmnConfig)

	if calledRegister {
		t.Errorf("expected registerNF to be called on error")
	}
	if keepAliveTimer == nil {
		t.Error("expected keepAliveTimer to be initialized by startKeepAliveTimer")
	}
}

func TestHeartbeatNF_WhenNfUpdateFails_ThenNfRegistersIsCalled(t *testing.T) {
	keepAliveTimer = time.NewTimer(60 * time.Second)
	calledRegister := false
	originalSendRegisterNFInstance := consumer.SendRegisterNFInstance
	originalSendUpdateNFInstance := consumer.SendUpdateNFInstance
	defer func() {
		consumer.SendRegisterNFInstance = originalSendRegisterNFInstance
		consumer.SendUpdateNFInstance = originalSendUpdateNFInstance
		if keepAliveTimer != nil {
			keepAliveTimer.Stop()
		}
	}()

	consumer.SendUpdateNFInstance = func(patchItem []models.PatchItem) (*models.NFProfile, *models.ProblemDetails, error) {
		return &models.NFProfile{}, nil, errors.New("mock error")
	}

	consumer.SendRegisterNFInstance = func(plmnConfig []models.PlmnId) (*models.NFProfile, string, error) {
		profile := models.NewNFProfileWithDefaults()
		profile.SetHeartBeatTimer(60)
		calledRegister = true
		return profile, "", nil
	}

	plmnConfig := []models.PlmnId{}
	heartbeatNF(plmnConfig)

	if !calledRegister {
		t.Errorf("expected registerNF to be called on error")
	}
	if keepAliveTimer == nil {
		t.Error("expected keepAliveTimer to be initialized by startKeepAliveTimer")
	}
}

func TestHeartbeatNF_UsesDefaultTimerWhenUpdateReturnsNilProfile(t *testing.T) {
	keepAliveTimer = time.NewTimer(60 * time.Second)
	originalSendUpdateNFInstance := consumer.SendUpdateNFInstance
	defer func() {
		consumer.SendUpdateNFInstance = originalSendUpdateNFInstance
		if keepAliveTimer != nil {
			keepAliveTimer.Stop()
		}
	}()

	consumer.SendUpdateNFInstance = func(patchItem []models.PatchItem) (*models.NFProfile, *models.ProblemDetails, error) {
		return nil, nil, nil
	}

	heartbeatNF(nil)

	if keepAliveTimer == nil {
		t.Error("expected keepAliveTimer to be initialized when update returns nil profile")
	}
}

func TestStartKeepAliveTimer_UsesProfileTimerOnlyWhenGreaterThanZero(t *testing.T) {
	testCases := []struct {
		name             string
		profileTime      int32
		expectedDuration time.Duration
	}{
		{
			name:             "Profile heartbeat time is zero, use default time",
			profileTime:      0,
			expectedDuration: 60 * time.Second,
		},
		{
			name:             "Profile heartbeat time is smaller than zero, use default time",
			profileTime:      -5,
			expectedDuration: 60 * time.Second,
		},
		{
			name:             "Profile heartbeat time is greater than zero, use profile time",
			profileTime:      15,
			expectedDuration: 15 * time.Second,
		},
		{
			name:             "Profile heartbeat time is greater than default time, use profile time",
			profileTime:      90,
			expectedDuration: 90 * time.Second,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keepAliveTimer = time.NewTimer(25 * time.Second)
			defer func() {
				if keepAliveTimer != nil {
					keepAliveTimer.Stop()
				}
			}()
			var capturedDuration time.Duration

			afterFunc = func(d time.Duration, _ func()) *time.Timer {
				capturedDuration = d
				return time.NewTimer(25 * time.Second)
			}
			defer func() { afterFunc = time.AfterFunc }()

			startKeepAliveTimer(tc.profileTime, nil)
			if tc.expectedDuration != capturedDuration {
				t.Errorf("expected %v duration, got %v", tc.expectedDuration, capturedDuration)
			}
		})
	}
}
