package cmd

import (
	"btptfexporter/tfutils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
)

func exportSubaccount(subaccountID string, configDir string) error {

	dataBlock, err := readSubaccountDataSource(subaccountID)
	if err != nil {
		fmt.Println("Error getting data source:", err)
		return err
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return err
	}
	dataBlockFile := filepath.Join(TmpFolder, "main.tf")
	err = tfutils.CreateFileWithContent(dataBlockFile, dataBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return err
	}

	jsonBytes, err := getSubaccountTfStateData(TmpFolder)
	if err != nil {
		log.Fatalf("error json.Marshal: %s", err)
		return err
	}

	var data map[string]interface{}
	subaccountName := string(jsonBytes)
	err = json.Unmarshal([]byte(subaccountName), &data)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	importBlock, err := getSubaccountImportBlock(data, subaccountID)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	importFileName := "subaccount_import.tf"
	importFileName = filepath.Join(currentDir, configDir, importFileName)

	err = tfutils.CreateFileWithContent(importFileName, importBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return err
	}

	log.Println("btp subaccount has been exported. Please check " + configDir + " folder")
	return nil
}

func readSubaccountDataSource(subaccountId string) (string, error) {
	choice := "btp_subaccount"
	dsDoc, err := tfutils.GetDocsForResource("SAP", "btp", "btp", "data-sources", choice, "v1.3.0", "github.com")

	if err != nil {
		log.Fatalf("read doc failed!")
		return "", err
	}
	dataBlock := strings.Replace(dsDoc.Import, "The ID of the subaccount", subaccountId, -1)
	return dataBlock, nil

}

func getSubaccountTfStateData(configDir string) ([]byte, error) {
	execPath, err := exec.LookPath("terraform")
	if err != nil {
		log.Fatalf("error finding Terraform: %s", err)
		return nil, err
	}
	// create a new Terraform instance
	tf, err := tfexec.NewTerraform(configDir, execPath)
	if err != nil {
		log.Fatalf("error running NewTerraform: %s", err)
		return nil, err
	}

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		log.Fatalf("error running Init: %s", err)
		return nil, err
	}
	err = tf.Apply(context.Background())
	if err != nil {
		log.Fatalf("error running Apply: %s", err)
		return nil, err
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		log.Fatalf("error running Show: %s", err)
		return nil, err
	}

	jsonBytes, err := json.Marshal(state.Values.RootModule.Resources[0].AttributeValues)
	if err != nil {
		log.Fatalf("error json.Marshal: %s", err)
		return nil, err
	}

	return jsonBytes, nil

}

func getSubaccountImportBlock(data map[string]interface{}, subaccountId string) (string, error) {
	choice := "btp_subaccount"
	resource_doc, err := tfutils.GetDocsForResource("SAP", "btp", "btp", "resources", choice, "v1.3.0", "github.com")
	if err != nil {
		log.Fatalf("read doc failed!")
		return "", err
	}

	var importBlock string
	saName := fmt.Sprintf("%v", data["name"])
	template := strings.Replace(resource_doc.Import, "<resource_name>", saName, -1)
	template = strings.Replace(template, "<subaccount_id>", subaccountId, -1)
	importBlock += template + "\n"

	return importBlock, nil
}
