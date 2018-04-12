package main

import (
	"time"

	"k8s.io/apimachinery/pkg/types"
)

type ScaleOperation struct {
	DeploymentID   types.UID
	ScaleDirection string
	CreatedAt      time.Time
}

func NewScaleOperation(deploymentID types.UID, scaleDirection string) ScaleOperation {
	return ScaleOperation{DeploymentID: deploymentID, ScaleDirection: scaleDirection, CreatedAt: time.Now()}
}
