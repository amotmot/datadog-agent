// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver
// +build kubeapiserver

package webhook

import (
	"github.com/DataDog/datadog-agent/pkg/clusteragent/admission/common"
	"github.com/DataDog/datadog-agent/pkg/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildLabelSelector returns the mutating webhooks object selector based on the configuration
func buildLabelSelector() *metav1.LabelSelector {
	var labelSelector *metav1.LabelSelector

	if config.Datadog.GetBool("admission_controller.mutate_unlabelled") {
		// Accept all, ignore pods if they're explicitly filtered-out
		labelSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      common.EnabledLabelKey,
					Operator: metav1.LabelSelectorOpNotIn,
					Values:   []string{"false"},
				},
			},
		}
	} else {
		// Ignore all, accept pods if they're explicitly allowed
		labelSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				common.EnabledLabelKey: "true",
			},
		}
	}

	labelSelector.MatchExpressions = append(
		labelSelector.MatchExpressions,
		azureAKSLabelSelectorRequirement(),
	)

	return labelSelector
}

// Returns the label selector needed to make webhooks work on Azure AKS.
// AKS adds this requirement automatically if we don't, so we need to add it to
// avoid conflicts when updating the webhook.
//
// Ref: https://docs.microsoft.com/en-us/azure/aks/faq#can-i-use-admission-controller-webhooks-on-aks
// Ref: https://github.com/Azure/AKS/issues/1771
func azureAKSLabelSelectorRequirement() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      "control-plane",
		Operator: metav1.LabelSelectorOpDoesNotExist,
	}
}
