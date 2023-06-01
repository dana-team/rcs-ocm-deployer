package webhooks

import (
	"testing"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
)

type TestCase struct {
	cappName    string
	scaleMetric string
	mutated     bool
}

func TestDefaultsWebhook(t *testing.T) {
	tests := []TestCase{
		{
			cappName: "i-forgot-metric",
			mutated:  true,
		}, {
			cappName:    "i-put-metric",
			scaleMetric: "cpu",
			mutated:     false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.cappName, func(t *testing.T) {
			capp := rcsv1alpha1.Capp{
				ObjectMeta: metav1.ObjectMeta{
					Name: tc.cappName,
				},
				Spec: rcsv1alpha1.CappSpec{
					ConfigurationSpec: knativev1.ConfigurationSpec{},
					ScaleMetric:       tc.scaleMetric,
					Site:              "local-cluster",
					RouteSpec: rcsv1alpha1.RouteSpec{
						Hostname: "test.com",
					},
				},
			}
			g := NewWithT(t)
			rc := DefaultMutator{}

			oldCapp := capp
			rc.handle(&capp)
			if tc.mutated {
				g.Expect(capp.Spec.ScaleMetric).Should(Equal(DefaultScaleMetric))
			} else {
				g.Expect(capp.Spec.ScaleMetric == oldCapp.Spec.ScaleMetric).Should(BeTrue())
			}
		})
	}
}
