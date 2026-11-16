// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2025 Canonical Ltd.
// SPDX-License-Identifier: Apache-2.0

package nfregistration

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omec-project/nssf/consumer"
	"github.com/omec-project/openapi/v2/models"
)

// The registration logic is tested in openapi/nrfregistration. These tests check that the NSSF wiring
// reaches the right consumer functions.

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestStartNfRegistrationService_NonEmptyConfigRegistersThenEmptyConfigDeregisters(t *testing.T) {
	origRegister, origDeregister := consumer.SendRegisterNFInstance, consumer.SendDeregisterNFInstance
	t.Cleanup(func() {
		consumer.SendRegisterNFInstance, consumer.SendDeregisterNFInstance = origRegister, origDeregister
	})
	var registeredWith atomic.Value
	var deregistered atomic.Int32
	consumer.SendRegisterNFInstance = func(_ context.Context, plmnConfig []models.PlmnId) (*models.NFProfile, string, error) {
		registeredWith.Store(plmnConfig)
		return models.NewNFProfileWithDefaults(), "", nil
	}
	consumer.SendDeregisterNFInstance = func(context.Context) error {
		deregistered.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	ch := make(chan []models.PlmnId)
	go func() { defer close(done); StartNfRegistrationService(ctx, ch) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	plmns := []models.PlmnId{*models.NewPlmnId("123", "45")}
	ch <- plmns
	waitFor(t, func() bool { return registeredWith.Load() != nil }, "SendRegisterNFInstance was not called")
	if got := registeredWith.Load().([]models.PlmnId); len(got) != 1 || got[0].Mcc != "123" {
		t.Errorf("registered with %+v", got)
	}

	ch <- nil
	waitFor(t, func() bool { return deregistered.Load() == 1 }, "SendDeregisterNFInstance was not called for an empty config")
}

func TestDeregisterNF_CallsConsumer(t *testing.T) {
	orig := consumer.SendDeregisterNFInstance
	t.Cleanup(func() { consumer.SendDeregisterNFInstance = orig })
	var called atomic.Int32
	consumer.SendDeregisterNFInstance = func(context.Context) error { called.Add(1); return nil }

	DeregisterNF()
	if called.Load() != 1 {
		t.Errorf("SendDeregisterNFInstance calls = %d, want 1", called.Load())
	}
}

func TestHeartbeatUsesSendUpdateNFInstance(t *testing.T) {
	origRegister, origUpdate := consumer.SendRegisterNFInstance, consumer.SendUpdateNFInstance
	t.Cleanup(func() { consumer.SendRegisterNFInstance, consumer.SendUpdateNFInstance = origRegister, origUpdate })
	var updated atomic.Int32
	profile := models.NewNFProfileWithDefaults()
	profile.SetHeartBeatTimer(1)
	consumer.SendRegisterNFInstance = func(context.Context, []models.PlmnId) (*models.NFProfile, string, error) { return profile, "", nil }
	consumer.SendUpdateNFInstance = func(_ context.Context, patch []models.PatchItem) (*models.NFProfile, *models.ProblemDetails, error) {
		updated.Add(1)
		return profile, nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	ch := make(chan []models.PlmnId)
	go func() { defer close(done); StartNfRegistrationService(ctx, ch) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	ch <- []models.PlmnId{*models.NewPlmnId("123", "45")}
	waitFor(t, func() bool { return updated.Load() >= 1 }, "heartbeat did not call SendUpdateNFInstance")
}
