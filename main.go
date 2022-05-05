package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Release struct {
	Id          string `json:"id"`
	TagName     string `json:"tag_name"`
	DownloadURL string `json:"zipball_url"`
}

type Secret struct {
	name  string
	value string
}

type Input struct {
	Name        string
	Description string
	IsSecret    bool
	Required    bool
}

// Stack init config

type StackInitConfig struct {
	Name string `yaml:"name"`
	On   struct {
		Workflow_dispatch string `yaml:"workflow_dispatch"`
	} `yaml:"on"`
	Jobs struct {
		StackInitialization struct {
			RunsOn string `yaml:"runs-on"`
			Steps  []struct {
				Run  string `yaml:"run"`
				Name string `yaml:"name"`
				Env  []struct {
					Bucket string `yaml:"BUCKET"`
					Region string `yaml:"REGION"`
				} `yaml:"env"`
				Uses string `yaml:"uses"`
				With struct {
					PersistCredentials bool   `yaml:"persist-credentials"`
					NodeVersion        string `yaml:"node-version"`
					CheckLatest        bool   `yaml:"check-latest"`
					RegistryUrl        string `yaml:"registry-url"`
					AwsAccessKeyId     string `yaml:"aws-access-key-id"`
					AwsSecretAccessKey string `yaml:"aws-secret-access-key"`
					AwsRegion          string `yaml:"aws-region"`
				} `yaml:"with"`
			}
		} `yaml:"stack-initialization"`
	} `yaml:"jobs"`
}

