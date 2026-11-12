// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF NSSAI Availability
 *
 * NSSF NSSAI Availability Service
 */

package producer

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/nssf/util"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/utils"
)

// Get available subscription ID from configuration
// In this implementation, string converted from 32-bit integer is used as subscription ID.
// The smallest unused positive ID is returned; the order of stored subscriptions is irrelevant.
func getUnusedSubscriptionIDLocked() (string, error) {
	used := make(map[uint32]struct{}, len(factory.NssfConfig.Subscriptions))
	for _, subscription := range factory.NssfConfig.Subscriptions {
		tempID, err := strconv.ParseUint(subscription.SubscriptionId, 10, 32)
		if err != nil {
			// Not an ID this allocator could have produced; it cannot collide
			continue
		}
		used[uint32(tempID)] = struct{}{}
	}
	// Wider counter so that math.MaxUint32 itself is scanned without overflow
	for idx := uint64(1); idx <= math.MaxUint32; idx++ {
		if _, taken := used[uint32(idx)]; !taken {
			return strconv.FormatUint(idx, 10), nil
		}
	}
	return "", fmt.Errorf("no available subscription ID")
}

// NSSAIAvailability subscription POST method
func NSSAIAvailabilityPostProcedure(createData models.NssfEventSubscriptionCreateData) (
	*models.NssfEventSubscriptionCreatedData, *models.ProblemDetails,
) {
	response := models.NewNssfEventSubscriptionCreatedDataWithDefaults()

	var subscription factory.Subscription
	factory.ConfigLock.Lock()
	defer factory.ConfigLock.Unlock()

	tempID, err := getUnusedSubscriptionIDLocked()
	if err != nil {
		logger.Nssaiavailability.Warnln(err.Error())
		problemDetails := utils.ProblemDetails(util.UNSUPPORTED_RESOURCE, http.StatusNotFound, err.Error())
		return nil, problemDetails
	}

	subscription.SubscriptionId = tempID
	subscription.SubscriptionData = &createData

	factory.NssfConfig.Subscriptions = append(factory.NssfConfig.Subscriptions, subscription)

	response.SetSubscriptionId(subscription.SubscriptionId)
	timeExpiry := subscription.SubscriptionData.GetExpiry()
	if !timeExpiry.IsZero() {
		response.SetExpiry(timeExpiry)
	}
	response.SetAuthorizedNssaiAvailabilityData(util.AuthorizeOfTaListFromConfig(subscription.SubscriptionData.GetTaiList()))

	return response, nil
}

func NSSAIAvailabilityUnsubscribeProcedure(subscriptionId string) *models.ProblemDetails {
	var problemDetails *models.ProblemDetails

	factory.ConfigLock.Lock()
	defer factory.ConfigLock.Unlock()
	for i, subscription := range factory.NssfConfig.Subscriptions {
		if subscription.SubscriptionId == subscriptionId {
			factory.NssfConfig.Subscriptions = append(factory.NssfConfig.Subscriptions[:i],
				factory.NssfConfig.Subscriptions[i+1:]...)
			return nil
		}
	}

	// No specific subscription ID exists
	problemDetails = utils.ProblemDetails(
		util.UNSUPPORTED_RESOURCE,
		http.StatusNotFound,
		fmt.Sprintf("Subscription ID '%s' is not available", subscriptionId),
	)
	return problemDetails
}
