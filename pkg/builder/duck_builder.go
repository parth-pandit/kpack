package builder

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
)

type DuckBuilder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status v1alpha1.BuilderStatus `json:"status"`
}

func (b *DuckBuilder) Ready() bool {
	return b.Status.GetCondition(duckv1alpha1.ConditionReady).IsTrue() &&
		(b.Generation == b.Status.ObservedGeneration)
}

func (b *DuckBuilder) BuildBuilderSpec() BuildBuilderSpec {
	return v1 BuildBuilderSpec{
		Image:            b.Status.LatestImage,
		ImagePullSecrets: b.Spec.ImagePullSecrets,
	}
}

func (b *Builder) ImagePullSecrets() []v1.LocalObjectReference {
	return b.Spec.ImagePullSecrets
}

func (b *Builder) Image() string {
	return b.Spec.Image
}

func (b *Builder) BuildpackMetadata() BuildpackMetadataList {
	return b.Status.BuilderMetadata
}

func (b *Builder) RunImage() string {
	return b.Status.Stack.RunImage
}
