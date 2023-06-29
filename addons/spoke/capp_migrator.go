package addons

import (
	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
)

func syncCappStatus(sourceCappStatus *rcsv1alpha1.CappStatus, destinationCappStatus *rcsv1alpha1.CappStatus) {
	destinationCappStatus.ApplicationLinks = sourceCappStatus.ApplicationLinks
	destinationCappStatus.RevisionInfo = sourceCappStatus.RevisionInfo
	destinationCappStatus.KnativeObjectStatus = sourceCappStatus.KnativeObjectStatus
}
