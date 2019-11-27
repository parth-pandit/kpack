package builder

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	
	"github.com/pivotal/kpack/pkg/client/informers/externalversions/build/v1alpha1"
	v1alpha1Listers "github.com/pivotal/kpack/pkg/client/listers/build/v1alpha1"
	expv1alpha1Listers "github.com/pivotal/kpack/pkg/client/listers/experimental/v1alpha1"
)

type BuilderInformer struct {
	BuilderInformer v1alpha1.BuilderInformer
}

func (bi *BuilderInformer) AddEventHandler(handler cache.ResourceEventHandler)  {
}


func (bi *BuilderInformer) Lister()  {
}


type BuilderLister struct {
	BuilderLister        v1alpha1Listers.BuilderLister
	ClusterBuilderLister v1alpha1Listers.ClusterBuilderLister
	CustomBuilderLister  expv1alpha1Listers.CustomBuilderLister
}

func (bi *BuilderInformer) Get(reference corev1.ObjectReference) {

}