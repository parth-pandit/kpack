package custombuilder_test

import (
	"errors"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/controller"
	rtesting "knative.dev/pkg/reconciler/testing"

	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	expv1alpha1 "github.com/pivotal/kpack/pkg/apis/experimental/v1alpha1"
	"github.com/pivotal/kpack/pkg/client/clientset/versioned/fake"
	"github.com/pivotal/kpack/pkg/reconciler/testhelpers"
	"github.com/pivotal/kpack/pkg/reconciler/v1alpha1/custombuilder"
	"github.com/pivotal/kpack/pkg/registry"
)

func TestCustomBuilderReconciler(t *testing.T) {
	spec.Run(t, "Custom Builder Reconciler", testCustomBuilderReconciler)
}

func testCustomBuilderReconciler(t *testing.T, when spec.G, it spec.S) {
	const (
		testNamespace                 = "some-namespace"
		customBuilderName             = "custom-builder"
		customBuilderKey              = testNamespace + "/" + customBuilderName
		customBuilderTag              = "example.com/custom-builder"
		customBuilderIdentifier       = "example.com/custom-builder@sha256:resolved-builder-digest"
		initialGeneration       int64 = 1
	)

	var (
		builderCreator  = &fakeBuilderCreator{}
		keychainFactory = &fakeKeychainFactory{}
	)

	rt := testhelpers.ReconcilerTester(t,
		func(t *testing.T, row *rtesting.TableRow) (reconciler controller.Reconciler, lists rtesting.ActionRecorderList, list rtesting.EventList, reporter *rtesting.FakeStatsReporter) {
			listers := testhelpers.NewListers(row.Objects)
			fakeClient := fake.NewSimpleClientset(listers.BuildServiceObjects()...)
			r := &custombuilder.Reconciler{
				Client:              fakeClient,
				CustomBuilderLister: listers.GetCustomBuilderLister(),
				BuilderCreator:      builderCreator,
				KeychainFactory:     keychainFactory,
			}
			return r, rtesting.ActionRecorderList{fakeClient}, rtesting.EventList{Recorder: record.NewFakeRecorder(10)}, &rtesting.FakeStatsReporter{}
		})

	customBuilder := &expv1alpha1.CustomBuilder{
		ObjectMeta: v1.ObjectMeta{
			Name:       customBuilderName,
			Generation: initialGeneration,
			Namespace:  testNamespace,
		},
		Spec: expv1alpha1.CustomBuilderSpec{
			Tag: customBuilderTag,
			Stack: expv1alpha1.Stack{
				BaseBuilderImage: "example.com/some-base-image",
			},
			Store: expv1alpha1.Store{
				Image: "example.com/some-store-image",
			},
			Order: []expv1alpha1.Group{
				{
					Group: []expv1alpha1.Buildpack{
						{
							ID:       "buildpack.id.1",
							Version:  "1.0.0",
							Optional: false,
						},
						{
							ID:       "buildpack.id.2",
							Version:  "2.0.0",
							Optional: false,
						},
					},
				},
			},
			ServiceAccount: "some-service-account",
		},
	}

	when("#Reconcile", func() {
		it("saves metadata to the status", func() {
			builderCreator.Record = v1alpha1.BuilderRecord{
				Image: customBuilderIdentifier,
				Stack: v1alpha1.BuildStack{
					RunImage: "example.com/run-image@sha256:123456",
					ID:       "fake.stack.id",
				},
				Buildpacks: v1alpha1.BuildpackMetadataList{
					{
						ID:      "buildpack.id.1",
						Version: "1.0.0",
					},
					{
						ID:      "buildpack.id.2",
						Version: "2.0.0",
					},
				},
			}

			expectedBuilder := &expv1alpha1.CustomBuilder{
				ObjectMeta: customBuilder.ObjectMeta,
				Spec:       customBuilder.Spec,
				Status: v1alpha1.BuilderStatus{
					Status: duckv1alpha1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1alpha1.Conditions{
							{
								Type:   duckv1alpha1.ConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
					BuilderMetadata: []v1alpha1.BuildpackMetadata{
						{
							ID:      "buildpack.id.1",
							Version: "1.0.0",
						},
						{
							ID:      "buildpack.id.2",
							Version: "2.0.0",
						},
					},
					Stack: v1alpha1.BuildStack{
						RunImage: "example.com/run-image@sha256:123456",
						ID:       "fake.stack.id",
					},
					LatestImage: customBuilderIdentifier,
				},
			}

			rt.Test(rtesting.TableRow{
				Key:     customBuilderKey,
				Objects: []runtime.Object{customBuilder},
				WantErr: false,
				WantStatusUpdates: []clientgotesting.UpdateActionImpl{
					{
						Object: expectedBuilder,
					},
				},
			})

			assert.Equal(t, customBuilder.Spec.ServiceAccount, keychainFactory.SecretRef.ServiceAccount)
			assert.Equal(t, customBuilder.Namespace, keychainFactory.SecretRef.Namespace)
			assert.Len(t, keychainFactory.SecretRef.ImagePullSecrets, 0)
		})

		it("does not update the status with no status change", func() {
			builderCreator.Record = v1alpha1.BuilderRecord{
				Image: customBuilderIdentifier,
				Stack: v1alpha1.BuildStack{
					RunImage: "example.com/run-image@sha256:123456",
					ID:       "fake.stack.id",
				},
				Buildpacks: v1alpha1.BuildpackMetadataList{
					{
						ID:      "buildpack.id.1",
						Version: "1.0.0",
					},
				},
			}

			customBuilder.Status = v1alpha1.BuilderStatus{
				Status: duckv1alpha1.Status{
					ObservedGeneration: customBuilder.Generation,
					Conditions: duckv1alpha1.Conditions{
						{
							Type:   duckv1alpha1.ConditionReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
				BuilderMetadata: []v1alpha1.BuildpackMetadata{
					{
						ID:      "buildpack.id.1",
						Version: "1.0.0",
					},
				},
				Stack: v1alpha1.BuildStack{
					RunImage: "example.com/run-image@sha256:123456",
					ID:       "fake.stack.id",
				},
				LatestImage: customBuilderIdentifier,
			}

			rt.Test(rtesting.TableRow{
				Key:     customBuilderKey,
				Objects: []runtime.Object{customBuilder},
				WantErr: false,
			})
		})

		it("updates status on creation error", func() {
			builderCreator.CreateErr = errors.New("create error")

			expectedBuilder := &expv1alpha1.CustomBuilder{
				ObjectMeta: customBuilder.ObjectMeta,
				Spec:       customBuilder.Spec,
				Status: v1alpha1.BuilderStatus{
					Status: duckv1alpha1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1alpha1.Conditions{
							{
								Type:    duckv1alpha1.ConditionReady,
								Status:  corev1.ConditionFalse,
								Message: "create error",
							},
						},
					},
				},
			}

			rt.Test(rtesting.TableRow{
				Key:     customBuilderKey,
				Objects: []runtime.Object{customBuilder},
				WantErr: true,
				WantStatusUpdates: []clientgotesting.UpdateActionImpl{
					{
						Object: expectedBuilder,
					},
				},
			})
		})
	})
}

type fakeBuilderCreator struct {
	Record    v1alpha1.BuilderRecord
	CreateErr error
}

func (f *fakeBuilderCreator) CreateBuilder(authn.Keychain, *expv1alpha1.CustomBuilder) (v1alpha1.BuilderRecord, error) {
	return f.Record, f.CreateErr
}

type fakeKeychainFactory struct {
	SecretRef registry.SecretRef
}

func (f *fakeKeychainFactory) KeychainForSecretRef(secretRef registry.SecretRef) (authn.Keychain, error) {
	f.SecretRef = secretRef
	return &fakeKeychain{}, nil
}

type fakeKeychain struct {
}

func (f *fakeKeychain) Resolve(authn.Resource) (authn.Authenticator, error) {
	return authn.Anonymous, nil
}
