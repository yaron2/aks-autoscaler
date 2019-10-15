# AKS Autoscaler

[![GoDoc](https://godoc.org/github.com/yaron2/aks-autoscaler?status.svg)](https://godoc.org/github.com/yaron2/aks-autoscaler)
[![Go Report Card](https://goreportcard.com/badge/github.com/yaron2/aks-autoscaler)](https://goreportcard.com/report/github.com/yaron2/aks-autoscaler)

A node autoscaler for AKS (Azure Container Service)

## About

Use this auto scaler implementation as a Pod inside your k8s cluster to scale your cluster in and out.
The auto scaler uses the native aks scale command that **should** (nothing is certain in life and code) drain your nodes before scaling.

Min / Max nodes and namespace exclusion functionalities are supported.

## Usage
Either create your own image using the Dockerfile or use the ready Docker image in /kubernetes/autoscaler.yaml.

Fill the following environment variables inside the YAML:

1) APP_ID - A Service Principal ID of an AAD app that has contributor permissions for the AKS cluster
2) PASSWORD - The password of the AD app
3) TENANT_ID - Your Azure subscription's TenantID
4) AKS_NAME - The name of the Container Service
5) RESOURCE_GROUP - The name of the Resource Group of the AKS cluster (NOT the MC_ Resource Group)
6) MAX_NODES **(Optional)** - The maximum number of nodes to scale up to
7) MIN_NODES **(Optional)** - The minimum number of nodes to scale down to
8) EXCLUDED_NAMESPACES **(Optional)** - A comma seperated string of namespaces to skip for unschedulable pods.

Example:
```shell
$ kubectl create -f ./kubernetes/autoscaler.yaml
```
