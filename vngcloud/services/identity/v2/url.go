package v2

import (
	lsclient "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/client"
)

func getAccessTokenUrl(psc lsclient.IServiceClient) string {
	return psc.ServiceURL("auth", "token")
}
