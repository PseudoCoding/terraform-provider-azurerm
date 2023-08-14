// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package compute

import (
	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-03-01/virtualmachines"
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2023-03-01/virtualmachinescalesets"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/compute/validate"
	keyVaultValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
	"github.com/tombuildsstuff/kermit/sdk/compute/2023-03-01/compute"
)

func additionalUnattendContentSchema() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		// whilst the SDK supports updating, the API doesn't:
		//   Code="PropertyChangeNotAllowed"
		//   Message="Changing property 'windowsConfiguration.additionalUnattendContent' is not allowed."
		//   Target="windowsConfiguration.additionalUnattendContent
		ForceNew: true,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"content": {
					Type:      pluginsdk.TypeString,
					Required:  true,
					ForceNew:  true,
					Sensitive: true,
				},
				"setting": {
					Type:     pluginsdk.TypeString,
					Required: true,
					ForceNew: true,
					ValidateFunc: validation.StringInSlice([]string{
						string(compute.SettingNamesAutoLogon),
						string(compute.SettingNamesFirstLogonCommands),
					}, false),
				},
			},
		},
	}
}

func expandAdditionalUnattendContent(input []interface{}) *[]virtualmachines.AdditionalUnattendContent {
	if len(input) == 0 {
		return nil
	}
	output := make([]virtualmachines.AdditionalUnattendContent, 0)

	for _, v := range input {
		raw := v.(map[string]interface{})

		output = append(output, virtualmachines.AdditionalUnattendContent{
			SettingName: pointer.To(virtualmachines.SettingNames(raw["setting"].(string))),
			Content:     utils.String(raw["content"].(string)),

			// no other possible values
			PassName:      pointer.To(virtualmachines.PassNamesOobeSystem),
			ComponentName: pointer.To(virtualmachines.ComponentNamesMicrosoftNegativeWindowsNegativeShellNegativeSetup),
		})
	}

	return &output
}

func flattenAdditionalUnattendContent(input *[]virtualmachines.AdditionalUnattendContent, d *pluginsdk.ResourceData) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	existing := make([]interface{}, 0)
	if v, ok := d.GetOk("additional_unattend_content"); ok {
		existing = v.([]interface{})
	}

	output := make([]interface{}, 0)
	for i, v := range *input {
		// content isn't returned from the API as it's sensitive so we need to look it up
		content := ""
		if len(existing) > i {
			existingVal := existing[i]
			existingRaw, ok := existingVal.(map[string]interface{})
			if ok {
				contentRaw, ok := existingRaw["content"]
				if ok {
					content = contentRaw.(string)
				}
			}
		}

		output = append(output, map[string]interface{}{
			"content": content,
			"setting": string(pointer.From(v.SettingName)),
		})
	}

	return output
}

func expandVMSSAdditionalUnattendContent(input []interface{}) *[]virtualmachinescalesets.AdditionalUnattendContent {
	if len(input) == 0 {
		return nil
	}
	output := make([]virtualmachinescalesets.AdditionalUnattendContent, 0)

	for _, v := range input {
		raw := v.(map[string]interface{})

		output = append(output, virtualmachinescalesets.AdditionalUnattendContent{
			SettingName: pointer.To(virtualmachinescalesets.SettingNames(raw["setting"].(string))),
			Content:     utils.String(raw["content"].(string)),

			// no other possible values
			PassName:      pointer.To(virtualmachinescalesets.PassNamesOobeSystem),
			ComponentName: pointer.To(virtualmachinescalesets.ComponentNamesMicrosoftNegativeWindowsNegativeShellNegativeSetup),
		})
	}

	return &output
}

func flattenVMSSAdditionalUnattendContent(input *[]virtualmachinescalesets.AdditionalUnattendContent, d *pluginsdk.ResourceData) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	existing := make([]interface{}, 0)
	if v, ok := d.GetOk("additional_unattend_content"); ok {
		existing = v.([]interface{})
	}

	output := make([]interface{}, 0)
	for i, v := range *input {
		// content isn't returned from the API as it's sensitive so we need to look it up
		content := ""
		if len(existing) > i {
			existingVal := existing[i]
			existingRaw, ok := existingVal.(map[string]interface{})
			if ok {
				contentRaw, ok := existingRaw["content"]
				if ok {
					content = contentRaw.(string)
				}
			}
		}

		output = append(output, map[string]interface{}{
			"content": content,
			"setting": string(pointer.From(v.SettingName)),
		})
	}

	return output
}

