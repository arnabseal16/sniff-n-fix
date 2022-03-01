package controllers

import (
	"errors"
	"fmt"
	"strings"

	snfv1 "ccs.sniff-n-fix.com/snf-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *EventListenerReconciler) newResource(resourceType snfv1.ResourceType, name, namespace string) (Resource, error) {
	gvk, err := getGroupVersionKind(resourceType)
	if err != nil {
		return nil, err
	}
	newObject, err := r.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	newResource := newObject.(Resource)
	newResource.SetName(name)
	newResource.SetNamespace(namespace)
	SetKind(newResource, gvk.Kind)
	return newResource, nil
}

type Resource interface {
	metav1.Object
	runtime.Object
}

func SetKind(resource Resource, kind string) {
	getType(resource).SetKind(kind)
}

func getType(resource Resource) metav1.Type {
	t, _ := meta.TypeAccessor(resource)
	return t
}

func getGroupVersionKind(resourceType snfv1.ResourceType) (schema.GroupVersionKind, error) {
	registeredGvks := map[string]string{
		"Pod":        "v1",
		"Deployment": "v1",
	}

	kind := strings.Title(string(resourceType))
	apiVersion, found := registeredGvks[kind]
	if !found {
		return schema.GroupVersionKind{}, errors.New(fmt.Sprintf("Kind '%s' not supported", resourceType))
	}

	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    kind,
	}
	return gvk, nil
}
