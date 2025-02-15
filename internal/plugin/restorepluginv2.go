package plugin

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RestorePluginV2 is a restore item action plugin for Velero.
type RestorePluginV2 struct {
	log logrus.FieldLogger
}

// NewRestorePluginV2 instantiates a v2 RestorePlugin.
func NewRestorePluginV2(log logrus.FieldLogger) *RestorePluginV2 {
	return &RestorePluginV2{log: log}
}

// Name is required to implement the interface, but the Velero pod does not delegate this
// method -- it's used to tell velero what name it was registered under. The plugin implementation
// must define it, but it will never actually be called.
func (p *RestorePluginV2) Name() string {
	return "eth-eks/update-replicas"
}

// AppliesTo returns information about which resources this action should be invoked for.
// The IncludedResources and ExcludedResources slices can include both resources
// and resources with group names. These work: "ingresses", "ingresses.extensions".
// A RestoreItemAction's Execute function will only be invoked on items that match the returned
// selector. A zero-valued ResourceSelector matches all resources.
func (p *RestorePluginV2) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"statefulsets", "deployments"},
	}, nil
}

// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored
func (p *RestorePluginV2) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	item := input.Item.(*unstructured.Unstructured)
	replicasValue, exists := p.getReplicasAnnotation(item)
	if !exists {
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	replicas, err := p.parseReplicasValue(replicasValue)
	if err != nil {
		return nil, err
	}

	kind := item.GetObjectKind().GroupVersionKind().Kind
	resource, err := p.createResource(kind)
	if err != nil {
		return nil, err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), resource); err != nil {
		return nil, errors.WithStack(err)
	}

	replicasInt32, err := p.convertToInt32(replicas)
	if err != nil {
		return nil, err
	}

	p.log.Infof("Setting replicas to %d", replicasInt32)

	if err := p.setReplicas(resource, replicasInt32, kind); err != nil {
		return nil, err
	}

	return p.createOutput(resource)
}

func (p *RestorePluginV2) getReplicasAnnotation(item *unstructured.Unstructured) (interface{}, bool) {
	metadata := item.UnstructuredContent()["metadata"].(map[string]interface{})
	annotations, _ := metadata["annotations"].(map[string]interface{})
	value, exists := annotations["eth-eks.velero/replicas-value-after-recovery"]
	return value, exists
}

func (p *RestorePluginV2) parseReplicasValue(value interface{}) (string, error) {
	replicas, ok := value.(string)
	if !ok {
		return "", errors.New("replicas annotation value must be a string")
	}
	p.log.Infof("Replicas value: %s", replicas)
	return replicas, nil
}

func (p *RestorePluginV2) createResource(kind string) (interface{}, error) {
	switch kind {
	case "StatefulSet":
		p.log.Infof("Creating StatefulSet resource")
		return &apps.StatefulSet{}, nil
	case "Deployment":
		p.log.Infof("Creating Deployment resource")
		return &apps.Deployment{}, nil
	default:
		p.log.Infof("Unsupported kind: %s", kind)
		return nil, errors.Errorf("unsupported kind %s", kind)
	}
}

func (p *RestorePluginV2) convertToInt32(replicas string) (int32, error) {
	n, err := strconv.ParseInt(replicas, 10, 32)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse replicas value")
	}
	p.log.Infof("Converted replicas value to int32: %d", n)
	return int32(n), nil
}

func (p *RestorePluginV2) setReplicas(resource interface{}, replicas int32, kind string) error {
	switch kind {
	case "StatefulSet":
		sts := resource.(*apps.StatefulSet)
		sts.Spec.Replicas = &replicas
		p.log.Infof("Set replicas for StatefulSet: %d", replicas)
	case "Deployment":
		deploy := resource.(*apps.Deployment)
		deploy.Spec.Replicas = &replicas
		p.log.Infof("Set replicas for Deployment: %d", replicas)
	}
	return nil
}

func (p *RestorePluginV2) createOutput(resource interface{}) (*velero.RestoreItemActionExecuteOutput, error) {
	inputMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	p.log.Infof("Created output with resource: %v", inputMap)
	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: inputMap}), nil
}

func (p *RestorePluginV2) Cancel(operationID string, restore *v1.Restore) error {
	return nil
}

func (p *RestorePluginV2) AreAdditionalItemsReady(additionalItems []velero.ResourceIdentifier, restore *v1.Restore) (bool, error) {
	return true, nil
}