func bootDiagnosticsSchema() *pluginsdk.Schema {
	// lintignore:XS003
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				// TODO: should this be `storage_account_endpoint`?
				"storage_account_uri": {
					Type:     pluginsdk.TypeString,
					Optional: true,
					// TODO: validation
				},
			},
		},
	}
}

func expandBootDiagnostics(input []interface{}) *virtualmachines.DiagnosticsProfile {
	if len(input) == 0 {
		return &virtualmachines.DiagnosticsProfile{
			BootDiagnostics: &virtualmachines.BootDiagnostics{
				Enabled:    utils.Bool(false),
				StorageUri: utils.String(""),
			},
		}
	}

	// this serves the managed boot diagnostics, in this case we only have this empty block without `storage_account_uri` set
	if input[0] == nil {
		return &virtualmachines.DiagnosticsProfile{
			BootDiagnostics: &virtualmachines.BootDiagnostics{
				Enabled:    utils.Bool(true),
				StorageUri: utils.String(""),
			},
		}
	}

	raw := input[0].(map[string]interface{})

	storageAccountURI := raw["storage_account_uri"].(string)

	return &virtualmachines.DiagnosticsProfile{
		BootDiagnostics: &virtualmachines.BootDiagnostics{
			Enabled:    utils.Bool(true),
			StorageUri: utils.String(storageAccountURI),
		},
	}
}

func flattenBootDiagnostics(input *virtualmachines.DiagnosticsProfile) []interface{} {
	if input == nil || input.BootDiagnostics == nil || input.BootDiagnostics.Enabled == nil || !*input.BootDiagnostics.Enabled {
		return []interface{}{}
	}

	storageAccountUri := ""
	if input.BootDiagnostics.StorageUri != nil {
		storageAccountUri = *input.BootDiagnostics.StorageUri
	}

	return []interface{}{
		map[string]interface{}{
			"storage_account_uri": storageAccountUri,
		},
	}
}

func expandVMSSBootDiagnostics(input []interface{}) *virtualmachinescalesets.DiagnosticsProfile {
	if len(input) == 0 {
		return &virtualmachinescalesets.DiagnosticsProfile{
			BootDiagnostics: &virtualmachinescalesets.BootDiagnostics{
				Enabled:    utils.Bool(false),
				StorageUri: utils.String(""),
			},
		}
	}

	// this serves the managed boot diagnostics, in this case we only have this empty block without `storage_account_uri` set
	if input[0] == nil {
		return &virtualmachinescalesets.DiagnosticsProfile{
			BootDiagnostics: &virtualmachinescalesets.BootDiagnostics{
				Enabled:    utils.Bool(true),
				StorageUri: utils.String(""),
			},
		}
	}

	raw := input[0].(map[string]interface{})

	storageAccountURI := raw["storage_account_uri"].(string)

	return &virtualmachinescalesets.DiagnosticsProfile{
		BootDiagnostics: &virtualmachinescalesets.BootDiagnostics{
			Enabled:    utils.Bool(true),
			StorageUri: utils.String(storageAccountURI),
		},
	}
}

func flattenVMSSBootDiagnostics(input *virtualmachinescalesets.DiagnosticsProfile) []interface{} {
	if input == nil || input.BootDiagnostics == nil || input.BootDiagnostics.Enabled == nil || !*input.BootDiagnostics.Enabled {
		return []interface{}{}
	}

	storageAccountUri := ""
	if input.BootDiagnostics.StorageUri != nil {
		storageAccountUri = *input.BootDiagnostics.StorageUri
	}

	return []interface{}{
		map[string]interface{}{
			"storage_account_uri": storageAccountUri,
		},
	}
}

