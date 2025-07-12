// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/awslabs/diagram-as-code/internal/ctl"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {

	var outputFile string
	var verbose bool
	var cfnTemplate bool
	var generateDacFile bool
	var overrideDefFile string
	var isGoTemplate bool
	var serverMode bool
	var serverPort int

	var rootCmd = &cobra.Command{
		Use:     "awsdac <input filename>",
		Version: version,
		Short:   "Diagram-as-code for AWS architecture.",
		Long:    "This command line interface (CLI) tool enables drawing infrastructure diagrams for Amazon Web Services through YAML code.",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {

			// サーバーモードの場合は引数チェックをスキップ
			if serverMode {
				return nil
			}

			if len(args) == 0 {
				error_message := "awsdac: This tool requires an input file to run. Please provide a file path.\n"
				fmt.Println(error_message)
				cmd.Help()

				os.Exit(1)
			}

			inputFile := args[0]
			if !ctl.IsURL(inputFile) {

				if _, err := os.Stat(inputFile); os.IsNotExist(err) {
					fmt.Printf("awsdac: Input file '%s' does not exist.\n", inputFile)
					os.Exit(1)
				}
			}

			return nil

		},
		Run: func(cmd *cobra.Command, args []string) {

			if verbose {
				log.SetLevel(log.InfoLevel)
			} else {
				log.SetLevel(log.WarnLevel)
			}

			// サーバーモードの場合はHTTPサーバーを起動
			if serverMode {
				startHTTPServer(serverPort, &outputFile, cfnTemplate, generateDacFile, overrideDefFile, isGoTemplate)
				return
			}

			inputFile := args[0]

			if cfnTemplate {
				opts := ctl.CreateOptions{
					OverrideDefFile: overrideDefFile,
				}
				ctl.CreateDiagramFromCFnTemplate(inputFile, &outputFile, generateDacFile, &opts)
			} else {
				opts := ctl.CreateOptions{
					IsGoTemplate:    isGoTemplate,
					OverrideDefFile: overrideDefFile,
				}
				ctl.CreateDiagramFromDacFile(inputFile, &outputFile, &opts)
			}

		},
	}

	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "output.png", "Output file name")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&cfnTemplate, "cfn-template", "c", false, "[beta] Create diagram from CloudFormation template")
	rootCmd.PersistentFlags().BoolVarP(&generateDacFile, "dac-file", "d", false, "[beta] Generate YAML file in dac (diagram-as-code) format from CloudFormation template")
	rootCmd.PersistentFlags().StringVarP(&overrideDefFile, "override-def-file", "", "", "For testing purpose, override DefinitionFiles to another url/local file")
	rootCmd.PersistentFlags().BoolVarP(&isGoTemplate, "template", "t", false, "Processes the input file as a template according to text/template.")
	rootCmd.PersistentFlags().BoolVarP(&serverMode, "server", "s", false, "Run as HTTP server")
	rootCmd.PersistentFlags().IntVarP(&serverPort, "port", "p", 8080, "Port for HTTP server (default: 8080)")

	rootCmd.Execute()
}

// HTTPサーバーを起動する関数
func startHTTPServer(port int, outputFile *string, cfnTemplate bool, generateDacFile bool, overrideDefFile string, isGoTemplate bool) {
	http.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {
		handleGenerateRequest(w, r, outputFile, cfnTemplate, generateDacFile, overrideDefFile, isGoTemplate)
	})

	fmt.Printf("Starting HTTP server on port %d...\n", port)
	fmt.Printf("Generate diagram: GET /generate\n")

	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatal("Failed to start HTTP server:", err)
	}
}

// ダイアグラム生成リクエストを処理する関数
func handleGenerateRequest(w http.ResponseWriter, r *http.Request, outputFile *string, cfnTemplate bool, generateDacFile bool, overrideDefFile string, isGoTemplate bool) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// ハードコードされたファイルを使用
	inputFile := "examples/alb-ec2.yaml"

	// ファイルの存在確認
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Input file '%s' does not exist", inputFile), http.StatusNotFound)
		return
	}

	// ダイアグラム生成処理
	if cfnTemplate {
		opts := ctl.CreateOptions{
			OverrideDefFile: overrideDefFile,
		}
		ctl.CreateDiagramFromCFnTemplate(inputFile, outputFile, generateDacFile, &opts)
	} else {
		opts := ctl.CreateOptions{
			IsGoTemplate:    isGoTemplate,
			OverrideDefFile: overrideDefFile,
		}
		ctl.CreateDiagramFromDacFile(inputFile, outputFile, &opts)
	}

	// 成功レスポンス
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}
