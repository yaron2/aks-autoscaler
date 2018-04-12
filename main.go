package main

import (
	"os"
	"strconv"
)

type ScaleCommandExecutor interface {
	Scale(agents int32)
	Login()
}

func main() {
	maxNodes, _ := strconv.Atoi(os.Getenv("MAX_NODES"))
	minNodes, _ := strconv.Atoi(os.Getenv("MIN_NODES"))

	cliExecutor := NewCliExecutor(os.Getenv("APP_ID"), os.Getenv("PASSWORD"), os.Getenv("TENANT_ID"), os.Getenv("AKS_NAME"), os.Getenv("RESOURCE_GROUP"))
	autoscaler := NewAzureAutoScaler(cliExecutor, maxNodes, minNodes)

	autoscaler.Start()
}
