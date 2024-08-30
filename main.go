package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	cas "github.com/alibabacloud-go/cas-20200407/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	log "github.com/sirupsen/logrus"
)

const RENEW_THRESHOLD = 7 * 86400

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

func GetBasicInfo(client *cas.Client, domain string) (needRenew bool, results []cas.ListCloudResourcesResponseBodyData, err error) {
	req := &cas.ListCloudResourcesRequest{
		Keyword: &domain,
	}

	resp, err := client.ListCloudResourcesWithOptions(req, &util.RuntimeOptions{})
	if err != nil {
		return
	}
	log.Debugf("ListCloudResources response: %+v", resp)

	for _, entry := range resp.Body.Data {
		if *entry.Domain == domain && *entry.EnableHttps > 0 {
			endTime, err := strconv.ParseInt(*entry.CertEndTime, 10, 64)
			if err != nil {
				log.Errorf("Error parsing cert end time: %s", err)
				continue
			}

			expireTime := endTime/1000 - time.Now().Unix()
			log.Debugf("Cert %d for %s expires in %d secs", *entry.CertId, domain, expireTime)
			if expireTime < RENEW_THRESHOLD {
				needRenew = true
			}

			results = append(results, *entry)
		}
	}

	if len(results) == 0 {
		err = fmt.Errorf("no resources found for %s", domain)
		return
	}

	return
}

func ApplyNewCert(client *cas.Client, domain string) (newCertId int64, err error) {
	createCertReq := &cas.CreateCertificateForPackageRequestRequest{
		Domain:       tea.String(domain),
		ProductCode:  tea.String("digicert-free-1-free"),
		ValidateType: tea.String("DNS"),
	}

	resp, err := client.CreateCertificateForPackageRequestWithOptions(createCertReq, &util.RuntimeOptions{})
	if err != nil {
		return
	}
	log.Debugf("CreateCertificateForPackageRequest response: %+v", resp)
	orderId := resp.Body.OrderId

	log.Info("New certificate request created for ", domain)

	for i := 0; i < 20; i++ {
		time.Sleep(10 * time.Second)

		getOrderReq := &cas.ListUserCertificateOrderRequest{
			Keyword:   tea.String(domain),
			OrderType: tea.String("CPACK"),
		}

		orderResp, err := client.ListUserCertificateOrderWithOptions(getOrderReq, &util.RuntimeOptions{})
		if err != nil {
			return 0, err
		}
		log.Debugf("ListUserCertificateOrder response: %+v", orderResp)

		for _, cpackEntry := range orderResp.Body.CertificateOrderList {
			if *cpackEntry.OrderId == *orderId {
				log.Info("Order current status: ", *cpackEntry.Status)
				if *cpackEntry.Status == "ISSUED" {
					getCertReq := &cas.ListUserCertificateOrderRequest{
						Keyword:   tea.String(domain),
						OrderType: tea.String("CERT"),
					}

					certResp, err := client.ListUserCertificateOrderWithOptions(getCertReq, &util.RuntimeOptions{})
					if err != nil {
						return 0, err
					}

					log.Debugf("ListUserCertificateOrder response: %+v", certResp)
					for _, certEntry := range certResp.Body.CertificateOrderList {
						if *certEntry.InstanceId == *cpackEntry.InstanceId {
							return *certEntry.CertificateId, nil
						}
					}

					return 0, fmt.Errorf("cert not found")
				}
			}
		}
	}

	return 0, fmt.Errorf("timeout waiting for cert to be issued")
}

func DeployCert(client *cas.Client, certId int64, resourceIds []int64) error {
	listContactReq := &cas.ListContactRequest{}
	contactResp, err := client.ListContactWithOptions(listContactReq, &util.RuntimeOptions{})
	if err != nil {
		return err
	}

	log.Debugf("ListContact response: %+v", contactResp)
	if len(contactResp.Body.ContactList) == 0 {
		return fmt.Errorf("no contact found")
	}

	resourceStrIds := make([]string, len(resourceIds))
	for i, id := range resourceIds {
		resourceStrIds[i] = strconv.FormatInt(id, 10)
	}

	createDeployReq := &cas.CreateDeploymentJobRequest{
		Name:        tea.String("aliyun-cert-renew-auto-" + strconv.FormatInt(time.Now().Unix(), 10)),
		ResourceIds: tea.String(strings.Join(resourceStrIds, ",")),
		CertIds:     tea.String(strconv.FormatInt(certId, 10)),
		ContactIds:  tea.String(strconv.FormatInt(*contactResp.Body.ContactList[0].ContactId, 10)),
		JobType:     tea.String("user"),
	}

	deployResp, err := client.CreateDeploymentJobWithOptions(createDeployReq, &util.RuntimeOptions{})
	if err != nil {
		return err
	}
	log.Debugf("CreateDeploymentJob response: %+v", deployResp)
	log.Info("Deployment job created: ", *deployResp.Body.JobId)

	jobId := deployResp.Body.JobId

	time.Sleep(2 * time.Second)
	updateDeployReq := &cas.UpdateDeploymentJobStatusRequest{
		JobId:  jobId,
		Status: tea.String("scheduling"),
	}

	updateDeployResp, err := client.UpdateDeploymentJobStatusWithOptions(updateDeployReq, &util.RuntimeOptions{})
	if err != nil {
		return err
	}
	log.Debugf("UpdateDeploymentJobStatus response: %+v", updateDeployResp)
	log.Info("Deployment job submitted: ", *jobId)

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

	if os.Getenv("DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
		log.SetFormatter(&log.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := f.File[strings.LastIndex(f.File, string(os.PathSeparator))+1:]
				caller := strings.Replace(fmt.Sprintf("%s:%d", filename, f.Line), ".go", "", 1)
				return "", "[" + caller + strings.Repeat(" ", max(12-utf8.RuneCountInString(caller), 0)) + "]"
			},
		})
	}

	log.Info("AliyunCertRenew starting...")
	client, err := CreateClient()

	if err != nil {
		log.Fatal(err)
	}

	domainEnv := os.Getenv("DOMAIN")
	if domainEnv == "" {
		log.Fatal("no domain specified, exiting...")
	}

	domainList := strings.Split(domainEnv, ",")
	log.Info("Domains to check: ", domainList)

	for _, domain := range domainList {
		log.Infof(">>> Checking %s", domain)
		needRenew, resources, err := GetBasicInfo(client, domain)

		if err != nil {
			log.Error("Error fetching status for ", domain, ": ", err)
			continue
		}

		if !needRenew {
			log.Infof("No renewal needed for %s", domain)
			continue
		}

		log.Info("Certificate renewal needed for ", domain)
		newCertId, err := ApplyNewCert(client, domain)
		if err != nil {
			log.Error("Error applying new cert for ", domain, ": ", err)
			continue
		}
		log.Info("New cert created for ", domain, ": ", newCertId)

		resourceIds := make([]int64, len(resources))
		for i, res := range resources {
			resourceIds[i] = *res.Id
		}
		err = DeployCert(client, newCertId, resourceIds)

		if err != nil {
			log.Error("Error deploying new cert for ", domain, ": ", err)
			continue
		}
	}
}
