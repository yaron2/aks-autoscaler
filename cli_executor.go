package main

import (
	"fmt"
	"os"
	"os/exec"
)

type CliExecutor struct {
	AppID         string
	TenantID      string
	Password      string
	AksName       string
	ResourceGroup string
	LoggedIn      bool
}

func NewCliExecutor(appID string, password string, tenantID string, aksName string, resourceGroup string) *CliExecutor {
	if appID == "" {
		panic("appId cannot be empty")
	} else if password == "" {
		panic("password cannot be empty")
	} else if tenantID == "" {
		panic("tenantId cannot be empty")
	} else if aksName == "" {
		panic("aksName cannot be empty")
	} else if resourceGroup == "" {
		panic("resourceGroup cannot be empty")
	} else {
		return &CliExecutor{AppID: appID, Password: password, TenantID: tenantID, AksName: aksName, ResourceGroup: resourceGroup}
	}
}

func (c *CliExecutor) Scale(agents int32) {
	// TODO: Parse return message and return true or false
	if c.LoggedIn {
		fmt.Println("scaling to " + fmt.Sprint(agents) + " nodes")

		cmd := "az"
		cmdArgs := []string{"aks", "scale", "-g", c.ResourceGroup, "-n", c.AksName, "--node-count", fmt.Sprint(agents)}
		_, err := exec.Command(cmd, cmdArgs...).Output()

		if err != nil {
			fmt.Println(err.Error())
		}

		fmt.Println("scale operation successful")
	} else {
		panic("Must be logged in before executing commands")
	}
}

func (c *CliExecutor) Login() {
	if !c.LoggedIn {
		cmd := "az"
		cmdArgs := []string{"login", "--service-principal", "-u", c.AppID, "-p", c.Password, "-t", c.TenantID}
		if err := exec.Command(cmd, cmdArgs...).Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			panic("az login failed")
		}

		fmt.Println("login successful")
		c.LoggedIn = true
	}
}
