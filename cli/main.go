package main

import (
	"github.com/urfave/cli"
	"github.com/ArthurHlt/yint"
	"strings"
	"fmt"
	"os"
	"io/ioutil"
)

func main() {
	app := cli.NewApp()
	app.Name = "yint"
	app.Version = "1.0.0"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Arthur Halet",
			Email: "arthurh.halet@gmail.com",
		},
	}

	cli.VersionFlag = cli.BoolFlag{
		Name:  "version",
		Usage: "print the version",
	}

	flags := []cli.Flag{
		cli.StringSliceFlag{
			Name:  "ops-file, o",
			Usage: "Load manifest operations from a YAML file",
		},
		cli.StringSliceFlag{
			Name:  "var, v",
			Usage: "Set variable",
		},
		cli.StringSliceFlag{
			Name:  "var-file",
			Usage: "Set variable to file contents",
		},
		cli.StringSliceFlag{
			Name:  "vars-file, l",
			Usage: "Load variables from a YAML file",
		},
		cli.StringSliceFlag{
			Name:  "vars-env",
			Usage: "Load variables from environment variables (e.g.: 'MY' to load MY_var=value)",
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "Extract value out of template (e.g.: /private_key)",
		},
		cli.BoolFlag{
			Name:  "var-errs",
			Usage: "Expect all variables to be found, otherwise error",
		},
		cli.BoolFlag{
			Name:  "var-errs-unused",
			Usage: "Expect all variables to be used, otherwise error",
		},
		cli.BoolFlag{
			Name:  "stdin",
			Usage: "Read template from stdin",
		},
	}

	app.Flags = flags
	app.Action = Run
	app.Commands = []cli.Command{
		{
			Name:    "interpolate",
			Aliases: []string{"int"},
			Action:  Run,
			Flags:   flags,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "yint: %s\n", err.Error())
		os.Exit(1)
	}
}

func Run(c *cli.Context) error {
	varsKV, err := convertVarsKV(c.StringSlice("var"))
	if err != nil {
		return err
	}
	opts := yint.ApplyOpts{
		OpsFiles:        c.StringSlice("ops-file"),
		VarsKV:          varsKV,
		VarsFiles:       c.StringSlice("vars-file"),
		VarFiles:        c.StringSlice("var-file"),
		VarsEnv:         c.StringSlice("vars-env"),
		OpPath:          c.String("path"),
		VarErrors:       c.Bool("var-errs"),
		VarErrorsUnused: c.Bool("var-errs-unused"),
	}

	yamlPath := c.Args().First()
	if c.Bool("stdin") {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		opts.YamlContent = b
	} else {
		opts.YamlPath = yamlPath
	}

	b, err := yint.Apply(opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.App.Writer, string(b))
	return nil
}

func convertVarsKV(kvRawSlice []string) (map[string]interface{}, error) {
	varsKv := make(map[string]interface{})

	for _, kvRaw := range kvRawSlice {
		pieces := strings.SplitN(kvRaw, "=", 2)
		if len(pieces) != 2 {
			return varsKv, fmt.Errorf("Expected var '%s' to be in format 'name=path'", kvRaw)
		}
		if len(pieces[0]) == 0 {
			return varsKv, fmt.Errorf("Expected var '%s' to specify non-empty name", kvRaw)
		}

		if len(pieces[1]) == 0 {
			return varsKv, fmt.Errorf("Expected var '%s' to specify non-empty path", kvRaw)
		}
		varsKv[pieces[0]] = pieces[1]
	}
	return varsKv, nil
}
