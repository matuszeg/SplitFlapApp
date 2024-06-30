package proto

import (
	"SplitFlapApp/utils"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GenerateProtoFiles() {
	currentDirectoryPath, err := os.Getwd()
	if err != nil {
		panic("Unable to get current directory")
	}

	protoFilePath := filepath.Join(currentDirectoryPath, "thirdparty", "splitflap", "proto")
	nanopbFilePath := filepath.Join(currentDirectoryPath, "thirdparty", "nanopb")

	if _, err := os.Stat(nanopbFilePath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Nanopb submodule not found! Make sure you have inited/updated the submodule located at %s", protoFilePath))
	}

	nanopbGeneratorFilePath := filepath.Join(nanopbFilePath, "generator", "nanopb_generator.py")
	nanopbOutputPath := filepath.Join(currentDirectoryPath, "tmp")

	protoFile := filepath.Join(protoFilePath, "splitflap.proto")

	if _, err := os.Stat(nanopbOutputPath); os.IsNotExist(err) {
		err = os.Mkdir(nanopbOutputPath, 0744)
		if err != nil {
			errMsg := "Unable to create nanopbOutputPath"
			utils.Log(utils.LogLevel_Error, errMsg, err)
			panic(errMsg)
		}
	}

	err = filepath.Walk(nanopbFilePath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			input, err := os.ReadFile(path)
			if err != nil {
				log.Fatalln(err)
			}

			lines := strings.Split(string(input), "\n")

			for i, line := range lines {
				lines[i] = strings.Replace(line, "--python_out=", "--go_out=Mnanopb.proto=../../../../generated/nanopb:", -1)

			}

			output := strings.Join(lines, "\n")
			err = os.WriteFile(path, []byte(output), 0644)
			if err != nil {
				log.Fatalln(err)
			}

		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	nanopbCommand := exec.Command("python3", nanopbGeneratorFilePath, "-I", protoFilePath, "-D", nanopbOutputPath, protoFile)
	output, err := nanopbCommand.Output()
	if err != nil {
		errMsg := "Unable to run nanopb_generator.py"
		utils.Log(utils.LogLevel_Error, errMsg, err)
		panic(errMsg)
	}

	utils.Log(utils.LogLevel_Debug, string(output), nil)

	protocPath := filepath.Join(nanopbFilePath, "generator", "protoc")
	generatedFilesOutputPath := filepath.Join(currentDirectoryPath, "generated")

	goOut := fmt.Sprintf("--go_out=Mnanopb.proto=SplitFlapApp/generated/nanopb:%s", generatedFilesOutputPath)
	goOpt := fmt.Sprintf("--go_opt=Msplitflap.proto=./")

	command := exec.Command(protocPath, goOut, goOpt, "-I", protoFilePath, protoFile)
	if output, err = command.Output(); err != nil {
		errMsg := "Unable to run protoc to generate go protobuf files"
		utils.Log(utils.LogLevel_Error, errMsg, err)
		panic(errMsg)
	}
}