// YamlConfig is exported.
type YamlConfig struct {
	Version     string   `yaml:"version"`
	Uses        []string `yaml:"uses"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Branding    struct {
		Icon  string
		Color string
	} `yaml:"branding"`
	Inputs  []Input `yaml:"inputs"`
	Configs struct {
		RepoMetadata struct {
			Parameters struct {
				Description string
				Secrets     []struct {
					Name  string `yaml:"name"`
					Value string `yaml:"value"`
				} `yaml:"secrets"`
				Topics []string `yaml:"topics"`
			} `yaml:"parameters"`
		} `yaml:"repo-metadata"`
		Branches []struct {
			Name       string
			Parameters struct {
				EnforceAdmins bool
			}
		}
	} `yaml:"configs"`
}

func main() {
	fmt.Print("? Application name: ")
	var applicationName string
	fmt.Scanln(&applicationName)

	fmt.Print("? Description: ")
	var description string
	fmt.Scanln(&description)

	fmt.Println("? Repo visibility: (public/private)")
	var repoVisibility string
	fmt.Scanln(&repoVisibility)

	fmt.Println("---------------------------")
	fmt.Println("Parsing stack.yaml file...")
	fmt.Println("---------------------------")

	var fileName string = "stack.yml"

	if fileName == "" {
		fmt.Println("Please provide yaml file by using -f option")
		return
	}

	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return
	}

	var yamlConfig YamlConfig
	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}

	var stack_one = yamlConfig.Uses[0]
	var stack_two = yamlConfig.Uses[1]

	fmt.Printf("? Choose a release version for %s: \n", stack_one)
	var firstStackReleases = getReleases(stack_one)
	for firstStackReleases == nil {
		firstStackReleases = getReleases(stack_one)
	}
	for i := 0; i < len(firstStackReleases); i++ {
		fmt.Printf("%s\n", firstStackReleases[i].TagName)
	}

	var stack_one_version string
	fmt.Scanln(&stack_one_version)

	var firstStackUrl string

	for i := 0; i < len(firstStackReleases); i++ {
		if firstStackReleases[i].TagName == stack_one_version {
			firstStackUrl = firstStackReleases[i].DownloadURL
		}
	}

	var stack_one_repo_name string = strings.Split(stack_one, "/")[1] + ".zip"
	downloadReleaseByURL(firstStackUrl, stack_one_repo_name)
	unzip(stack_one_repo_name, strings.Split(stack_one, "/")[1])
	fmt.Printf("\n")
	fmt.Printf("? Choose a release version for %s: \n", stack_two)
	var secondStackReleases = getReleases(stack_two)
	for i := 0; i < len(secondStackReleases); i++ {
		fmt.Printf("%s\n", secondStackReleases[i].TagName)
	}

	for secondStackReleases == nil {
		secondStackReleases = getReleases(stack_two)
	}

	var stack_two_version string
	fmt.Scanln(&stack_two_version)

	var secondStackUrl string

	for i := 0; i < len(secondStackReleases); i++ {
		if secondStackReleases[i].TagName == stack_two_version {
			secondStackUrl = secondStackReleases[i].DownloadURL
		}
	}

	var stack_two_repo_name string = strings.Split(stack_two, "/")[1] + ".zip"
	downloadReleaseByURL(secondStackUrl, stack_two_repo_name)
	unzip(stack_two_repo_name, strings.Split(stack_two, "/")[1])
	fmt.Printf("\n")
	yamlFile, err = ioutil.ReadFile("stack.yml")
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return
	}

	var root YamlConfig
	err = yaml.Unmarshal(yamlFile, &root)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}

	fmt.Printf("\n============\n\n")
	mergeConfig(&root, "./nextjs-aws-s3-stack/3loka-nextjs-aws-s3-stack-b0420bb/.github/stacks/stack.yml", "./node-azure-stack/3loka-node-azure-stack-5d7af93/.github/stacks/stack.yml")
	input_map := make(map[string]string)
	fmt.Println("Please enter the values of the secrets required for the repo creation.")
	fmt.Printf("\n")
	for i := 0; i < len(root.Inputs); i++ {
		fmt.Printf("%s :", root.Inputs[i].Name)
		var input_var string
		fmt.Scanln(&input_var)
		input_map[root.Inputs[i].Name] = input_var
	}

	fmt.Println(input_map, "map")
	generateRepo(applicationName, "react-node")
	err = os.MkdirAll(`./generated-repo/.github/workflows/`, os.ModePerm)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}

	src_one := fmt.Sprintf("./nextjs-aws-s3-stack/3loka-nextjs-aws-s3-stack-b0420bb/.github/workflows/stack-init.yaml")
	dest_one := fmt.Sprintf("./generated-repo/.github/workflows/stack-init-1.yaml")
	fmt.Println(src_one, "->", dest_one)
	os.Rename(src_one, dest_one)

	src_two := fmt.Sprintf("./node-azure-stack/3loka-node-azure-stack-5d7af93/.github/workflows/stack-init.yaml")
	dest_two := fmt.Sprintf("./generated-repo/.github/workflows/stack-init-2.yaml")
	fmt.Println(src_two, "->", dest_two)
	os.Rename(src_two, dest_two)

	CopyDir("./generated-repo", "./react-node")

	os.Chdir("./react-node/")
	os.MkdirAll(`./react`, os.ModePerm)
	os.MkdirAll(`./node`, os.ModePerm)

	cmd := exec.Command("git", "add", ".")

	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	cmd = exec.Command("git", "commit", "-m", "create new repository")

	stdout, err = cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(stdout))

	cmd = exec.Command("git", "push", "origin", "main")

	stdout, err = cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(stdout))

	// for key, element := range input_map {
	// 	encrypted, _ := githubsecret.Encrypt("jMIfZj9yHwyuTeNjdVFhAFCXnyZb6PP2eOCGu8r5NAU=", element)
	// 	addRepoSecret(key, encrypted)
	// }

	// encrypted, _ := githubsecret.Encrypt("jMIfZj9yHwyuTeNjdVFhAFCXnyZb6PP2eOCGu8r5NAU=", "AKIAXV27PUM6LI4LQB54")
	// fmt.Println(encrypted)

	time.Sleep(5 * time.Second)
	fmt.Println("Triggering first workflow after 5 seconds of wait")
	triggerWorkflow("stack-init-1.yaml")
	time.Sleep(3 * time.Second)
	fmt.Println("Triggering second workflow after 3 seconds of wait")
	triggerWorkflowWithInputs("stack-init-2.yaml")
}

func addRepoSecret(secretName string, secretValue string) {
	url := "https://api.github.com/repos/gauravdas019/react-node/actions/secrets/" + secretName
	method := "PUT"

	payload := strings.NewReader(`{
	  "encrypted_value": {secretValue},
	  "key_id": "568250167242549743"
  }`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Authorization", "Bearer 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")
	req.Header.Add("Content-Type", "text/html")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

func CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}

	}

	return
}

func CopyDir(source string, dest string) (err error) {

	// get properties of source dir
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// create dest dir

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {

		sourcefilepointer := source + "/" + obj.Name()

		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			// create sub-directories - recursively
			err = CopyDir(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			// perform copy
			err = CopyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}

	}
	return
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func getReleases(stack string) []Release {
	apiURL := "https://api.github.com/repos/" + stack + "/releases"
	method := "GET"

	// proxyURL, err := url.Parse(os.Getenv("HTTP_PROXY"))
	client := &http.Client{}
	// client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	req, err := http.NewRequest(method, apiURL, nil)

	if err != nil {
		fmt.Println(err)
		return []Release{}
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return []Release{}
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return []Release{}
	}
	var releases []Release
	json.Unmarshal(body, &releases)

	// fmt.Printf("%+v", releases)
	return releases
}

func downloadReleaseByURL(apiURL string, outputFileName string) {
	method := "GET"

	// proxyURL, err := url.Parse(os.Getenv("HTTP_PROXY"))
	client := &http.Client{}
	// client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	req, err := http.NewRequest(method, apiURL, nil)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	out, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer out.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func mergeConfig(root *YamlConfig, paths ...string) {
	for _, path := range paths {
		fileContents, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		var config YamlConfig
		err = yaml.Unmarshal(fileContents, &config)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		root.Inputs = append(root.Inputs, config.Inputs...)
	}
}

func generateRepo(applicationName string, repoToCreate string) {
	fmt.Println("Create repository usng GH API")
	apiURL := "https://api.github.com/user/repos"
	method := "POST"

	payload := strings.NewReader(`{
		  "name": "react-node"
	  }`)

	// proxyURL, err := url.Parse(os.Getenv("HTTP_PROXY"))
	client := &http.Client{}
	// client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	req, err := http.NewRequest(method, apiURL, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Authorization", "Bearer PAT_TOKEN_HERE")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(body))
	repoToClone := "https://github.com/GITHUB_USERNAME/" + repoToCreate + ".git"
	cloneCmd := exec.Command("git", "clone", repoToClone, applicationName)

	stdout, err := cloneCmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(stdout))
	fmt.Println("Repo created successfully")
}

func triggerWorkflow(fileName string) {
	url := "https://api.github.com/repos/gauravdas019/react-node/actions/workflows/" + fileName + "/dispatches"
	method := "POST"

	payload := strings.NewReader(`{
    "ref": "main"
}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Authorization", "Bearer GITHUB_PAT_TOKEN_HERE")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

func triggerWorkflowWithInputs(fileName string) {
	url := "https://api.github.com/repos/gauravdas019/react-node/actions/workflows/" + fileName + "/dispatches"
	method := "POST"

	payload := strings.NewReader(`{
    "ref": "main",
	"inputs": {
		"AZURE_APP_SERVICE_NAME" : "app-svc"
	}
}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Authorization", "Bearer GITHUB_PAT_TOKEN_HERE")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

func copyFiles(src string, dest string) {

	bytesRead, err := ioutil.ReadFile(src)

	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(dest, bytesRead, 0644)

	if err != nil {
		log.Fatal(err)
	}
}

func parseStackInitFiles(src string) {
	yamlFile, err := ioutil.ReadFile("./react-node/.github/workflows/stack-init-1.yaml")
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return
	}

	var yamlConfig StackInitConfig
	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}

	fmt.Printf("%+v", yamlConfig.Jobs.StackInitialization.Steps)

	steps := yamlConfig

	data, err := yaml.Marshal(&steps)

	if err != nil {

		log.Fatal(err)
	}

	err2 := ioutil.WriteFile("stack-new-init.yaml", data, 0644)

	if err2 != nil {

		log.Fatal(err2)
	}

	fmt.Println("data written")
}
