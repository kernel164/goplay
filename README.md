goplay (simple ansible-playbook wrapper in go)
======

[![Build Status](https://drone.io/github.com/kernel164/goplay/status.png)](https://drone.io/github.com/kernel164/goplay/latest)

I wanted to place all the ansible-playbook command line arguments in a config file so that its easy to manage and ended up creating my first go application - goplay.

goplay supports all the ansible-playbook command line options and it can be configured in a single file - play.xml and do "goplay command" - similar to "fig up" 

ansible.cfg, inventory, vars, playbook all can be placed in a single file for easy management. Sample play.yml is shown below.

**File: play.yml**
```yml
command1:
  verbose: vvvv
  user: xyz
  sudo: true
  sudo_user: root
  ask_pass: false
  tags:
    - test
  ansible_cfg: |
    [defaults]
    roles_path = ./roles
  inventory: |
    [all]
    the.host.com
  vars: |
    packer_version: 0.5.2
    vagrant_version: 1.5.2
  playbook: |
    - hosts: "{{ hosts|default('all') }}"
      gather_facts: "{{ gather_facts|default(true) }}"
      roles:
        - packer
        - vagrant
```


```bash
$ goplay command1
ansible-playbook --extra-vars /tmp/goplay-7adb62018ba6f78b6e53fc4a5760cfec-vars --inventory-file /tmp/goplay-7ef0b9f2ff60c1ca61a649fb6fe747e7-inventory --sudo --sudo-user root --tags test --user xyz -vvvv /tmp/goplay-760d64134937fa1774ae0ab3f06f47d9-playbook
PLAY [all] ******************************************************************** 

GATHERING FACTS *************************************************************** 
<the.host.com> ESTABLISH CONNECTION FOR USER: xyz
<the.host.com> REMOTE_MODULE setup
....
```