func linuxSecretSchema() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				// whilst this isn't present in the nested object it's required when this is specified
				"key_vault_id": commonschema.ResourceIDReferenceRequired(commonids.KeyVaultId{}),

				// whilst we /could/ flatten this to `certificate_urls` we're intentionally not to keep this
				// closer to the Windows VMSS resource, which will also take a `store` param
				"certificate": {
					Type:     pluginsdk.TypeSet,
					Required: true,
					MinItems: 1,
					Elem: &pluginsdk.Resource{
						Schema: map[string]*pluginsdk.Schema{
							"url": {
								Type:         pluginsdk.TypeString,
								Required:     true,
								ValidateFunc: keyVaultValidate.NestedItemId,
							},
						},
					},
				},
			},
		},
	}
}

func expandLinuxSecrets(input []interface{}) *[]virtualmachines.VaultSecretGroup {
	output := make([]virtualmachines.VaultSecretGroup, 0)

	for _, raw := range input {
		v := raw.(map[string]interface{})

		keyVaultId := v["key_vault_id"].(string)
		certificatesRaw := v["certificate"].(*pluginsdk.Set).List()
		certificates := make([]virtualmachines.VaultCertificate, 0)
		for _, certificateRaw := range certificatesRaw {
			certificateV := certificateRaw.(map[string]interface{})

			url := certificateV["url"].(string)
			certificates = append(certificates, virtualmachines.VaultCertificate{
				CertificateUrl: utils.String(url),
			})
		}

		output = append(output, virtualmachines.VaultSecretGroup{
			SourceVault: &virtualmachines.SubResource{
				Id: utils.String(keyVaultId),
			},
			VaultCertificates: &certificates,
		})
	}

	return &output
}

func flattenLinuxSecrets(input *[]virtualmachines.VaultSecretGroup) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	output := make([]interface{}, 0)

	for _, v := range *input {
		keyVaultId := ""
		if v.SourceVault != nil && v.SourceVault.Id != nil {
			keyVaultId = *v.SourceVault.Id
		}

		certificates := make([]interface{}, 0)

		if v.VaultCertificates != nil {
			for _, c := range *v.VaultCertificates {
				if c.CertificateUrl == nil {
					continue
				}

				certificates = append(certificates, map[string]interface{}{
					"url": *c.CertificateUrl,
				})
			}
		}

		output = append(output, map[string]interface{}{
			"key_vault_id": keyVaultId,
			"certificate":  certificates,
		})
	}

	return output
}

func expandVMSSLinuxSecrets(input []interface{}) *[]virtualmachinescalesets.VaultSecretGroup {
	output := make([]virtualmachinescalesets.VaultSecretGroup, 0)

	for _, raw := range input {
		v := raw.(map[string]interface{})

		keyVaultId := v["key_vault_id"].(string)
		certificatesRaw := v["certificate"].(*pluginsdk.Set).List()
		certificates := make([]virtualmachinescalesets.VaultCertificate, 0)
		for _, certificateRaw := range certificatesRaw {
			certificateV := certificateRaw.(map[string]interface{})

			url := certificateV["url"].(string)
			certificates = append(certificates, virtualmachinescalesets.VaultCertificate{
				CertificateUrl: utils.String(url),
			})
		}

		output = append(output, virtualmachinescalesets.VaultSecretGroup{
			SourceVault: &virtualmachinescalesets.SubResource{
				Id: utils.String(keyVaultId),
			},
			VaultCertificates: &certificates,
		})
	}

	return &output
}

func flattenVMSSLinuxSecrets(input *[]virtualmachinescalesets.VaultSecretGroup) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	output := make([]interface{}, 0)

	for _, v := range *input {
		keyVaultId := ""
		if v.SourceVault != nil && v.SourceVault.Id != nil {
			keyVaultId = *v.SourceVault.Id
		}

		certificates := make([]interface{}, 0)

		if v.VaultCertificates != nil {
			for _, c := range *v.VaultCertificates {
				if c.CertificateUrl == nil {
					continue
				}

				certificates = append(certificates, map[string]interface{}{
					"url": *c.CertificateUrl,
				})
			}
		}

		output = append(output, map[string]interface{}{
			"key_vault_id": keyVaultId,
			"certificate":  certificates,
		})
	}

	return output
}

