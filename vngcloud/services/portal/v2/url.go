package v2

import lsclient "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/client"

func listAllQuotaUsedUrl(psc lsclient.IServiceClient) string {
	return psc.ServiceURL(
		psc.GetProjectId(),
		"quotas",
		"quotaUsed")
}
