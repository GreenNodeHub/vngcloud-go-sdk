package opensearch

import (
	lsclient "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/client"
	lsopensearchSvcV1 "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/services/opensearch/v1"
)

func NewOpenSearchServiceV1(psvcClient lsclient.IServiceClient) IOpenSearchServiceV1 {
	return &lsopensearchSvcV1.OpenSearchServiceV1{
		VopensearchClient: psvcClient,
	}
}