func planSchema() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		ForceNew: true,
		MaxItems: 1,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"name": {
					Type:     pluginsdk.TypeString,
					Required: true,
					ForceNew: true,
				},

				"product": {
					Type:     pluginsdk.TypeString,
					Required: true,
					ForceNew: true,
				},

				"publisher": {
					Type:     pluginsdk.TypeString,
					Required: true,
					ForceNew: true,
				},
			},
		},
	}
}

func expandPlan(input []interface{}) *virtualmachines.Plan {
	if len(input) == 0 || input[0] == nil {
		return nil
	}

	raw := input[0].(map[string]interface{})

	return &virtualmachines.Plan{
		Name:      utils.String(raw["name"].(string)),
		Product:   utils.String(raw["product"].(string)),
		Publisher: utils.String(raw["publisher"].(string)),
	}
}

func flattenPlan(input *virtualmachines.Plan) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	name := ""
	if input.Name != nil {
		name = *input.Name
	}

	product := ""
	if input.Product != nil {
		product = *input.Product
	}

	publisher := ""
	if input.Publisher != nil {
		publisher = *input.Publisher
	}

	return []interface{}{
		map[string]interface{}{
			"name":      name,
			"product":   product,
			"publisher": publisher,
		},
	}
}

func expandVMSSPlan(input []interface{}) *virtualmachinescalesets.Plan {
	if len(input) == 0 || input[0] == nil {
		return nil
	}

	raw := input[0].(map[string]interface{})

	return &virtualmachinescalesets.Plan{
		Name:      utils.String(raw["name"].(string)),
		Product:   utils.String(raw["product"].(string)),
		Publisher: utils.String(raw["publisher"].(string)),
	}
}

func flattenVMSSPlan(input *virtualmachinescalesets.Plan) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	name := ""
	if input.Name != nil {
		name = *input.Name
	}

	product := ""
	if input.Product != nil {
		product = *input.Product
	}

	publisher := ""
	if input.Publisher != nil {
		publisher = *input.Publisher
	}

	return []interface{}{
		map[string]interface{}{
			"name":      name,
			"product":   product,
			"publisher": publisher,
		},
	}
}

func sourceImageReferenceSchema(isVirtualMachine bool) *pluginsdk.Schema {
	// whilst originally I was hoping we could use the 'id' from `azurerm_platform_image' unfortunately Azure doesn't
	// like this as a value for the 'id' field:
	// Id /...../Versions/16.04.201909091 is not a valid resource reference."
	// as such the image is split into two fields (source_image_id and source_image_reference) to provide better validation
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		ForceNew: isVirtualMachine,
		MaxItems: 1,
		ExactlyOneOf: []string{
			"source_image_id",
			"source_image_reference",
		},
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"publisher": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				"offer": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				"sku": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     isVirtualMachine,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				"version": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     isVirtualMachine,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
		},
	}
}

func sourceImageReferenceSchemaOrchestratedVMSS() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		MaxItems: 1,
		ConflictsWith: []string{
			"source_image_id",
		},
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"publisher": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				"offer": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				"sku": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				"version": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
		},
	}
}

func isValidHotPatchSourceImageReference(referenceInput []interface{}, imageId string) bool {
	if imageId != "" {
		return false
	}

	if len(referenceInput) == 0 {
		return false
	}

	raw := referenceInput[0].(map[string]interface{})
	pub := raw["publisher"].(string)
	offer := raw["offer"].(string)
	sku := raw["sku"].(string)

	if pub == "MicrosoftWindowsServer" && offer == "WindowsServer" && (sku == "2022-datacenter-azure-edition-core" || sku == "2022-datacenter-azure-edition-core-smalldisk") {
		return true
	}

	return false
}

