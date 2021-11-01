package helpers

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func WriteKubeConfig(config *rest.Config) ([]byte, error) {
	return clientcmd.Write(api.Config{
		Clusters: map[string]*api.Cluster{
			config.ServerName: {
				Server:                   config.Host,
				CertificateAuthorityData: config.CAData,
			},
		},
		Contexts: map[string]*api.Context{
			config.ServerName: {
				Cluster:  config.ServerName,
				AuthInfo: config.Username,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			config.Username: {
				ClientKeyData:         config.KeyData,
				ClientCertificateData: config.CertData,
			},
		},
		CurrentContext: config.ServerName,
	})
}

