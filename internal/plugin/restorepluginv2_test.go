package plugin

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestUpdateSvcSelectorPluginV2_AppliesTo(t *testing.T) {
	t.Run("Only applies to Services", func(t *testing.T) {
		plugin := &UpdateSvcSelectorPluginV2{
			log: logrus.New(),
		}

		want := velero.ResourceSelector{
			IncludedResources: []string{"services"},
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

func TestUpdateSvcSelectorPluginV2_Execute(t *testing.T) {

	t.Run("Updates Service Selector", func(t *testing.T) {
		// Create the unstructured object with the annotation as a JSON string
		unstructuredObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]interface{}{
					"name":      "test-service",
					"namespace": "test-namespace",
					"annotations": map[string]interface{}{
						"eth-eks.velero/update-svc-selector": `{"app":"new-app","tier":"frontend"}`,
					},
				},
				"spec": map[string]interface{}{
					"selector": map[string]interface{}{
						"app":  "old-app",
						"tier": "backend",
					},
				},
			},
		}

		input := &velero.RestoreItemActionExecuteInput{
			Item: unstructuredObj,
		}

		plugin := &UpdateSvcSelectorPluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
			return
		}

		if output == nil || output.UpdatedItem == nil {
			t.Error("Expected non-nil output and UpdatedItem")
			return
		}

		var updatedService corev1.Service
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(output.UpdatedItem.UnstructuredContent(), &updatedService); err != nil {
			t.Errorf("Error converting output to Service: %v", err)
		}

		wantSelector := map[string]string{
			"app":  "new-app",
			"tier": "frontend",
		}
		if !reflect.DeepEqual(updatedService.Spec.Selector, wantSelector) {
			t.Errorf("Execute() got selector = %v, want %v", updatedService.Spec.Selector, wantSelector)
		}
	})

	t.Run("No changes when annotation is missing", func(t *testing.T) {
		unstructuredObj := &unstructured.Unstructured{}
		unstructuredObj.Object = map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "test-service",
				"namespace": "test-namespace",
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"app":  "original-app",
					"tier": "original-tier",
				},
			},
		}

		input := &velero.RestoreItemActionExecuteInput{
			Item: unstructuredObj,
		}

		plugin := &UpdateSvcSelectorPluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
		}

		var updatedService corev1.Service
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(output.UpdatedItem.UnstructuredContent(), &updatedService); err != nil {
			t.Errorf("Error converting output to Service: %v", err)
		}

		wantSelector := map[string]string{
			"app":  "original-app",
			"tier": "original-tier",
		}
		if !reflect.DeepEqual(updatedService.Spec.Selector, wantSelector) {
			t.Errorf("Execute() got selector = %v, want %v", updatedService.Spec.Selector, wantSelector)
		}
	})

	t.Run("Returns error when annotation value is invalid", func(t *testing.T) {
		unstructuredObj := &unstructured.Unstructured{}
		unstructuredObj.Object = map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "test-service",
				"namespace": "test-namespace",
				"annotations": map[string]interface{}{
					"eth-eks.velero/update-svc-selector": "invalid-value",
				},
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"app": "original-app",
				},
			},
		}

		input := &velero.RestoreItemActionExecuteInput{
			Item: unstructuredObj,
		}

		plugin := &UpdateSvcSelectorPluginV2{
			log: logrus.New(),
		}

		_, err := plugin.Execute(input)
		if err == nil {
			t.Error("Expected error when parsing invalid selector value, got nil")
		}
	})
}

func TestUpdateSvcSelectorPluginV2_getServiceSelectorFromAnnotation(t *testing.T) {
	plugin := &UpdateSvcSelectorPluginV2{
		log: logrus.New(),
	}

	tests := []struct {
		name    string
		item    *unstructured.Unstructured
		want    map[string]string
		wantErr bool
	}{
		{
			name: "valid annotation",
			item: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							"eth-eks.velero/update-svc-selector": `{"app":"new-app","env":"prod"}`,
						},
					},
				},
			},
			want: map[string]string{
				"app": "new-app",
				"env": "prod",
			},
			wantErr: false,
		},
		{
			name: "no annotations",
			item: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "annotation not present",
			item: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							"other-annotation": "value",
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "invalid selector type",
			item: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							"eth-eks.velero/update-svc-selector": "invalid-value",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid selector value type",
			item: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							"eth-eks.velero/update-svc-selector": `{"app": 123}`, // number instead of string
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plugin.getServiceSelectorFromAnnotation(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("getServiceSelectorFromAnnotation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getServiceSelectorFromAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}
