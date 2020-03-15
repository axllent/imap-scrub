package lib

import (
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	// Config module global
	Config = YamlConfig{}

	validActions = map[string]bool{
		"delete":             true,
		"save_attachments":   true,
		"remove_attachments": true,
	}
)

// YamlConfig config struct
type YamlConfig struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	SSL      *bool  `yaml:"ssl"`
	Port     *int   `yaml:"port"`
	User     string `yaml:"user"`
	Pass     string `yaml:"pass"`
	SavePath string `yaml:"save_path"`
	UseTrash bool   `yaml:"use_trash"`
	Rules    []Rule `yaml:"rules"`
}

// Rule struct
type Rule struct {
	Mailbox        string `yaml:"mailbox"`
	Size           uint32 `yaml:"min_size"`   // KB
	OlderThan      int    `yaml:"older_than"` // days
	From           string `yaml:"from"`
	To             string `yaml:"to"`
	Subject        string `yaml:"subject"`
	Body           string `yaml:"body"`
	Text           string `yaml:"text"`
	Actions        string `yaml:"actions"`
	IncludeUnread  bool   `yaml:"include_unread"`
	IncludeStarred bool   `yaml:"include_starred"`
}

// ReadConfig reads & parses the config into global config
func ReadConfig(file string) {
	yamlData, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(yamlData, &Config)

	if err != nil {
		Log.ErrorF("Error parsing %s:\n\n%s\n\n", file, err)
		os.Exit(2)
	}

	if Config.User == "" || Config.Pass == "" || Config.Host == "" {
		Log.Error("Please ensure host, user & password are set")
		os.Exit(2)
	}

	if Config.SSL == nil {
		ssl := true
		Config.SSL = &ssl
	}

	if Config.Port == nil {
		port := 143
		if *Config.SSL {
			port = 993
		}
		Config.Port = &port
	}

	// change kB to bytes
	for x, item := range Config.Rules {
		Config.Rules[x].Size = item.Size * 1024

		if item.Mailbox == "" {
			Log.Error("You must specify a mailbox for every rule")
			os.Exit(2)
		}

		if item.Actions == "" {
			Log.Error("You must have at least one action per rule")
			os.Exit(2)
		}

		raw := strings.ToLower(item.Actions)
		rawActions := strings.Split(raw, ",")
		actions := []string{}
		for _, a := range rawActions {
			a = strings.TrimSpace(a)
			if _, ok := validActions[a]; !ok {
				Log.ErrorF("\"%s\" is not a valid action", a)
				os.Exit(2)
			}
			actions = append(actions, a)
		}
		Config.Rules[x].Actions = strings.Join(actions, ", ")

		if Config.Rules[x].Delete() && Config.Rules[x].RemoveAttachments() {
			Log.Error("Your rule cannot contain both remove_attachments and delete")
			os.Exit(2)
		}
	}

	// if len(Config.Rules) == 0 {
	// 	Log.Error("You must have at least one rule")
	// 	os.Exit(2)
	// }
}

// Delete returns whether a rule is set to delete messages
func (r Rule) Delete() bool {
	return strings.Contains(r.Actions, "delete")
}

// RemoveAttachments returns whether a rule is set to delete messages
func (r Rule) RemoveAttachments() bool {
	return strings.Contains(r.Actions, "remove_attachments")
}

// SaveAttachments returns whether a rule is set to delete messages
func (r Rule) SaveAttachments() bool {
	return strings.Contains(r.Actions, "save_attachments")
}
