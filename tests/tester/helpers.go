package tester

import (
	"encoding/json"
	"os"
	"path/filepath"

	"get.porter.sh/porter/pkg/claims"
	"get.porter.sh/porter/tests/testdata"
	"github.com/stretchr/testify/require"
)

var testBundleBuilt = false

// PrepareTestBundle ensures that the mybuns test bundle has been built.
func (t Tester) PrepareTestBundle() {
	// This variable isn't set on windows and the mybuns bundle relies on it
	os.Setenv("USER", "porterci")

	// Build and publish an interesting test bundle and its dependency
	t.MakeTestBundle(testdata.MyDb, testdata.MyDbRef)
	t.MakeTestBundle(testdata.MyBuns, testdata.MyBunsRef)

	// Import a parameter and credential set for the bundle into the global namespace
	t.RequirePorter("parameters", "apply", filepath.Join(t.RepoRoot, "tests/testdata/params/mybuns.yaml"), "--namespace=")
	t.RequirePorter("credentials", "apply", filepath.Join(t.RepoRoot, "tests/testdata/creds/mybuns.yaml"), "--namespace=")
}

func (t Tester) MakeTestBundle(name string, ref string) {
	pwd, _ := os.Getwd()
	defer os.Chdir(pwd)
	os.Chdir(filepath.Join(t.RepoRoot, "tests/testdata/", name))

	// Rely on the auto build functionality to avoid long slow rebuilds when nothing has changed
	t.RequirePorter("publish", "--reference", ref)
}

func (t Tester) ShowInstallation(namespace string, name string) (claims.Installation, error) {
	output, err := t.RunPorter("show", name, "--namespace", namespace, "--output=json")
	if err != nil {
		return claims.Installation{}, err
	}

	var installation claims.Installation
	require.NoError(t.T, json.Unmarshal([]byte(output), &installation))
	return installation, nil
}

func (t Tester) RequireInstallationExists(namespace string, name string) claims.Installation {
	installation, err := t.ShowInstallation(namespace, name)
	require.NoError(t.T, err)
	require.Equal(t.T, name, installation.Name, "incorrect installation name")
	require.Equal(t.T, namespace, installation.Namespace, "incorrect installation namespace")
	return installation
}

func (t Tester) RequireInstallationNotFound(namespace string, name string) {
	_, err := t.ShowInstallation(namespace, name)
	t.RequireNotFoundReturned(err)
}

func (t Tester) RequireNotFoundReturned(err error) {
	require.Error(t.T, err)
	require.Contains(t.T, err.Error(), "not found")
}

func (t Tester) ListInstallations(allNamespaces bool, namespace string, name string, labels []string) ([]claims.Installation, error) {
	args := []string{
		"list",
		"--output=json",
		"--name", name,
	}
	if allNamespaces {
		args = append(args, "--all-namespaces")
	} else {
		args = append(args, "--namespace", namespace)
	}
	for _, l := range labels {
		args = append(args, "--label", l)
	}

	output, err := t.RunPorter(args...)
	if err != nil {
		return nil, err
	}

	var installations []claims.Installation
	require.NoError(t.T, json.Unmarshal([]byte(output), &installations))
	return installations, nil
}

func (t Tester) RequireInstallationInList(namespace, name string, list []claims.Installation) claims.Installation {
	for _, i := range list {
		if i.Namespace == namespace && i.Name == name {
			return i
		}
	}

	t.T.Fatalf("expected %s/%s to be in the list of installations", namespace, name)
	return claims.Installation{}
}

// EditYaml applies a set of yq transformations to a file.
func (t Test) EditYaml(path string, transformations ...func(yq *yaml.Editor) error) {
	yq := yaml.NewEditor(t.TestContext.Context)

	require.NoError(t.T, yq.ReadFile(path))
	for _, transform := range transformations {
		require.NoError(t.T, transform(yq))
	}
	require.NoError(t.T, yq.WriteFile(path))
}
