package cmd

import (
	"btptfexport/tfutils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Resource struct {
	Name   string
	Values []string
}

type ResourcesArr struct {
	Btp_resources []Resource
}

func exportFromFile(subaccount string, jsonfile string, resourceFile string, configDir string) {
	jsonFile, err := os.Open(jsonfile)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	var resources ResourcesArr

	err = json.Unmarshal(byteValue, &resources)

	if err != nil {
		log.Fatalf("error in unmarshall: %v", err)
		return
	}

	var resNames []string

	for i := 0; i < len(resources.Btp_resources); i++ {
		resNames = append(resNames, resources.Btp_resources[i].Name)
	}
	if len(resNames) == 0 {
		fmt.Println("No resource needs to be export")
		return
	}

	setupConfigDir(configDir)

	for _, resName := range resNames {
		var value []string
		for _, temp := range resources.Btp_resources {
			if temp.Name == resName {
				value = temp.Values
			}
		}
		if len(value) != 0 {
			generateConfigForResource(resName, value, subaccount, configDir)
		}
	}

	generateConfig(resourceFile, configDir)
}

func generateConfigForResource(resource string, values []string, subaccount string, configDir string) {
	//fmt.Println(resource)
	//fmt.Println(values)
	if resource == "environment-instances" {
		getEnvInstanceConfig(values, subaccount, configDir)
	}
	if resource == "subaccount" {
		exportSubaccount(subaccount, configDir, values)
	}
	if resource == "entitlements" {
		getEntitlementConfig(values, subaccount, configDir)
	}
	if resource == "subscriptions" {
		getSubscriptionConfig(values, subaccount, configDir)
	}
	if resource == "trust-configurations" {
		getTrustConfig(values, subaccount, configDir)
	}
}

func getTrustConfig(values []string, subaccountID string, configDir string) {
	dataBlock, err := readSubaccountTrustConfigurationsDataSource(subaccountID)
	if err != nil {
		fmt.Println("error getting data source:", err)
		return
	}

	dataBlockFile := filepath.Join(TmpFolder, "main.tf")
	err = tfutils.CreateFileWithContent(dataBlockFile, dataBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return
	}

	jsonBytes, err := GetTfStateData(TmpFolder, SubaccountTrustConfigurationType)
	if err != nil {
		log.Fatalf("error json.Marshal: %s", err)
		return
	}

	jsonString := string(jsonBytes)
	var data map[string]interface{}
	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	importBlock, err := getImportBlock4(data, subaccountID, values)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(0)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("error getting current directory:", err)
		return
	}

	if len(importBlock) == 0 {
		log.Println("No trust configuration found for the given subaccount")
		return
	}

	importFileName := "subaccount_trust_configurations_import.tf"
	importFileName = filepath.Join(currentDir, configDir, importFileName)

	err = tfutils.CreateFileWithContent(importFileName, importBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return
	}

	log.Println("subaccount trust configuration has been exported. Please check " + configDir + " folder")
}

func getImportBlock4(data map[string]interface{}, subaccountId string, values []string) (string, error) {
	choice := "btp_subaccount_trust_configuration"
	resource_doc, err := tfutils.GetDocsForResource("SAP", "btp", "btp", "resources", choice, BtpProviderVersion, "github.com")
	if err != nil {
		log.Fatalf("read doc failed!")
		return "", err
	}

	var importBlock string
	var subaccountAllTrusts []string
	trusts := data["values"].([]interface{})

	for x, value := range trusts {

		trust := value.(map[string]interface{})
		subaccountAllTrusts = append(subaccountAllTrusts, fmt.Sprintf("%v", trust["origin"]))
		if slices.Contains(values, fmt.Sprintf("%v", trust["origin"])) {
			template := strings.Replace(resource_doc.Import, "<resource_name>", "trust"+fmt.Sprint(x), -1)
			template = strings.Replace(template, "<subaccount_id>", subaccountId, -1)
			template = strings.Replace(template, "<origin>", fmt.Sprintf("%v", trust["origin"]), -1)
			importBlock += template + "\n"
		}
	}

	missingTrust, subset := isSubset(subaccountAllTrusts, values)

	if !subset {
		return "", fmt.Errorf("trust configuration %s not found in the subaccount. Please adjust it in the provided file", missingTrust)

	}

	return importBlock, nil
}

