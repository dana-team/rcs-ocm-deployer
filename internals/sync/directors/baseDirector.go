package directors

import (
	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
)

type Director interface {
	AssembleManifests(capp rcsv1alpha1.Capp) ([]workv1.Manifest, error)
}
