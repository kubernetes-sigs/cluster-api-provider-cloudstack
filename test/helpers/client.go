package helpers

import (
	"bytes"
	"io"
	"os"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"gopkg.in/yaml.v3"
)

type CSConf struct {
	APIKey    string `yaml:"api-key"`
	SecretKey string `yaml:"secret-key"`
	APIUrl    string `yaml:"api-url"`
	VerifySSL string `yaml:"verify-ssl"`
}

type Config struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Tind       string            `yaml:"type"`
	Metadata   map[string]string `yaml:"metadata"`
	StringData CSConf            `yaml:"stringData"`
}

func UnmarshalAllConfigs(in []byte, out *[]Config) error {
	r := bytes.NewReader(in)
	decoder := yaml.NewDecoder(r)
	for {
		var conf Config
		if err := decoder.Decode(&conf); err != nil {
			// Break when there are no more documents to decode
			if err != io.EOF {
				return err
			}
			break
		}
		*out = append(*out, conf)
	}
	return nil
}

// NewCSClient creates a CloudStack-Go client from the cloud-config file.
func NewCSClient() (*cloudstack.CloudStackClient, error) {
	content, err := os.ReadFile(os.Getenv("PROJECT_DIR") + "/cloud-config.yaml")
	if err != nil {
		return nil, err
	}
	configs := &[]Config{}
	if err := UnmarshalAllConfigs(content, configs); err != nil {
		return nil, err
	}
	config := (*configs)[0].StringData

	return cloudstack.NewAsyncClient(config.APIUrl, config.APIKey, config.SecretKey, config.VerifySSL == "true"), nil
}
