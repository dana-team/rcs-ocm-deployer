package statusspoke

import (
	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
)

func syncCappStatus(sourceCappStatus *cappv1alpha1.CappStatus, destinationCappStatus *cappv1alpha1.CappStatus) {
	if sourceCappStatus.ApplicationLinks.ConsoleLink != "" {
		destinationCappStatus.ApplicationLinks.ConsoleLink = sourceCappStatus.ApplicationLinks.ConsoleLink
	}
	destinationCappStatus.RevisionInfo = sourceCappStatus.RevisionInfo
	destinationCappStatus.KnativeObjectStatus = sourceCappStatus.KnativeObjectStatus
	destinationCappStatus.LoggingStatus = sourceCappStatus.LoggingStatus
	destinationCappStatus.StateStatus = sourceCappStatus.StateStatus
	destinationCappStatus.RouteStatus = sourceCappStatus.RouteStatus
}