func getSubscriptionConfig(values []string, subaccountID string, configDir string) {
	dataBlock, err := readSubaccountSubscriptionDataSource(subaccountID)
	if err != nil {
		fmt.Println("error getting data source:", err)
		return
	}

	dataBlockFile := filepath.Join(TmpFolder, "main.tf")
	err = tfutils.CreateFileWithContent(dataBlockFile, dataBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return
	}

	jsonBytes, err := GetTfStateData(TmpFolder, SubaccountSubscriptionType)
	if err != nil {
		log.Fatalf("error json.Marshal: %s", err)
		return
	}

	jsonString := string(jsonBytes)
	var data map[string]interface{}
	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	importBlock, err := getImportBlock3(data, subaccountID, values)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(0)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("error getting current directory:", err)
		return
	}

	if len(importBlock) == 0 {
		log.Println("No subscription found for the given subaccount")
		return
	}

	importFileName := "subaccount_subscriptions_import.tf"
	importFileName = filepath.Join(currentDir, configDir, importFileName)

	err = tfutils.CreateFileWithContent(importFileName, importBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return
	}

	log.Println("subaccount subscriptions has been exported. Please check " + configDir + " folder")

}

func getImportBlock3(data map[string]interface{}, subaccountId string, values []string) (string, error) {
	choice := "btp_subaccount_subscription"
	resource_doc, err := tfutils.GetDocsForResource("SAP", "btp", "btp", "resources", choice, BtpProviderVersion, "github.com")
	if err != nil {
		log.Fatalf("read doc failed!")
		return "", err
	}

	var importBlock string
	var subaccountAllSubscriptions []string
	subscriptions := data["values"].([]interface{})

	for _, value := range subscriptions {
		subscription := value.(map[string]interface{})
		subaccountAllSubscriptions = append(subaccountAllSubscriptions, fmt.Sprintf("%v", subscription["app_name"])+"_"+fmt.Sprintf("%v", subscription["plan_name"]))
		if slices.Contains(values, fmt.Sprintf("%v", subscription["app_name"])+"_"+fmt.Sprintf("%v", subscription["plan_name"])) {
			template := strings.Replace(resource_doc.Import, "<resource_name>", strings.Replace(fmt.Sprintf("%v", subscription["app_name"]), "-", "_", -1), -1)
			template = strings.Replace(template, "<subaccount_id>", subaccountId, -1)
			template = strings.Replace(template, "<app_name>", fmt.Sprintf("%v", subscription["app_name"]), -1)
			template = strings.Replace(template, "<plan_name>", fmt.Sprintf("%v", subscription["plan_name"]), -1)
			importBlock += template + "\n"
		}
	}

	missingSubscription, subset := isSubset(subaccountAllSubscriptions, values)

	if !subset {
		return "", fmt.Errorf("subscription %s not found in the subaccount. Please adjust it in the provided file", missingSubscription)

	}

	return importBlock, nil
}

func getEntitlementConfig(values []string, subaccountID string, configDir string) {
	dataBlock, err := readSubaccountEntilementsDataSource(subaccountID)
	if err != nil {
		fmt.Println("error getting data source:", err)
		return
	}

	dataBlockFile := filepath.Join(TmpFolder, "main.tf")
	err = tfutils.CreateFileWithContent(dataBlockFile, dataBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return
	}

	jsonBytes, err := GetTfStateData(TmpFolder, SubaccountEntitlementType)
	if err != nil {
		log.Fatalf("error json.Marshal: %s", err)
		return
	}

	jsonString := string(jsonBytes)
	var data map[string]interface{}
	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	importBlock, err := getImportBlock2(data, subaccountID, values)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(0)
	}

	if len(importBlock) == 0 {
		log.Println("No Entitlement found for the given subaccount")
		return
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("error getting current directory:", err)
		return
	}

	importFileName := "subaccount_entitlements_import.tf"
	importFileName = filepath.Join(currentDir, configDir, importFileName)

	err = tfutils.CreateFileWithContent(importFileName, importBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return
	}

	log.Println("btp subaccount entitlements has been exported. Please check " + configDir + " folder")

}

