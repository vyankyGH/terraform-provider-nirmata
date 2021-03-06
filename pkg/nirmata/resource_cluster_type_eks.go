package nirmata

import (
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	client "github.com/nirmata/go-client/pkg/client"
)

func resourceEksClusterType() *schema.Resource {
	return &schema.Resource{
		Create: resourceEksClusterTypeCreate,
		Read:   resourceEksClusterTypeRead,
		Update: resourceEksClusterTypeUpdate,
		Delete: resourceEksClusterTypeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateName,
			},
			"version": {
				Type:     schema.TypeString,
				Required: true,
			},
			"credentials": {
				Type:     schema.TypeString,
				Required: true,
			},
			"region": {
				Type:     schema.TypeString,
				Required: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"subnet_id": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
			},
			"cluster_role_arn": {
				Type:     schema.TypeString,
				Required: true,
			},
			"security_groups": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
			},
			"log_types": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"enable_private_endpoint": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"enable_secrets_encryption": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"kms_key_arn": {
				Type:     schema.TypeString,
				Optional: true, // required if enable_secrets_encryption = true
			},
			"enable_identity_provider": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"system_metadata": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"allow_override_credentials": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"cluster_field_override": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"addons": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: addonSchema,
				},
			},
			"vault_auth": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: vaultAuthSchema,
				},
			},
			"nodepools": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: eksNodePoolSchema,
				},
			},
			"nodepool_field_override": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

var eksNodePoolSchema = map[string]*schema.Schema{
	"name": {
		Type:     schema.TypeString,
		Required: true,
	},
	"instance_type": {
		Type:     schema.TypeString,
		Required: true,
	},
	"disk_size": {
		Type:         schema.TypeInt,
		Required:     true,
		ValidateFunc: validateEKSDiskSize,
	},
	"security_groups": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Required: true,
	},
	"iam_role": {
		Type:     schema.TypeString,
		Required: true,
	},
	"ssh_key_name": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"ami_type": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"image_id": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"node_annotations": {
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	},
	"node_labels": {
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	},
}

func resourceEksClusterTypeCreate(d *schema.ResourceData, meta interface{}) error {
	apiClient := meta.(client.Client)

	credentials := d.Get("credentials").(string)
	cloudCredID, err := apiClient.QueryByName(client.ServiceClusters, "CloudCredentials", credentials)
	if err != nil {
		log.Printf("Error - %v", err)
		return err
	}

	name := d.Get("name").(string)
	version := d.Get("version").(string)
	region := d.Get("region").(string)
	securityGroups := d.Get("security_groups")
	clusterRoleArn := d.Get("cluster_role_arn").(string)
	vpcID := d.Get("vpc_id").(string)
	subnetID := d.Get("subnet_id")
	logTypes := d.Get("log_types")
	privateEndpointAccess := d.Get("enable_private_endpoint")
	enableSecretsEncryption := d.Get("enable_secrets_encryption")
	keyArn := d.Get("kms_key_arn")
	enableIdentityProvider := d.Get("enable_identity_provider")
	systemMetadata := d.Get("system_metadata")
	allowOverrideCredentials := d.Get("allow_override_credentials").(bool)
	clusterFieldOverride := d.Get("cluster_field_override")
	nodepoolFieldOverride := d.Get("nodepool_field_override")

	fieldsToOverride := map[string]interface{}{
		"cluster":  clusterFieldOverride,
		"nodePool": nodepoolFieldOverride,
	}

	var nodeobjArr = make([]interface{}, 0)
	nodepools := d.Get("nodepools").([]interface{})
	for i, node := range nodepools {
		element, ok := node.(map[string]interface{})
		if ok {
			nodePoolObj := map[string]interface{}{
				"modelIndex": "NodePoolType",
				"name":       name + "-node-pool-" + strconv.Itoa(i),
				"spec": map[string]interface{}{
					"modelIndex":      "NodePoolSpec",
					"nodeLabels":      element["node_labels"],
					"nodeAnnotations": element["node_annotations"],
					"eksConfig": map[string]interface{}{
						"instanceType":   element["instance_type"],
						"diskSize":       element["disk_size"],
						"securityGroups": element["security_groups"],
						"nodeIamRole":    element["iam_role"],
						"keyName":        element["ssh_key_name"],
						"amiType":        element["ami_type"],
						"imageId":        element["image_id"],
					},
				},
			}

			nodeobjArr = append(nodeobjArr, nodePoolObj)
		}
	}

	addons := addOnsSchemaToAddOns(d)

	clusterTypeData := map[string]interface{}{
		"name":        name,
		"description": "",
		"modelIndex":  "ClusterType",
		"spec": map[string]interface{}{
			"clusterMode":    "providerManaged",
			"modelIndex":     "ClusterSpec",
			"version":        version,
			"cloud":          "aws",
			"systemMetadata": systemMetadata,
			"addons":         addons,
			"cloudConfigSpec": map[string]interface{}{
				"modelIndex":               "CloudConfigSpec",
				"credentials":              cloudCredID.UUID(),
				"allowOverrideCredentials": allowOverrideCredentials,
				"fieldsToOverride":         fieldsToOverride,
				"eksConfig": map[string]interface{}{
					"region":                  region,
					"vpcId":                   vpcID,
					"subnetId":                subnetID,
					"clusterRoleArn":          clusterRoleArn,
					"securityGroups":          securityGroups,
					"logTypes":                logTypes,
					"privateEndpointAccess":   privateEndpointAccess,
					"enableIdentityProvider":  enableIdentityProvider,
					"enableSecretsEncryption": enableSecretsEncryption,
					"keyArn":                  keyArn,
				},
				"nodePoolTypes": nodeobjArr,
			},
		},
	}

	if _, ok := d.GetOk("vault_auth"); ok {
		vl := d.Get("vault_auth").([]interface{})
		vault := vl[0].(map[string]interface{})
		clusterTypeData["spec"].(map[string]interface{})["vault"] = vaultAuthSchemaToVaultAuthSpec(vault)
	}

	txn := make(map[string]interface{})
	var objArr = make([]interface{}, 0)
	objArr = append(objArr, clusterTypeData)
	txn["create"] = objArr
	data, err := apiClient.PostFromJSON(client.ServiceClusters, "txn", txn, nil)
	if err != nil {
		log.Printf("[ERROR] - failed to create cluster type  with data : %v", err)
		return err
	}

	obj, resultErr := extractCreateFromTxnResult(data, "ClusterType")
	if resultErr != nil {
		log.Printf("[ERROR] - %v", err)
		return resultErr
	}

	d.SetId(obj.ID().UUID())
	return nil
}

func resourceEksClusterTypeRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceEksClusterTypeUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceEksClusterTypeDelete(d *schema.ResourceData, meta interface{}) error {
	return deleteObj(d, meta, client.ServiceClusters, "ClusterType")
}
