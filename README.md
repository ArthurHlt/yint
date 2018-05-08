# Yint [![Build Status](https://travis-ci.org/ArthurHlt/yint.svg?branch=master)](https://travis-ci.org/ArthurHlt/yint)

Yint is a small library easy to use with a cli implementation which works the same way as https://bosh.io/docs/cli-v2/#misc (see interpolate) .

It uses same implementation as you may found at https://github.com/cloudfoundry/bosh-cli (some code come from directly from 
it but without heavy dependencies).

Yaml patching use https://github.com/cppforlife/go-patch/ library.


## Use as a library

Example:

```go
package main

import (
	"github.com/ArthurHlt/yint"
	"fmt"
)

var toto string = `
toto:
  key1: val1
`
var totoOps string = `
---

- type: replace
  path: /toto/key1
  value: titi
`

func main() {
	b, err := yint.Apply(yint.ApplyOpts{
		YamlContent: []byte(toto),
		YamlPath:    "manifest.yml",
		OpsFiles:    []string{"ops-file.yml", "ops-file2.yml"},
		OpsContent:  []byte(totoOps),

		//VarsKV: map[string]interface{}{
		//	"foo": "bar",
		//},
		//VarFiles: "foo=./content",
		//VarsEnv: []string{"MY"},
		VarFiles: []string{"vars.yml"},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
```

Options details:

```go
type ApplyOpts struct {
	YamlContent     []byte                 // Template byte content that will be interpolated (will be append to YamlPath if exists)
	YamlPath        string                 // Path to a template that will be interpolated (will be append to YamlContent if not empty)
	OpsContent      []byte                 // Load manifest operations from byte content (will be append with loaded OpsFiles if exists)
	OpsFiles        []string               // Load manifest operations from one or more YAML file(s) (will be append with loaded OpsContent if exists)
	VarsKV          map[string]interface{} // Set variable to inject
	VarFiles        []string               // Set variable to file contents
	VarsFiles       []string               // Load variables from a YAML file
	VarsEnv         []string               // Load variables from environment variables (e.g.: 'MY' to load MY_var=value)
	OpPath          string                 // Extract value out of template (e.g.: /private_key)
	VarErrors       bool                   // Expect all variables to be found, otherwise error
	VarErrorsUnused bool                   // Expect all variables to be used, otherwise error
}
```

## Install the cli


### On *nix system

You can install this via the command-line with either `curl` or `wget`.

#### via curl

```bash
$ sh -c "$(curl -fsSL https://raw.github.com/ArthurHlt/yint/master/bin/install.sh)"
```

#### via wget

```bash
$ sh -c "$(wget https://raw.github.com/ArthurHlt/yint/master/bin/install.sh -O -)"
```

### On windows

You can install it by downloading the `.exe` corresponding to your cpu from releases page: https://github.com/ArthurHlt/yint/releases .
Alternatively, if you have terminal interpreting shell you can also use command line script above, it will download file in your current working dir.

### From go command line

Simply run in terminal:

```bash
$ go get github.com/ArthurHlt/yint/cli/yint
```

## Use the cli

### Like `bosh int`

`yint interpolate manifest.yml [-v ...] [-o ...] [--path op-path] [--stdin]` (Alias: `int`)

Interpolates variables into a manifest sending result to stdout. Operation files and variables can be provided to adjust and fill in template.

--path flag can be used to extract portion of a YAML document.

Example:
```bash
$ yint int bosh-deployment/bosh.yml \
  -o bosh-deployment/virtualbox/cpi.yml \
  -o bosh-deployment/virtualbox/outbound-network.yml \
  -o bosh-deployment/bosh-lite.yml \
  -o bosh-deployment/jumpbox-user.yml \
  -v director_name=vbox \
  -v internal_ip=192.168.56.6 \
  -v internal_gw=192.168.56.1 \
  -v internal_cidr=192.168.56.0/24 \
  -v network_name=vboxnet0 \
  -v outbound_network_name=NatNetwork

$ yint int creds.yml --path /admin_password
skh32i7rdfji4387hg

$ yint int creds.yml --path /director_ssl/ca
-----BEGIN CERTIFICATE-----
...
```

### Simplest form

`yint [-v ...] [-o ...] [--path op-path] [--stdin] manifest.yml`

Example:
```bash
$ yint -o bosh-deployment/virtualbox/cpi.yml \
  -o bosh-deployment/virtualbox/outbound-network.yml \
  -o bosh-deployment/bosh-lite.yml \
  -o bosh-deployment/jumpbox-user.yml \
  -v director_name=vbox \
  -v internal_ip=192.168.56.6 \
  -v internal_gw=192.168.56.1 \
  -v internal_cidr=192.168.56.0/24 \
  -v network_name=vboxnet0 \
  -v outbound_network_name=NatNetwork \
  bosh-deployment/bosh.yml

$ yint --path /admin_password creds.yml
skh32i7rdfji4387hg

$ yint --path /director_ssl/ca creds.yml
-----BEGIN CERTIFICATE-----
...
```

