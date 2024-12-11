package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type instance interface {
	client.Object

	SetStatusDelta(string)
	GetStatusDelta() string
	GetInsync() bool
}

func GetDeltaString(spec interface{}, current interface{}, parameters map[string]interface{}) (string, error) {

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	currentBytes, err := json.Marshal(current)
	if err != nil {
		return "", err
	}

	var specData map[string]interface{}
	var currentData map[string]interface{}

	err = json.Unmarshal([]byte(specBytes), &specData)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(currentBytes), &currentData)
	if err != nil {
		return "", err
	}

	diff := cmp.Diff(currentData, specData)
	deltaString := collectDiffValues(diff, parameters)
	deltaString = strings.TrimSuffix(deltaString, "\n")
	return deltaString, nil
}

func SetInstanceDelta(
	inst instance, spec, current interface{},
	parameters map[string]interface{},
	status client.StatusWriter, log logr.Logger,
) {
	kind := inst.GetObjectKind().GroupVersionKind().Kind
	log.Info(fmt.Sprintf("Updating delta for kind %s", kind))

	oldDelta := inst.GetStatusDelta()
	if inst.GetInsync() {
		inst.SetStatusDelta("")
	} else if delta, err := GetDeltaString(spec, current, parameters); err == nil {
		inst.SetStatusDelta(delta)
	} else {
		log.Info(fmt.Sprintf("Failed to get Delta string for kind %s: %s\n", kind, err))
	}

	if oldDelta != inst.GetStatusDelta() {
		err := status.Update(context.TODO(), inst)
		if err != nil {
			log.Info(fmt.Sprintf("Failed to update the status for kind %s: %s", kind, err))
		}
	}
}
