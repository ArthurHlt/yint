package yint

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"os/user"
	"github.com/cppforlife/go-patch/patch"
	"io"
	"bytes"
)

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

func Apply(opts ApplyOpts) ([]byte, error) {
	result := make([]byte, 0)
	if len(opts.YamlContent) == 0 && opts.YamlPath == "" {
		return result, fmt.Errorf("A yaml template must be given.")
	}

	if len(opts.OpsContent) == 0 && len(opts.OpsFiles) == 0 {
		return result, fmt.Errorf("An ops-file must be given.")
	}

	if opts.VarsKV == nil {
		opts.VarsKV = make(map[string]interface{})
	}

	yamlContent, err := loadAllType(opts.YamlContent, opts.YamlPath)
	if err != nil {
		return result, err
	}

	opsContent, err := loadAllType(opts.OpsContent, opts.OpsFiles...)
	if err != nil {
		return result, err
	}

	ops, err := loadOps(opsContent)
	if err != nil {
		return result, err
	}

	vars := opts.VarsKV
	varsEnv, err := loadVarsEnvs(opts.VarsEnv...)
	if err != nil {
		return result, err
	}
	varsFiles, err := loadVarsFiles(opts.VarFiles...)
	if err != nil {
		return result, err
	}
	varsFile, err := loadVarFiles(opts.VarFiles...)
	if err != nil {
		return result, err
	}
	vars = mergeMap(vars, varsEnv, varsFiles, varsFile)

	tpl := NewTemplate(yamlContent)

	evalOpts := EvaluateOpts{
		ExpectAllKeys:     opts.VarErrors,
		ExpectAllVarsUsed: opts.VarErrorsUnused,
	}

	if opts.OpPath != "" {
		ptr, err := patch.NewPointerFromString(opts.OpPath)
		if err != nil {
			return result, err
		}
		evalOpts.PostVarSubstitutionOp = patch.FindOp{Path: ptr}

		// Printing YAML indented multiline strings (eg SSH key) is not useful
		evalOpts.UnescapedMultiline = true
	}

	result, err = tpl.Evaluate(StaticVariables(vars), ops, evalOpts)
	if err != nil {
		return result, fmt.Errorf("Evaluating: %s", err.Error())
	}
	return result, nil
}

func mergeMap(s map[string]interface{}, as ... map[string]interface{}) map[string]interface{} {
	for _, a := range as {
		for k, v := range a {
			s[k] = v
		}
	}
	return s
}

func loadOps(data []byte) (patch.Ops, error) {
	opDefs := make([]patch.OpDefinition, 0)
	dec := yaml.NewDecoder(bytes.NewBuffer(data))
	for {
		var opsDefsTmp []patch.OpDefinition
		err := dec.Decode(&opsDefsTmp)
		if err == io.EOF {
			break
		}
		opDefs = append(opDefs, opsDefsTmp...)
	}

	return patch.NewOpsFromDefinitions(opDefs)
}

func loadVarFiles(varFiles ... string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for _, varFile := range varFiles {
		r, err := loadVarFile(varFile)
		if err != nil {
			return result, err
		}
		result = mergeMap(result, r)
	}
	return result, nil
}

func loadVarFile(varFile string) (map[string]interface{}, error) {
	if varFile == "" {
		return map[string]interface{}{}, nil
	}
	pieces := strings.SplitN(varFile, "=", 2)
	if len(pieces) != 2 {
		return map[string]interface{}{}, fmt.Errorf("Expected var '%s' to be in format 'name=path'", varFile)
	}

	if len(pieces[0]) == 0 {
		return map[string]interface{}{}, fmt.Errorf("Expected var '%s' to specify non-empty name", varFile)
	}

	if len(pieces[1]) == 0 {
		return map[string]interface{}{}, fmt.Errorf("Expected var '%s' to specify non-empty path", varFile)
	}

	path := pieces[1]

	b, err := loadFile(path)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("Reading variable from file '%s': %s", path, err.Error())
	}
	return map[string]interface{}{
		pieces[0]: string(b),
	}, nil
}

func loadVarsFiles(files ... string) (map[string]interface{}, error) {
	b, err := loadFiles(files...)
	if err != nil {
		return map[string]interface{}{}, err
	}
	var vars map[string]interface{}

	err = yaml.Unmarshal(b, &vars)
	return vars, err
}

func loadVarsEnvs(prefixes ... string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, prefix := range prefixes {
		v, err := loadVarsEnv(prefix)
		if err != nil {
			return result, err
		}
		result = mergeMap(result, v)
	}
	return result, nil
}

func loadVarsEnv(prefix string) (map[string]interface{}, error) {
	vars := map[string]interface{}{}

	for _, envVar := range os.Environ() {
		pieces := strings.SplitN(envVar, "=", 2)
		if len(pieces) != 2 {
			return vars, fmt.Errorf("Expected environment variable to be key-value pair")
		}

		if !strings.HasPrefix(pieces[0], prefix+"_") {
			continue
		}

		var val interface{}

		err := yaml.Unmarshal([]byte(pieces[1]), &val)
		if err != nil {
			return vars, fmt.Errorf("Deserializing YAML from environment variable '%s': %s", pieces[0], err.Error())
		}

		vars[strings.TrimPrefix(pieces[0], prefix+"_")] = val
	}

	return vars, nil
}

func loadAllType(b []byte, files ... string) ([]byte, error) {
	content, err := loadFiles(files...)
	if err != nil {
		return []byte{}, err
	}
	return appendYmlByte(b, content), nil
}

func appendYmlByte(s []byte, as ... []byte) []byte {
	result := make([]byte, 0)
	if len(s) != 0 {
		result = s
	}
	for _, a := range as {
		if len(result) > 0 {
			result = append(result, byte('\n'))
		}
		result = append(result, a...)
	}
	return result
}

func loadFiles(paths ... string) ([]byte, error) {
	result := make([]byte, 0)
	for _, path := range paths {
		b, err := loadFile(path)
		if err != nil {
			return []byte{}, err
		}
		result = appendYmlByte(result, b)
	}
	return result, nil
}

func loadFile(path string) ([]byte, error) {
	if path == "" {
		return []byte{}, nil
	}
	var err error
	path, err = expandPath(path)
	if err != nil {
		return []byte{}, fmt.Errorf("Expanding path for file '%s': %s", path, err.Error())
	}
	f, err := os.Open(path)
	if err != nil {
		return []byte{}, fmt.Errorf("Loading file '%s': %s", path, err.Error())
	}

	return ioutil.ReadAll(f)
}

func expandPath(path string) (string, error) {

	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("Getting current user home dir: %s", err.Error())
		}
		path = filepath.Join(usr.HomeDir, path[1:])
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("Getting absolute path: %s", err.Error())
	}

	return path, nil
}
