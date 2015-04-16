package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"gopkg.in/codegangsta/cli.v1"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	//"text/template"
)

type Config struct {
	Ansible_path        string
	Ansible_Cfg_file    string
	Ansible_Cfg         string
	Var_file            string
	Vars                string
	Inventory_file      string
	Inventory           string
	Limit               string
	Module_path         string
	Private_key_file    string
	Sudo                bool
	Sudo_user           string
	Step                bool
	Su                  bool
	Su_user             string
	Tags                []string
	Skip_tags           []string
	User                string
	Timeout_in_secs     int64
	Vault_password_file string
	Start_at_task       string
	Playbook_file       string
	Playbook            string
	Verbose             string
	Forks               int32
	Connection          string
	Ask_pass            bool
	Ask_su_pass         bool
	Ask_sudo_pass       bool
	Ask_vault_pass      bool
}

var TmpFiles map[string]bool = make(map[string]bool)
var md5Hash = md5.New()

func check(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
		//panic(e)
	}
}

func expandValue(text string) string {
	return os.ExpandEnv(text)
	//tmpl, terr := template.New("test").Parse(text)
	//check(terr)
	//err := tmpl.Execute(os.Stdout, obj)
	//check(err)
	//return text
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func newTmpFile(content string, suffix string) string {
	// fmt.Sprintf("%x", time.Now().UTC().UnixNano())
	newFile := "/tmp/goplay-" + GetMD5Hash(content) + "-" + suffix
	TmpFiles[newFile] = true
	return newFile
}

func cleanup() {
	fmt.Println("removing....")
	for k, _ := range TmpFiles {
		fmt.Printf("removing %s\n", k)
		os.Remove(k)
	}
}

func main() {
	// defer cleanup()
	app := cli.NewApp()
	app.Name = "goplay"
	app.Usage = "simple ansible-playbook wrapper"
	app.Version = "0.1.3"
	app.Author = "nobody"
	app.Usage = "goplay [global options] command"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "file, f", Value: "play.yml", Usage: "play file"},
		cli.StringFlag{Name: "env-file, e", Value: "env.yml", Usage: "env file"},
		cli.StringSliceFlag{Name: "env, E", Value: &cli.StringSlice{}, Usage: "env variables"},
		cli.StringSliceFlag{Name: "tag, T", Value: &cli.StringSlice{}, Usage: "tags"},
	}
	app.Action = func(c *cli.Context) {
		cmd := c.Args().First()
		file := c.String("file")
		efile := c.String("env-file")
		envVars := c.StringSlice("env")
		tags := c.StringSlice("tag")

		// check config file
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("Can't find %s. Are you in the right directory?\n", file)
			os.Exit(1)
		}

		if len(cmd) == 0 {
			fmt.Printf("default settings not found?\n")
			os.Exit(1)
		}

		// check ansible-playbook path
		ansible, berr := exec.LookPath("ansible-playbook")
		check(berr)

		// check env file (use only if present)
		if _, env_file_err := os.Stat(efile); !os.IsNotExist(env_file_err) {
			envfiledata, envioerr := ioutil.ReadFile(efile)
			check(envioerr)
			envmap := make(map[interface{}]interface{})
			envyerr := yaml.Unmarshal([]byte(envfiledata), &envmap)
			check(envyerr)
			for k, v := range envmap {
				serr := os.Setenv(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v))
				check(serr)
			}
		}
		for _, envVal := range envVars {
			envVals := strings.Split(envVal, "=")
			serr := os.Setenv(envVals[0], envVals[1])
			check(serr)
		}

		// read the config file.
		filedata, ioerr := ioutil.ReadFile(file)
		check(ioerr)

		parsedCmdConfig := Config{}
		m := make(map[interface{}]interface{})
		yerr := yaml.Unmarshal([]byte(filedata), &m)
		check(yerr)
		//fmt.Printf("--- m:\n%v\n\n", m)
		cmdConfig, ok := m[cmd]
		if !ok {
			fmt.Printf("Can't find '%s' in %s. Have you defined config for '%s' in the file?\n", cmd, file, cmd)
			os.Exit(1)
		}

		typedCmdConfig := cmdConfig.(map[interface{}]interface{})

		//fmt.Printf("--- m.deploy:\n%v\n\n", typedCmdConfig)
		cmdConfigRawData, ymerr := yaml.Marshal(&typedCmdConfig)
		check(ymerr)
		//fmt.Printf("--- m.deploy dump:\n%s\n\n", string(cmdConfigRawData))

		uerr := yaml.Unmarshal(cmdConfigRawData, &parsedCmdConfig)
		check(uerr)
		//fmt.Printf("--- t:\n%v\n\n", parsedCmdConfig)

		// run ansible
		args := []string{"ansible-playbook"}
		if parsedCmdConfig.Ask_pass {
			args = append(args, "--ask-pass")
		}
		if parsedCmdConfig.Ask_su_pass {
			args = append(args, "--ask-su-pass")
		}
		if parsedCmdConfig.Ask_sudo_pass {
			args = append(args, "--ask-sudo-pass")
		}
		if parsedCmdConfig.Ask_vault_pass {
			args = append(args, "--ask-vault-pass")
		}
		// --check
		if len(parsedCmdConfig.Connection) > 0 {
			args = append(args, "--connection", expandValue(parsedCmdConfig.Connection))
		}
		// --diff
		if len(parsedCmdConfig.Var_file) > 0 {
			args = append(args, "--extra-vars", "@" + expandValue(parsedCmdConfig.Var_file))
		} else if len(parsedCmdConfig.Vars) > 0 {
			expandedVarsContent := expandValue(parsedCmdConfig.Vars)
			tmpVarsFile := newTmpFile(expandedVarsContent, "vars")
			iwerr := ioutil.WriteFile(tmpVarsFile, []byte(expandedVarsContent), 0644)
			check(iwerr)
			args = append(args, "--extra-vars", "@" + tmpVarsFile)
		}
		if parsedCmdConfig.Forks > 0 {
			args = append(args, "--forks", string(parsedCmdConfig.Forks))
		}
		if len(parsedCmdConfig.Inventory_file) > 0 {
			args = append(args, "--inventory-file", parsedCmdConfig.Inventory_file)
		} else if len(parsedCmdConfig.Inventory) > 0 {
			expandedInventoryContent := expandValue(parsedCmdConfig.Inventory)
			tmpHostsFile := newTmpFile(expandedInventoryContent, "inventory")
			iwerr := ioutil.WriteFile(tmpHostsFile, []byte(expandedInventoryContent), 0644)
			check(iwerr)
			args = append(args, "--inventory-file", tmpHostsFile)
		}
		if len(parsedCmdConfig.Limit) > 0 {
			args = append(args, "--limit", parsedCmdConfig.Limit)
		}
		if len(parsedCmdConfig.Module_path) > 0 {
			args = append(args, "--module-path", expandValue(parsedCmdConfig.Module_path))
		}
		if len(parsedCmdConfig.Private_key_file) > 0 {
			args = append(args, "--private-key", expandValue(parsedCmdConfig.Private_key_file))
		}
		if len(parsedCmdConfig.Skip_tags) > 0 {
			args = append(args, "--skip-tags", expandValue(strings.Join(parsedCmdConfig.Skip_tags, ",")))
		}
		if len(parsedCmdConfig.Start_at_task) > 0 {
			args = append(args, "--start-at-task", parsedCmdConfig.Start_at_task)
		}
		if parsedCmdConfig.Step {
			args = append(args, "--step")
		}
		if parsedCmdConfig.Su {
			args = append(args, "--su")
		}
		if len(parsedCmdConfig.Su_user) > 0 {
			args = append(args, "--su-user", expandValue(parsedCmdConfig.Su_user))
		}
		if parsedCmdConfig.Sudo {
			args = append(args, "--sudo")
		}
		if len(parsedCmdConfig.Sudo_user) > 0 {
			args = append(args, "--sudo-user", expandValue(parsedCmdConfig.Sudo_user))
		}
		if len(parsedCmdConfig.Tags) > 0 {
			args = append(args, "--tags", expandValue(strings.Join(parsedCmdConfig.Tags, ",")))
		}
		if len(tags) > 0 {
			args = append(args, "--tags", expandValue(strings.Join(tags, ",")))
		}
		if parsedCmdConfig.Timeout_in_secs > 0 {
			args = append(args, "--timeout", string(parsedCmdConfig.Timeout_in_secs))
		}
		if len(parsedCmdConfig.User) > 0 {
			args = append(args, "--user", expandValue(parsedCmdConfig.User))
		}
		if len(parsedCmdConfig.Vault_password_file) > 0 {
			args = append(args, "--vault-password-file", expandValue(parsedCmdConfig.Vault_password_file))
		}
		if len(parsedCmdConfig.Verbose) > 0 {
			args = append(args, "-"+expandValue(parsedCmdConfig.Verbose))
		}
		if len(parsedCmdConfig.Playbook_file) > 0 {
			args = append(args, expandValue(parsedCmdConfig.Playbook_file))
		} else if len(parsedCmdConfig.Playbook) > 0 {
			expandedPlaybookContent := expandValue(parsedCmdConfig.Playbook)
			tmpPlaybookFile := newTmpFile(expandedPlaybookContent, "playbook")
			iwerr := ioutil.WriteFile(tmpPlaybookFile, []byte(expandedPlaybookContent), 0644)
			check(iwerr)
			args = append(args, tmpPlaybookFile)
		}
		fmt.Printf("%v\n", strings.Join(args, " "))
		env := os.Environ()
		if len(parsedCmdConfig.Ansible_Cfg_file) > 0 {
			env = append(env, "ANSIBLE_CONFIG="+expandValue(parsedCmdConfig.Ansible_Cfg_file))
		} else if len(parsedCmdConfig.Ansible_Cfg) > 0 {
			expandedAnsibleCfgContent := expandValue(parsedCmdConfig.Ansible_Cfg)
			tmpCfgFile := newTmpFile(expandedAnsibleCfgContent, "ansible.cfg")
			iwerr := ioutil.WriteFile(tmpCfgFile, []byte(expandedAnsibleCfgContent), 0644)
			check(iwerr)
			env = append(env, "ANSIBLE_CONFIG="+tmpCfgFile)
		}
		exeerr := syscall.Exec(ansible, args, env)
		check(exeerr) // not reachable
	}

	app.Run(os.Args)
}
