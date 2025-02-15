package plugin

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRestorePluginV2_AppliesTo(t *testing.T) {
	t.Run("Only applies to Deployments and StatefulSets", func(t *testing.T) {
		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		want := velero.ResourceSelector{
			IncludedResources: []string{"statefulsets", "deployments"},
		}
		got, err := plugin.AppliesTo()
		if err != nil {
			t.Errorf("AppliesTo() error = %v", err)
			return
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("AppliesTo() got = %v, want %v", got, want)
		}
	})
}

func TestRestorePluginV2_Execute(t *testing.T) {
	t.Run("Updates Deployment Replicas", func(t *testing.T) {
		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"eth-eks.velero/replicas-value-after-recovery": "3",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
			},
		}

		deploymentUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
		if err != nil {
			t.Errorf("Error converting Deployment to unstructured: %v", err)
		}

		// Add kind information
		deploymentUnstructured["kind"] = "Deployment"

		input := &velero.RestoreItemActionExecuteInput{
			Item: &unstructured.Unstructured{
				Object: deploymentUnstructured,
			},
		}

		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
		}

		got := output.UpdatedItem.UnstructuredContent()["spec"].(map[string]interface{})["replicas"]
		want := int64(3)
		if got != want {
			t.Errorf("Execute() got = %v, want %v", got, want)
		}
	})

	t.Run("Updates StatefulSet Replicas", func(t *testing.T) {
		statefulSet := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"eth-eks.velero/replicas-value-after-recovery": "5",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: int32Ptr(2),
			},
		}

		statefulSetUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&statefulSet)
		if err != nil {
			t.Errorf("Error converting StatefulSet to unstructured: %v", err)
		}

		statefulSetUnstructured["kind"] = "StatefulSet"

		input := &velero.RestoreItemActionExecuteInput{
			Item: &unstructured.Unstructured{
				Object: statefulSetUnstructured,
			},
		}

		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
		}

		got := output.UpdatedItem.UnstructuredContent()["spec"].(map[string]interface{})["replicas"]
		want := int64(5)
		if got != want {
			t.Errorf("Execute() got = %v, want %v", got, want)
		}
	})

	t.Run("No changes when annotation is missing", func(t *testing.T) {
		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(2),
			},
		}

		deploymentUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
		if err != nil {
			t.Errorf("Error converting Deployment to unstructured: %v", err)
		}

		deploymentUnstructured["kind"] = "Deployment"

		input := &velero.RestoreItemActionExecuteInput{
			Item: &unstructured.Unstructured{
				Object: deploymentUnstructured,
			},
		}

		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
		}

		got := output.UpdatedItem.UnstructuredContent()["spec"].(map[string]interface{})["replicas"]
		want := int64(2)
		if got != want {
			t.Errorf("Execute() got = %v, want %v", got, want)
		}
	})

	t.Run("Returns error when annotation value is invalid", func(t *testing.T) {
		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"eth-eks.velero/replicas-value-after-recovery": "invalid",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(2),
			},
		}

		deploymentUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
		if err != nil {
			t.Errorf("Error converting Deployment to unstructured: %v", err)
		}

		deploymentUnstructured["kind"] = "Deployment"

		input := &velero.RestoreItemActionExecuteInput{
			Item: &unstructured.Unstructured{
				Object: deploymentUnstructured,
			},
		}

		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		_, err = plugin.Execute(input)
		if err == nil {
			t.Error("Expected error when parsing invalid replicas value, got nil")
		}
		expectedError := "failed to parse replicas value: strconv.ParseInt: parsing \"invalid\": invalid syntax"
		if err.Error() != expectedError {
			t.Errorf("Expected error message %q, got %q", expectedError, err.Error())
		}
	})
}

func int32Ptr(i int32) *int32 {
	return &i
}
