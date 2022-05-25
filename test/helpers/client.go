package helpers

import (
	"os"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

func NewCSClient() (*cloudstack.CloudStackClient, error) {
	projDir := os.Getenv("PROJECT_DIR")
	conf := cloud.Config{}
	ccPath := projDir + "/cloud-config"
	if rawCfg, err := ini.Load(ccPath); err != nil {
		return nil, errors.Wrapf(err, "reading config at path %s:", ccPath)
	} else if g := rawCfg.Section("Global"); len(g.Keys()) == 0 {
		return nil, errors.New("section Global not found")
	} else if err = rawCfg.Section("Global").StrictMapTo(&conf); err != nil {
		return nil, errors.Wrapf(err, "parsing [Global] section from config at path %s:", ccPath)
	}
	csClient := cloudstack.NewAsyncClient(conf.APIURL, conf.APIKey, conf.SecretKey, conf.VerifySSL)
	return csClient, nil
}
