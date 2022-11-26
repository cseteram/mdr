package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type config struct {
	Secrets struct {
		DeveloperKey string `yaml:"developerKey"`
	} `yaml:"secrets"`

	Postgres struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Dbname   string `yaml:"dbname"`
	} `yaml:"postgres"`

	Profile struct {
		Nickname  string `yaml:"nickname"`
		AvatarURL string `yaml:"avatarUrl"`
	} `yaml:"profile"`

	Notifications []struct {
		Name       string `yaml:"name"`
		ChannelID  string `yaml:"channelId"`
		WebhookURL string `yaml:"webhookUrl"`
	} `yaml:"notifications"`
}

func Parse(path string) (*config, error) {
	ret := new(config)

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read %v", path)
	}

	err = yaml.Unmarshal(yamlFile, ret)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal %v", path)
	}

	return ret, nil
}