func expandSourceImageReference(referenceInput []interface{}, imageId string) *virtualmachines.ImageReference {
	if imageId != "" {
		// With Version            : "/communityGalleries/publicGalleryName/images/myGalleryImageName/versions/(major.minor.patch | latest)"
		// Versionless(e.g. latest): "/communityGalleries/publicGalleryName/images/myGalleryImageName"
		if _, errors := validation.Any(validate.CommunityGalleryImageID, validate.CommunityGalleryImageVersionID)(imageId, "source_image_id"); len(errors) == 0 {
			return &virtualmachines.ImageReference{
				CommunityGalleryImageId: utils.String(imageId),
			}
		}

		// With Version            : "/sharedGalleries/galleryUniqueName/images/myGalleryImageName/versions/(major.minor.patch | latest)"
		// Versionless(e.g. latest): "/sharedGalleries/galleryUniqueName/images/myGalleryImageName"
		if _, errors := validation.Any(validate.SharedGalleryImageID, validate.SharedGalleryImageVersionID)(imageId, "source_image_id"); len(errors) == 0 {
			return &virtualmachines.ImageReference{
				SharedGalleryImageId: utils.String(imageId),
			}
		}

		return &virtualmachines.ImageReference{
			Id: utils.String(imageId),
		}
	}

	raw := referenceInput[0].(map[string]interface{})
	return &virtualmachines.ImageReference{
		Publisher: utils.String(raw["publisher"].(string)),
		Offer:     utils.String(raw["offer"].(string)),
		Sku:       utils.String(raw["sku"].(string)),
		Version:   utils.String(raw["version"].(string)),
	}
}

func flattenSourceImageReference(input *virtualmachines.ImageReference, hasImageId bool) []interface{} {
	// since the image id is pulled out as a separate field, if that's set we should return an empty block here
	if input == nil || hasImageId {
		return []interface{}{}
	}

	var publisher, offer, sku, version string

	if input.Publisher != nil {
		publisher = *input.Publisher
	}
	if input.Offer != nil {
		offer = *input.Offer
	}
	if input.Sku != nil {
		sku = *input.Sku
	}
	if input.Version != nil {
		version = *input.Version
	}

	return []interface{}{
		map[string]interface{}{
			"publisher": publisher,
			"offer":     offer,
			"sku":       sku,
			"version":   version,
		},
	}
}

func expandVMSSSourceImageReference(referenceInput []interface{}, imageId string) *virtualmachinescalesets.ImageReference {
	if imageId != "" {
		// With Version            : "/communityGalleries/publicGalleryName/images/myGalleryImageName/versions/(major.minor.patch | latest)"
		// Versionless(e.g. latest): "/communityGalleries/publicGalleryName/images/myGalleryImageName"
		if _, errors := validation.Any(validate.CommunityGalleryImageID, validate.CommunityGalleryImageVersionID)(imageId, "source_image_id"); len(errors) == 0 {
			return &virtualmachinescalesets.ImageReference{
				CommunityGalleryImageId: utils.String(imageId),
			}
		}

		// With Version            : "/sharedGalleries/galleryUniqueName/images/myGalleryImageName/versions/(major.minor.patch | latest)"
		// Versionless(e.g. latest): "/sharedGalleries/galleryUniqueName/images/myGalleryImageName"
		if _, errors := validation.Any(validate.SharedGalleryImageID, validate.SharedGalleryImageVersionID)(imageId, "source_image_id"); len(errors) == 0 {
			return &virtualmachinescalesets.ImageReference{
				SharedGalleryImageId: utils.String(imageId),
			}
		}

		return &virtualmachinescalesets.ImageReference{
			Id: utils.String(imageId),
		}
	}

	raw := referenceInput[0].(map[string]interface{})
	return &virtualmachinescalesets.ImageReference{
		Publisher: utils.String(raw["publisher"].(string)),
		Offer:     utils.String(raw["offer"].(string)),
		Sku:       utils.String(raw["sku"].(string)),
		Version:   utils.String(raw["version"].(string)),
	}
}

func flattenVMSSSourceImageReference(input *virtualmachinescalesets.ImageReference, hasImageId bool) []interface{} {
	// since the image id is pulled out as a separate field, if that's set we should return an empty block here
	if input == nil || hasImageId {
		return []interface{}{}
	}

	var publisher, offer, sku, version string

	if input.Publisher != nil {
		publisher = *input.Publisher
	}
	if input.Offer != nil {
		offer = *input.Offer
	}
	if input.Sku != nil {
		sku = *input.Sku
	}
	if input.Version != nil {
		version = *input.Version
	}

	return []interface{}{
		map[string]interface{}{
			"publisher": publisher,
			"offer":     offer,
			"sku":       sku,
			"version":   version,
		},
	}
}