func getImportBlock2(data map[string]interface{}, subaccountId string, values []string) (string, error) {

	choice := "btp_subaccount_entitlement"
	resource_doc, err := tfutils.GetDocsForResource("SAP", "btp", "btp", "resources", choice, BtpProviderVersion, "github.com")
	if err != nil {
		log.Fatalf("read doc failed!")
		return "", err
	}

	var importBlock string
	var subaccountAllEntitlements []string
	for key, value := range data {

		subaccountAllEntitlements = append(subaccountAllEntitlements, strings.Replace(key, ":", "_", -1))
		if slices.Contains(values, strings.Replace(key, ":", "_", -1)) {
			template := strings.Replace(resource_doc.Import, "<resource_name>", strings.Replace(key, ":", "_", -1), -1)
			template = strings.Replace(template, "<subaccount_id>", subaccountId, -1)
			if subMap, ok := value.(map[string]interface{}); ok {
				for subKey, subValue := range subMap {
					template = strings.Replace(template, "<"+subKey+">", fmt.Sprintf("%v", subValue), -1)
				}
			}
			importBlock += template + "\n"
		}
	}

	missingEntitlement, subset := isSubset(subaccountAllEntitlements, values)

	if !subset {
		return "", fmt.Errorf("entitlement %s not found in the subaccount. Please adjust it in the provided file", missingEntitlement)

	}

	return importBlock, nil

}

func getEnvInstanceConfig(values []string, subaccountID string, configDir string) {

	dataBlock, err := readDataSource(subaccountID)
	if err != nil {
		fmt.Println("error getting data source:", err)
		return
	}

	dataBlockFile := filepath.Join(TmpFolder, "main.tf")
	err = tfutils.CreateFileWithContent(dataBlockFile, dataBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return
	}

	jsonBytes, err := GetTfStateData(TmpFolder, EnvironmentInstanceType)
	if err != nil {
		log.Fatalf("error json.Marshal: %s", err)
		return
	}

	jsonString := string(jsonBytes)
	var data map[string]interface{}
	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	importBlock, err := getImportBlock1(data, subaccountID, values)

	if err != nil {
		fmt.Println("error:", err)
		os.Exit(0)
	}

	if len(importBlock) == 0 {
		log.Println("No environment instance found for the given subaccount")
		return
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("error getting current directory:", err)
		return
	}
	importFileName := "btp_environment_instances_import.tf"
	importFileName = filepath.Join(currentDir, configDir, importFileName)

	err = tfutils.CreateFileWithContent(importFileName, importBlock)
	if err != nil {
		log.Fatalf("create file %s failed!", dataBlockFile)
		return
	}

	log.Println(" environment instances have been exported. Please check " + configDir + " folder")

}

func getImportBlock1(data map[string]interface{}, subaccountId string, values []string) (string, error) {
	choice := "btp_subaccount_environment_instance"
	resource_doc, err := tfutils.GetDocsForResource("SAP", "btp", "btp", "resources", choice, BtpProviderVersion, "github.com")
	if err != nil {
		log.Fatalf("read doc failed!")
		return "", err
	}

	var importBlock string
	var subaccountAllEnvInstances []string
	environmentInstances := data["values"].([]interface{})

	for _, value := range environmentInstances {

		environmentInstance := value.(map[string]interface{})
		subaccountAllEnvInstances = append(subaccountAllEnvInstances, fmt.Sprintf("%v", environmentInstance["environment_type"]))
		if slices.Contains(values, fmt.Sprintf("%v", environmentInstance["environment_type"])) {
			template := strings.Replace(resource_doc.Import, "<resource_name>", fmt.Sprintf("%v", environmentInstance["environment_type"]), -1)
			template = strings.Replace(template, "<subaccount_id>", subaccountId, -1)
			template = strings.Replace(template, "<environment_instance_id>", fmt.Sprintf("%v", environmentInstance["id"]), -1)
			importBlock += template + "\n"
		}
	}

	missingEnvInstance, subset := isSubset(subaccountAllEnvInstances, values)

	if !subset {
		return "", fmt.Errorf("environment instance %s not found in the subaccount. Please adjust it in the provided file", missingEnvInstance)

	}

	return importBlock, nil
}

func isSubset(superSet []string, subset []string) (string, bool) {
	for _, value := range subset {
		if !slices.Contains(superSet, value) {
			return value, false
		}
	}
	return "", true
}
