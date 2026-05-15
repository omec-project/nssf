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
	"time"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/nssf/util"
	"github.com/omec-project/openapi/v2/models"
)

// Get available subscription ID from configuration
// In this implementation, string converted from 32-bit integer is used as subscription ID
func getUnusedSubscriptionIDLocked() (string, error) {
	var idx uint32 = 1
	for _, subscription := range factory.NssfConfig.Subscriptions {
		tempID, err := strconv.Atoi(subscription.SubscriptionId)
		if err != nil {
			return "", err
		}
		if uint32(tempID) == idx {
			if idx == math.MaxUint32 {
				return "", fmt.Errorf("no available subscription ID")
			}
			idx = idx + 1
		} else {
			break
		}
	}
	return strconv.Itoa(int(idx)), nil
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
		problemDetails := models.NewProblemDetails()
		problemDetails.SetTitle(util.UNSUPPORTED_RESOURCE)
		problemDetails.SetStatus(http.StatusNotFound)
		problemDetails.SetDetail(err.Error())
		return nil, problemDetails
	}

	subscription.SubscriptionId = tempID
	subscription.SubscriptionData = &createData

	factory.NssfConfig.Subscriptions = append(factory.NssfConfig.Subscriptions, subscription)

	response.SubscriptionId = subscription.SubscriptionId
	if !subscription.SubscriptionData.Expiry.IsZero() {
		response.Expiry = new(time.Time)
		*response.Expiry = *subscription.SubscriptionData.Expiry
	}
	response.AuthorizedNssaiAvailabilityData = util.AuthorizeOfTaListFromConfig(subscription.SubscriptionData.TaiList)

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
	problemDetails = models.NewProblemDetails()
	problemDetails.SetTitle(util.UNSUPPORTED_RESOURCE)
	problemDetails.SetStatus(http.StatusNotFound)
	problemDetails.SetDetail(fmt.Sprintf("Subscription ID '%s' is not available", subscriptionId))
	return problemDetails
}
