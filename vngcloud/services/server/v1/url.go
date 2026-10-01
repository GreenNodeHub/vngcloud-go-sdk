package v1

import lsclient "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/client"

func createSystemTagUrl(psc lsclient.IServiceClient) string {
	return psc.ServiceURL(
		psc.GetProjectId(),
		"tags")
}
