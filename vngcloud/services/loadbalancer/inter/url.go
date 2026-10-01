package inter

import lsclient "github.com/GreenNodeHub/vngcloud-go-sdk/v2/vngcloud/client"

func createLoadBalancerUrl(psc lsclient.IServiceClient) string {
	return psc.ServiceURL("loadBalancers")
}
