// This file is auto-generated, don't edit it. Thanks.
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	cas "github.com/alibabacloud-go/cas-20200407/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

func CreateClient() (*cas.Client, error) {
	keyId := os.Getenv("ACCESS_KEY_ID")
	keySecret := os.Getenv("ACCESS_KEY_SECRET")

	if keyId == "" || keySecret == "" {
		return nil, fmt.Errorf("please set ACCESS_KEY_ID and ACCESS_KEY_SECRET")
	}

	config := &openapi.Config{
		AccessKeyId:     tea.String(keyId),
		AccessKeySecret: tea.String(keySecret),
	}

	config.Endpoint = tea.String("cas.aliyuncs.com")
	client := &cas.Client{}
	client, err := cas.NewClient(config)
	return client, err
}

func GetBasicInfo(client *cas.Client, domain string) error {
	req := &cas.ListCloudResourcesRequest{
		Keyword: &domain,
	}

	resp, err := client.ListCloudResourcesWithOptions(req, &util.RuntimeOptions{})
	if err != nil {
		return err
	}

	fmt.Println(resp)
	return nil
}

func printVersion() {
	if info, ok := debug.ReadBuildInfo(); ok {
		var revision string
		var modified bool

		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value[:7]
			}
			if setting.Key == "vcs.modified" {
				modified = setting.Value == "true"
			}
		}

		if revision != "" {
			if modified {
				revision += " (modified)"
			}
			fmt.Printf("AliyunCertRenew version %s\n", revision)
		} else {
			fmt.Println("AliyunCertRenew version unknown")
			fmt.Printf("%+v", info)
		}
	} else {
		fmt.Println("AliyunCertRenew version unknown")
	}
}

func main() {
	printVersion()

	client, err := CreateClient()

	if err != nil {
		fmt.Println(err)
		return
	}

	GetBasicInfo(client, "ali.01c.host")
}
