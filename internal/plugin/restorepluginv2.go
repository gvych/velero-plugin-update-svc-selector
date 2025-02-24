package plugin

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// UpdateSvcSelectorPluginV2 is a restore item action plugin for Velero.
type UpdateSvcSelectorPluginV2 struct {
	log logrus.FieldLogger
}

// NewUpdateSvcSelectorPluginV2 instantiates a v2 UpdateSvcSelectorPlugin.
func NewUpdateSvcSelectorPluginV2(log logrus.FieldLogger) *UpdateSvcSelectorPluginV2 {
	return &UpdateSvcSelectorPluginV2{log: log}
}

// Name returns the name of the plugin
func (p *UpdateSvcSelectorPluginV2) Name() string {
	return "eth-eks/update-svc-selector"
}

// AppliesTo returns information about which resources this action should be invoked for.
func (p *UpdateSvcSelectorPluginV2) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"services"},
	}, nil
}

// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored
func (p *UpdateSvcSelectorPluginV2) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.log.Info("Executing UpdateSvcSelectorPluginV2")

	item := input.Item.(*unstructured.Unstructured)
	newSelector, err := p.getServiceSelectorFromAnnotation(item)
	if err != nil {
		return nil, err
	}
	if newSelector == nil {
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	service, err := p.convertToService(item)
	if err != nil {
		return nil, err
	}

	service.Spec.Selector = newSelector
	p.log.Infof("Updating service selector to: %v", newSelector)

	updatedItem, err := p.convertToUnstructured(service)
	if err != nil {
		return nil, err
	}

	return velero.NewRestoreItemActionExecuteOutput(updatedItem), nil
}

func (p *UpdateSvcSelectorPluginV2) getServiceSelectorFromAnnotation(item *unstructured.Unstructured) (map[string]string, error) {
	metadata := item.UnstructuredContent()["metadata"].(map[string]interface{})
	annotations := metadata["annotations"]
	if annotations == nil {
		return nil, nil
	}

	annotationsMap := annotations.(map[string]interface{})
	selectorAnnotation, exists := annotationsMap["eth-eks.velero/update-svc-selector"]
	if !exists {
		return nil, nil
	}

	selectorStr, ok := selectorAnnotation.(string)
	if !ok {
		return nil, errors.New("service-selector annotation must be a string")
	}

	var selectorMap map[string]string
	if err := json.Unmarshal([]byte(selectorStr), &selectorMap); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal selector annotation")
	}

	return selectorMap, nil
}

func (p *UpdateSvcSelectorPluginV2) convertToService(item *unstructured.Unstructured) (*corev1.Service, error) {
	var service corev1.Service
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), &service); err != nil {
		return nil, errors.WithStack(err)
	}
	return &service, nil
}

func (p *UpdateSvcSelectorPluginV2) convertToUnstructured(service *corev1.Service) (*unstructured.Unstructured, error) {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(service)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &unstructured.Unstructured{Object: unstructuredObj}, nil
}

func (p *UpdateSvcSelectorPluginV2) Progress(_ string, _ *v1.Restore) (velero.OperationProgress, error) {
	return velero.OperationProgress{Completed: true}, nil
}

func (p *UpdateSvcSelectorPluginV2) Cancel(operationID string, restore *v1.Restore) error {
	return nil
}

func (p *UpdateSvcSelectorPluginV2) AreAdditionalItemsReady(additionalItems []velero.ResourceIdentifier, restore *v1.Restore) (bool, error) {
	return true, nil
}