func winRmListenerSchema() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeSet,
		Optional: true,
		// Whilst the SDK allows you to modify this, the API does not:
		//   Code="PropertyChangeNotAllowed"
		//   Message="Changing property 'windowsConfiguration.winRM.listeners' is not allowed."
		//   Target="windowsConfiguration.winRM.listeners"
		ForceNew: true,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"protocol": {
					Type:     pluginsdk.TypeString,
					Required: true,
					ForceNew: true,
					ValidateFunc: validation.StringInSlice([]string{
						string(compute.ProtocolTypesHTTP),
						string(compute.ProtocolTypesHTTPS),
					}, false),
				},

				"certificate_url": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					ForceNew:     true,
					ValidateFunc: keyVaultValidate.NestedItemId,
				},
			},
		},
	}
}

func expandWinRMListener(input []interface{}) *virtualmachines.WinRMConfiguration {
	listeners := make([]virtualmachines.WinRMListener, 0)

	for _, v := range input {
		raw := v.(map[string]interface{})

		listener := virtualmachines.WinRMListener{
			Protocol: pointer.To(virtualmachines.ProtocolTypes(raw["protocol"].(string))),
		}

		certificateUrl := raw["certificate_url"].(string)
		if certificateUrl != "" {
			listener.CertificateUrl = utils.String(certificateUrl)
		}

		listeners = append(listeners, listener)
	}

	return &virtualmachines.WinRMConfiguration{
		Listeners: &listeners,
	}
}

func flattenWinRMListener(input *virtualmachines.WinRMConfiguration) []interface{} {
	if input == nil || input.Listeners == nil {
		return []interface{}{}
	}

	output := make([]interface{}, 0)

	for _, v := range *input.Listeners {
		certificateUrl := ""
		if v.CertificateUrl != nil {
			certificateUrl = *v.CertificateUrl
		}

		output = append(output, map[string]interface{}{
			"certificate_url": certificateUrl,
			"protocol":        string(pointer.From(v.Protocol)),
		})
	}

	return output
}

func expandVMSSWinRMListener(input []interface{}) *virtualmachinescalesets.WinRMConfiguration {
	listeners := make([]virtualmachinescalesets.WinRMListener, 0)

	for _, v := range input {
		raw := v.(map[string]interface{})

		listener := virtualmachinescalesets.WinRMListener{
			Protocol: pointer.To(virtualmachinescalesets.ProtocolTypes(raw["protocol"].(string))),
		}

		certificateUrl := raw["certificate_url"].(string)
		if certificateUrl != "" {
			listener.CertificateUrl = utils.String(certificateUrl)
		}

		listeners = append(listeners, listener)
	}

	return &virtualmachinescalesets.WinRMConfiguration{
		Listeners: &listeners,
	}
}

func flattenVMSSWinRMListener(input *virtualmachinescalesets.WinRMConfiguration) []interface{} {
	if input == nil || input.Listeners == nil {
		return []interface{}{}
	}

	output := make([]interface{}, 0)

	for _, v := range *input.Listeners {
		certificateUrl := ""
		if v.CertificateUrl != nil {
			certificateUrl = *v.CertificateUrl
		}

		output = append(output, map[string]interface{}{
			"certificate_url": certificateUrl,
			"protocol":        string(pointer.From(v.Protocol)),
		})
	}

	return output
}

func windowsSecretSchema() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				// whilst this isn't present in the nested object it's required when this is specified
				"key_vault_id": commonschema.ResourceIDReferenceRequired(commonids.KeyVaultId{}),

				"certificate": {
					Type:     pluginsdk.TypeSet,
					Required: true,
					MinItems: 1,
					Elem: &pluginsdk.Resource{
						Schema: map[string]*pluginsdk.Schema{
							"store": {
								Type:     pluginsdk.TypeString,
								Required: true,
							},
							"url": {
								Type:         pluginsdk.TypeString,
								Required:     true,
								ValidateFunc: keyVaultValidate.NestedItemId,
							},
						},
					},
				},
			},
		},
	}
}

func expandWindowsSecrets(input []interface{}) *[]virtualmachines.VaultSecretGroup {
	output := make([]virtualmachines.VaultSecretGroup, 0)

	for _, raw := range input {
		v := raw.(map[string]interface{})

		keyVaultId := v["key_vault_id"].(string)
		certificatesRaw := v["certificate"].(*pluginsdk.Set).List()
		certificates := make([]virtualmachines.VaultCertificate, 0)
		for _, certificateRaw := range certificatesRaw {
			certificateV := certificateRaw.(map[string]interface{})

			store := certificateV["store"].(string)
			url := certificateV["url"].(string)
			certificates = append(certificates, virtualmachines.VaultCertificate{
				CertificateStore: utils.String(store),
				CertificateUrl:   utils.String(url),
			})
		}

		output = append(output, virtualmachines.VaultSecretGroup{
			SourceVault: &virtualmachines.SubResource{
				Id: utils.String(keyVaultId),
			},
			VaultCertificates: &certificates,
		})
	}

	return &output
}

func flattenWindowsSecrets(input *[]virtualmachines.VaultSecretGroup) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	output := make([]interface{}, 0)

	for _, v := range *input {
		keyVaultId := ""
		if v.SourceVault != nil && v.SourceVault.Id != nil {
			keyVaultId = *v.SourceVault.Id
		}

		certificates := make([]interface{}, 0)

		if v.VaultCertificates != nil {
			for _, c := range *v.VaultCertificates {
				store := ""
				if c.CertificateStore != nil {
					store = *c.CertificateStore
				}

				url := ""
				if c.CertificateUrl != nil {
					url = *c.CertificateUrl
				}

				certificates = append(certificates, map[string]interface{}{
					"store": store,
					"url":   url,
				})
			}
		}

		output = append(output, map[string]interface{}{
			"key_vault_id": keyVaultId,
			"certificate":  certificates,
		})
	}

	return output
}

func expandVMSSWindowsSecrets(input []interface{}) *[]virtualmachinescalesets.VaultSecretGroup {
	output := make([]virtualmachinescalesets.VaultSecretGroup, 0)

	for _, raw := range input {
		v := raw.(map[string]interface{})

		keyVaultId := v["key_vault_id"].(string)
		certificatesRaw := v["certificate"].(*pluginsdk.Set).List()
		certificates := make([]virtualmachinescalesets.VaultCertificate, 0)
		for _, certificateRaw := range certificatesRaw {
			certificateV := certificateRaw.(map[string]interface{})

			store := certificateV["store"].(string)
			url := certificateV["url"].(string)
			certificates = append(certificates, virtualmachinescalesets.VaultCertificate{
				CertificateStore: utils.String(store),
				CertificateUrl:   utils.String(url),
			})
		}

		output = append(output, virtualmachinescalesets.VaultSecretGroup{
			SourceVault: &virtualmachinescalesets.SubResource{
				Id: utils.String(keyVaultId),
			},
			VaultCertificates: &certificates,
		})
	}

	return &output
}

func flattenVMSSWindowsSecrets(input *[]virtualmachinescalesets.VaultSecretGroup) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	output := make([]interface{}, 0)

	for _, v := range *input {
		keyVaultId := ""
		if v.SourceVault != nil && v.SourceVault.Id != nil {
			keyVaultId = *v.SourceVault.Id
		}

		certificates := make([]interface{}, 0)

		if v.VaultCertificates != nil {
			for _, c := range *v.VaultCertificates {
				store := ""
				if c.CertificateStore != nil {
					store = *c.CertificateStore
				}

				url := ""
				if c.CertificateUrl != nil {
					url = *c.CertificateUrl
				}

				certificates = append(certificates, map[string]interface{}{
					"store": store,
					"url":   url,
				})
			}
		}

		output = append(output, map[string]interface{}{
			"key_vault_id": keyVaultId,
			"certificate":  certificates,
		})
	}

	return output
}
