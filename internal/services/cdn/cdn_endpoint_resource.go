// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cdn

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/cdn/mgmt/2020-09-01/cdn" // nolint: staticcheck
	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cdn/migration"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cdn/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cdn/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tags"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceCdnEndpoint() *pluginsdk.Resource {
	resource := &pluginsdk.Resource{
		Create: resourceCdnEndpointCreate,
		Read:   resourceCdnEndpointRead,
		Update: resourceCdnEndpointUpdate,
		Delete: resourceCdnEndpointDelete,

		SchemaVersion: 1,
		StateUpgraders: pluginsdk.StateUpgrades(map[int]pluginsdk.StateUpgrade{
			0: migration.CdnEndpointV0ToV1{},
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.EndpointID(id)
			return err
		}),

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:     pluginsdk.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": commonschema.Location(),

			"resource_group_name": commonschema.ResourceGroupName(),

			"profile_name": {
				Type:     pluginsdk.TypeString,
				Required: true,
				ForceNew: true,
			},

			"origin_host_header": {
				Type:     pluginsdk.TypeString,
				Optional: true,
			},

			"is_http_allowed": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  true,
			},

			"is_https_allowed": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  true,
			},

			"origin_path": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				Computed: true,
			},

			"querystring_caching_behaviour": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				Default:  string(cdn.QueryStringCachingBehaviorIgnoreQueryString),
				ValidateFunc: validation.StringInSlice([]string{
					string(cdn.QueryStringCachingBehaviorBypassCaching),
					string(cdn.QueryStringCachingBehaviorIgnoreQueryString),
					string(cdn.QueryStringCachingBehaviorNotSet),
					string(cdn.QueryStringCachingBehaviorUseQueryString),
				}, false),
			},

			"content_types_to_compress": {
				Type:     pluginsdk.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &pluginsdk.Schema{
					Type: pluginsdk.TypeString,
				},
				Set: pluginsdk.HashString,
			},

			"is_compression_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
			},

			"probe_path": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				Computed: true,
			},

			"geo_filter": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"relative_path": {
							Type:     pluginsdk.TypeString,
							Required: true,
						},
						"action": {
							Type:     pluginsdk.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(cdn.ActionTypeAllow),
								string(cdn.ActionTypeBlock),
							}, false),
						},
						"country_codes": {
							Type:     pluginsdk.TypeList,
							Required: true,
							Elem: &pluginsdk.Schema{
								Type: pluginsdk.TypeString,
							},
						},
					},
				},
			},

			"optimization_type": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(cdn.OptimizationTypeDynamicSiteAcceleration),
					string(cdn.OptimizationTypeGeneralMediaStreaming),
					string(cdn.OptimizationTypeGeneralWebDelivery),
					string(cdn.OptimizationTypeLargeFileDownload),
					string(cdn.OptimizationTypeVideoOnDemandMediaStreaming),
				}, false),
			},

			"fqdn": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"global_delivery_rule": endpointGlobalDeliveryRule(),

			"delivery_rule": endpointDeliveryRule(),

			"tags": tags.Schema(),
		},
	}

	if !features.FourPointOhBeta() {
		resource.Schema["origin"] = &pluginsdk.Schema{
			Type:          pluginsdk.TypeSet,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ConflictsWith: []string{"origins"},
			Deprecated:    "This property has been deprecated in favour of the `origins` property and will be removed from the v4.0 azurerm provider.",
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"name": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ForceNew:     true,
						ValidateFunc: validate.OriginName,
					},

					"host_name": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ForceNew:     true,
						ValidateFunc: validation.StringIsNotEmpty,
					},

					"http_port": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						ForceNew:     true,
						Default:      80,
						ValidateFunc: validation.IntBetween(1, 65535),
					},

					"https_port": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						ForceNew:     true,
						Default:      443,
						ValidateFunc: validation.IntBetween(1, 65535),
					},
				},
			},
		}

		resource.Schema["origins"] = &pluginsdk.Schema{
			Type:          pluginsdk.TypeSet,
			Optional:      true,
			Computed:      true,
			ConflictsWith: []string{"origin"},
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"name": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ValidateFunc: validate.OriginName,
					},

					"host_name": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ValidateFunc: validation.Any(validation.IsIPv6Address, validation.IsIPv4Address, validation.StringIsNotEmpty),
					},

					"http_port": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						Default:      80,
						ValidateFunc: validation.IntBetween(1, 65535),
					},

					"https_port": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						Default:      443,
						ValidateFunc: validation.IntBetween(1, 65535),
					},

					// NOTE: If this is not defined in the configuration file it will default
					// to the `host_name` value...
					"origin_host_header": {
						Type:         pluginsdk.TypeString,
						Optional:     true,
						Computed:     true,
						ValidateFunc: validation.Any(validation.IsIPv6Address, validation.IsIPv4Address, validation.StringIsNotEmpty),
					},

					"priority": {
						Type:     pluginsdk.TypeInt,
						Optional: true,
						// Default:      1,
						ValidateFunc: validation.IntBetween(1, 5),
					},

					"weight": {
						Type:     pluginsdk.TypeInt,
						Optional: true,
						// Default:      1000,
						ValidateFunc: validation.IntBetween(1, 1000),
					},

					"id": {
						Type:     pluginsdk.TypeString,
						Computed: true,
					},
				},
			},
		}

		// Origin or Origins is a required field, one or the other must be defined...
		resource.CustomizeDiff = pluginsdk.CustomDiffWithAll(
			func(ctx context.Context, diff *pluginsdk.ResourceDiff, v interface{}) error {
				origin := diff.Get("origin").(*pluginsdk.Set).List()
				origins := diff.Get("origins").(*pluginsdk.Set).List()

				if len(origin) == 0 && len(origins) == 0 {
					return fmt.Errorf("at least one of the following fields must be defined: `origin` or `origins`")
				}

				return nil
			},
		)
	} else {
		resource.Schema["origins"] = &pluginsdk.Schema{
			Type:     pluginsdk.TypeSet,
			Required: true,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"name": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ValidateFunc: validate.OriginName,
					},

					"host_name": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ValidateFunc: validation.StringIsNotEmpty,
					},

					"http_port": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						Default:      80,
						ValidateFunc: validation.IntBetween(1, 65535),
					},

					"https_port": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						Default:      443,
						ValidateFunc: validation.IntBetween(1, 65535),
					},

					// NOTE: If this is not defined in the configuration file it will default
					// to the `host_name` value...
					"origin_host_header": {
						Type:         pluginsdk.TypeString,
						Optional:     true,
						Computed:     true,
						ValidateFunc: validation.Any(validation.IsIPv6Address, validation.IsIPv4Address, validation.StringIsNotEmpty),
					},

					"priority": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						Default:      1,
						ValidateFunc: validation.IntBetween(1, 5),
					},

					"weight": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						Default:      1000,
						ValidateFunc: validation.IntBetween(1, 1000),
					},

					"id": {
						Type:     pluginsdk.TypeString,
						Computed: true,
					},
				},
			},
		}
	}

	return resource
}

func resourceCdnEndpointCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	endpointsClient := meta.(*clients.Client).Cdn.EndpointsClient
	profilesClient := meta.(*clients.Client).Cdn.ProfilesClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for Azure ARM CDN EndPoint creation.")

	id := parse.NewEndpointID(subscriptionId, d.Get("resource_group_name").(string), d.Get("profile_name").(string), d.Get("name").(string))
	existing, err := endpointsClient.Get(ctx, id.ResourceGroup, id.ProfileName, id.Name)
	if err != nil {
		if !utils.ResponseWasNotFound(existing.Response) {
			return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
		}
	}

	if !utils.ResponseWasNotFound(existing.Response) {
		return tf.ImportAsExistsError("azurerm_cdn_endpoint", id.ID())
	}

	location := azure.NormalizeLocation(d.Get("location").(string))
	httpAllowed := d.Get("is_http_allowed").(bool)
	httpsAllowed := d.Get("is_https_allowed").(bool)
	cachingBehaviour := d.Get("querystring_caching_behaviour").(string)
	originPath := d.Get("origin_path").(string)
	probePath := d.Get("probe_path").(string)
	optimizationType := d.Get("optimization_type").(string)
	t := d.Get("tags").(map[string]interface{})

	endpoint := cdn.Endpoint{
		Location: &location,
		EndpointProperties: &cdn.EndpointProperties{
			IsHTTPAllowed:              &httpAllowed,
			IsHTTPSAllowed:             &httpsAllowed,
			QueryStringCachingBehavior: cdn.QueryStringCachingBehavior(cachingBehaviour),
		},
		Tags: tags.Expand(t),
	}

	if v, ok := d.GetOk("origin_host_header"); ok {
		endpoint.EndpointProperties.OriginHostHeader = utils.String(v.(string))
	}

	if _, ok := d.GetOk("content_types_to_compress"); ok {
		contentTypes := expandArmCdnEndpointContentTypesToCompress(d)
		endpoint.EndpointProperties.ContentTypesToCompress = &contentTypes
	}

	if _, ok := d.GetOk("geo_filter"); ok {
		geoFilters := expandCdnEndpointGeoFilters(d)
		endpoint.EndpointProperties.GeoFilters = geoFilters
	}

	if v, ok := d.GetOk("is_compression_enabled"); ok {
		endpoint.EndpointProperties.IsCompressionEnabled = utils.Bool(v.(bool))
	}

	if optimizationType != "" {
		endpoint.EndpointProperties.OptimizationType = cdn.OptimizationType(optimizationType)
	}

	if originPath != "" {
		endpoint.EndpointProperties.OriginPath = utils.String(originPath)
	}

	if probePath != "" {
		endpoint.EndpointProperties.ProbePath = utils.String(probePath)
	}

	if !features.FourPointOhBeta() {
		originCount := len(d.Get("origin").(*pluginsdk.Set).List())

		if originCount > 0 {
			origins := expandAzureRmCdnEndpointOrigin(d)
			if len(origins) > 0 {
				endpoint.EndpointProperties.Origins = &origins
			}
		}
	}

	originsRaw := d.Get("origins").(*pluginsdk.Set).List()
	originsCount := len(originsRaw)
	if originsCount > 0 {
		origins := expandAzureRmCdnEndpointOrigins(originsRaw, nil)

		if originsCount > 1 {
			return fmt.Errorf("%s: creating more than one 'origins' is not allowed if the Default Origin Group has not been set", id)
		}

		// NOTE: If the endpoint does not have an origin group associated with it you cannot
		// specify priority, weight or origin_host_header for the origin (e.g., it's in single origin mode)...
		if err := validateAzureRmCdnEndpointOriginsInvalidProperties(originsRaw[0].(map[string]interface{}), id); err != nil {
			return err
		}

		endpoint.EndpointProperties.Origins = &origins
	}

	profile, err := profilesClient.Get(ctx, id.ResourceGroup, id.ProfileName)
	if err != nil {
		return fmt.Errorf("retrieving parent CDN Profile for %s: %+v", id, err)
	}

	if profile.Sku != nil {
		globalDeliveryRulesRaw := d.Get("global_delivery_rule").([]interface{})
		deliveryRulesRaw := d.Get("delivery_rule").([]interface{})
		deliveryPolicy, err := expandArmCdnEndpointDeliveryPolicy(globalDeliveryRulesRaw, deliveryRulesRaw)
		if err != nil {
			return fmt.Errorf("expanding `global_delivery_rule` or `delivery_rule`: %s", err)
		}

		if profile.Sku.Name != cdn.SkuNameStandardMicrosoft && len(*deliveryPolicy.Rules) > 0 {
			return fmt.Errorf("`global_delivery_rule` and `delivery_rule` are only allowed when `Standard_Microsoft` sku is used. Profile sku:  %s", profile.Sku.Name)
		}

		if profile.Sku.Name == cdn.SkuNameStandardMicrosoft {
			endpoint.EndpointProperties.DeliveryPolicy = deliveryPolicy
		}
	}

	future, err := endpointsClient.Create(ctx, id.ResourceGroup, id.ProfileName, id.Name, endpoint)
	if err != nil {
		return fmt.Errorf("creating %s: %+v", id, err)
	}

	if err = future.WaitForCompletionRef(ctx, endpointsClient.Client); err != nil {
		return fmt.Errorf("waiting for the creation of %s: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceCdnEndpointRead(d, meta)
}

func resourceCdnEndpointUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	endpointsClient := meta.(*clients.Client).Cdn.EndpointsClient
	profilesClient := meta.(*clients.Client).Cdn.ProfilesClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for Azure ARM CDN EndPoint update.")

	id, err := parse.EndpointID(d.Id())
	if err != nil {
		return err
	}

	existing, err := endpointsClient.Get(ctx, id.ResourceGroup, id.ProfileName, id.Name)
	if err != nil {
		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}

	location := azure.NormalizeLocation(d.Get("location").(string))
	httpAllowed := d.Get("is_http_allowed").(bool)
	httpsAllowed := d.Get("is_https_allowed").(bool)
	cachingBehaviour := d.Get("querystring_caching_behaviour").(string)
	originPath := d.Get("origin_path").(string)
	probePath := d.Get("probe_path").(string)
	optimizationType := d.Get("optimization_type").(string)
	t := d.Get("tags").(map[string]interface{})

	// NOTE: "Only tags can be updated after creating an endpoint." So only
	// call 'PATCH' if the only thing that has changed are the tags, else
	// call the 'PUT' instead. https://learn.microsoft.com/rest/api/cdn/endpoints/update?tabs=HTTP
	// see issue #22326 for more details.
	updateTypePATCH := true

	if d.HasChanges("is_http_allowed", "is_https_allowed", "querystring_caching_behaviour", "origin_path",
		"probe_path", "optimization_type", "origin_host_header", "content_types_to_compress", "geo_filter",
		"is_compression_enabled", "probe_path", "geo_filter", "optimization_type", "global_delivery_rule",
		"delivery_rule", "origins") {
		updateTypePATCH = false
	}

	if updateTypePATCH {
		log.Printf("[INFO] No changes detected using PATCH for Azure ARM CDN EndPoint update.")

		if !d.HasChange("tags") {
			log.Printf("[INFO] 'tags' did not change, skipping Azure ARM CDN EndPoint update.")
			return resourceCdnEndpointRead(d, meta)
		}

		endpoint := cdn.EndpointUpdateParameters{
			EndpointPropertiesUpdateParameters: &cdn.EndpointPropertiesUpdateParameters{},
			Tags:                               tags.Expand(t),
		}

		future, err := endpointsClient.Update(ctx, id.ResourceGroup, id.ProfileName, id.Name, endpoint)
		if err != nil {
			return fmt.Errorf("updating %s: %+v", *id, err)
		}

		if err = future.WaitForCompletionRef(ctx, endpointsClient.Client); err != nil {
			return fmt.Errorf("waiting for update of %s: %+v", *id, err)
		}
	} else {
		log.Printf("[INFO] One or more fields have changed using PUT for Azure ARM CDN EndPoint update.")

		endpoint := cdn.Endpoint{
			Location: &location,
			EndpointProperties: &cdn.EndpointProperties{
				IsHTTPAllowed:              &httpAllowed,
				IsHTTPSAllowed:             &httpsAllowed,
				QueryStringCachingBehavior: cdn.QueryStringCachingBehavior(cachingBehaviour),
			},
			Tags: tags.Expand(t),
		}

		if v, ok := d.GetOk("origin_host_header"); ok {
			endpoint.EndpointProperties.OriginHostHeader = utils.String(v.(string))
		}

		if _, ok := d.GetOk("content_types_to_compress"); ok {
			contentTypes := expandArmCdnEndpointContentTypesToCompress(d)
			endpoint.EndpointProperties.ContentTypesToCompress = &contentTypes
		}

		if _, ok := d.GetOk("geo_filter"); ok {
			geoFilters := expandCdnEndpointGeoFilters(d)
			endpoint.EndpointProperties.GeoFilters = geoFilters
		}

		if v, ok := d.GetOk("is_compression_enabled"); ok {
			endpoint.EndpointProperties.IsCompressionEnabled = utils.Bool(v.(bool))
		}

		if optimizationType != "" {
			endpoint.EndpointProperties.OptimizationType = cdn.OptimizationType(optimizationType)
		}

		if originPath != "" {
			endpoint.EndpointProperties.OriginPath = utils.String(originPath)
		}

		if probePath != "" {
			endpoint.EndpointProperties.ProbePath = utils.String(probePath)
		}

		// NOTE: Origin is ForceNew so there will never be an update, only create...
		originsRaw := d.Get("origins").(*pluginsdk.Set).List()
		origins := expandAzureRmCdnEndpointOrigins(originsRaw, &existing)
		originsCount := len(origins)

		if originsCount > 1 && existing.DefaultOriginGroup == nil {
			return fmt.Errorf("%s: creating more than one 'origins' is not allowed if the Default Origin Group has not been set", id)
		}

		endpoint.EndpointProperties.Origins = &origins

		profile, err := profilesClient.Get(ctx, id.ResourceGroup, id.ProfileName)
		if err != nil {
			return fmt.Errorf("retrieving parent CDN Profile for %s: %+v", id, err)
		}

		if profile.Sku != nil {
			globalDeliveryRulesRaw := d.Get("global_delivery_rule").([]interface{})
			deliveryRulesRaw := d.Get("delivery_rule").([]interface{})
			deliveryPolicy, err := expandArmCdnEndpointDeliveryPolicy(globalDeliveryRulesRaw, deliveryRulesRaw)
			if err != nil {
				return fmt.Errorf("expanding `global_delivery_rule` or `delivery_rule`: %s", err)
			}

			if profile.Sku.Name != cdn.SkuNameStandardMicrosoft && len(*deliveryPolicy.Rules) > 0 {
				return fmt.Errorf("`global_delivery_rule` and `delivery_rule` are only allowed when `Standard_Microsoft` sku is used. Profile sku:  %s", profile.Sku.Name)
			}

			if profile.Sku.Name == cdn.SkuNameStandardMicrosoft {
				endpoint.EndpointProperties.DeliveryPolicy = deliveryPolicy
			}
		}

		future, err := endpointsClient.Create(ctx, id.ResourceGroup, id.ProfileName, id.Name, endpoint)
		if err != nil {
			return fmt.Errorf("updating %s: %+v", id, err)
		}

		if err = future.WaitForCompletionRef(ctx, endpointsClient.Client); err != nil {
			return fmt.Errorf("waiting for update of %s: %+v", id, err)
		}
	}

	return resourceCdnEndpointRead(d, meta)
}

func resourceCdnEndpointRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cdn.EndpointsClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.EndpointID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.ProfileName, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}

	d.Set("name", id.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("profile_name", id.ProfileName)
	d.Set("location", location.NormalizeNilable(resp.Location))

	if props := resp.EndpointProperties; props != nil {
		d.Set("fqdn", props.HostName)
		d.Set("is_http_allowed", props.IsHTTPAllowed)
		d.Set("is_https_allowed", props.IsHTTPSAllowed)
		d.Set("querystring_caching_behaviour", props.QueryStringCachingBehavior)
		d.Set("origin_host_header", props.OriginHostHeader)
		d.Set("origin_path", props.OriginPath)
		d.Set("probe_path", props.ProbePath)
		d.Set("optimization_type", string(props.OptimizationType))

		compressionEnabled := false
		if v := props.IsCompressionEnabled; v != nil {
			compressionEnabled = *v
		}
		d.Set("is_compression_enabled", compressionEnabled)

		contentTypes := flattenAzureRMCdnEndpointContentTypes(props.ContentTypesToCompress)
		if err := d.Set("content_types_to_compress", contentTypes); err != nil {
			return fmt.Errorf("setting `content_types_to_compress`: %+v", err)
		}

		geoFilters := flattenCdnEndpointGeoFilters(props.GeoFilters)
		if err := d.Set("geo_filter", geoFilters); err != nil {
			return fmt.Errorf("setting `geo_filter`: %+v", err)
		}

		if !features.FourPointOhBeta() {
			origins := flattenAzureRMCdnEndpointOrigin(props.Origins)
			if err := d.Set("origin", origins); err != nil {
				return fmt.Errorf("setting `origin`: %+v", err)
			}
		}

		origins := flattenAzureRMCdnEndpointOrigins(props.Origins, subscriptionId, id)
		if err := d.Set("origins", origins); err != nil {
			return fmt.Errorf("setting `origins`: %+v", err)
		}

		flattenedDeliveryPolicies, err := flattenEndpointDeliveryPolicy(props.DeliveryPolicy)
		if err != nil {
			return err
		}
		if err := d.Set("global_delivery_rule", flattenedDeliveryPolicies.globalDeliveryRules); err != nil {
			return fmt.Errorf("setting `global_delivery_rule`: %+v", err)
		}
		if err := d.Set("delivery_rule", flattenedDeliveryPolicies.deliveryRules); err != nil {
			return fmt.Errorf("setting `delivery_rule`: %+v", err)
		}
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceCdnEndpointDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cdn.EndpointsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.EndpointID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.Delete(ctx, id.ResourceGroup, id.ProfileName, id.Name)
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", *id, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for deletion of %s: %+v", *id, err)
	}

	return nil
}

func expandCdnEndpointGeoFilters(d *pluginsdk.ResourceData) *[]cdn.GeoFilter {
	filters := make([]cdn.GeoFilter, 0)

	inputFilters := d.Get("geo_filter").([]interface{})
	for _, v := range inputFilters {
		input := v.(map[string]interface{})
		action := input["action"].(string)
		relativePath := input["relative_path"].(string)

		inputCountryCodes := input["country_codes"].([]interface{})
		countryCodes := make([]string, 0)

		for _, v := range inputCountryCodes {
			if v != nil {
				countryCode := v.(string)
				countryCodes = append(countryCodes, countryCode)
			}
		}

		filter := cdn.GeoFilter{
			Action:       cdn.GeoFilterActions(action),
			RelativePath: utils.String(relativePath),
			CountryCodes: &countryCodes,
		}
		filters = append(filters, filter)
	}

	return &filters
}

func flattenCdnEndpointGeoFilters(input *[]cdn.GeoFilter) []interface{} {
	results := make([]interface{}, 0)

	if filters := input; filters != nil {
		for _, filter := range *filters {
			relativePath := ""
			if filter.RelativePath != nil {
				relativePath = *filter.RelativePath
			}

			outputCodes := make([]interface{}, 0)
			if codes := filter.CountryCodes; codes != nil {
				for _, code := range *codes {
					outputCodes = append(outputCodes, code)
				}
			}

			results = append(results, map[string]interface{}{
				"action":        string(filter.Action),
				"country_codes": outputCodes,
				"relative_path": relativePath,
			})
		}
	}

	return results
}

func expandArmCdnEndpointContentTypesToCompress(d *pluginsdk.ResourceData) []string {
	results := make([]string, 0)
	input := d.Get("content_types_to_compress").(*pluginsdk.Set).List()

	for _, v := range input {
		contentType := v.(string)
		results = append(results, contentType)
	}

	return results
}

func flattenAzureRMCdnEndpointContentTypes(input *[]string) []interface{} {
	output := make([]interface{}, 0)

	if input != nil {
		for _, v := range *input {
			output = append(output, v)
		}
	}

	return output
}

// TODO: Remove in 4.0
func expandAzureRmCdnEndpointOrigin(d *pluginsdk.ResourceData) []cdn.DeepCreatedOrigin {
	configs := d.Get("origin").(*pluginsdk.Set).List()
	origins := make([]cdn.DeepCreatedOrigin, 0)

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		hostName := data["host_name"].(string)

		origin := cdn.DeepCreatedOrigin{
			Name: utils.String(name),
			DeepCreatedOriginProperties: &cdn.DeepCreatedOriginProperties{
				HostName: utils.String(hostName),
			},
		}

		if v, ok := data["https_port"]; ok {
			port := v.(int)
			origin.DeepCreatedOriginProperties.HTTPSPort = utils.Int32(int32(port))
		}

		if v, ok := data["http_port"]; ok {
			port := v.(int)
			origin.DeepCreatedOriginProperties.HTTPPort = utils.Int32(int32(port))
		}

		origins = append(origins, origin)
	}

	return origins
}

func expandAzureRmCdnEndpointOrigins(input []interface{}, endpoint *cdn.Endpoint) []cdn.DeepCreatedOrigin {
	origins := make([]cdn.DeepCreatedOrigin, 0)

	if len(input) == 0 {
		return origins
	}

	for _, v := range input {
		data := v.(map[string]interface{})

		origin := cdn.DeepCreatedOrigin{
			DeepCreatedOriginProperties: &cdn.DeepCreatedOriginProperties{},
		}

		if v, ok := data["name"]; ok {
			origin.Name = pointer.To(v.(string))
		}

		if v, ok := data["host_name"]; ok {
			origin.DeepCreatedOriginProperties.HostName = pointer.To(v.(string))
		}

		if v, ok := data["http_port"]; ok {
			origin.DeepCreatedOriginProperties.HTTPPort = pointer.To(int32(v.(int)))
		}

		if v, ok := data["https_port"]; ok {
			origin.DeepCreatedOriginProperties.HTTPSPort = pointer.To(int32(v.(int)))
		}

		// NOTE: If the endpoint does not have an origin group associated with it you cannot
		// specify priority, weight or origin_host_header for the origin...
		if endpoint != nil && endpoint.DefaultOriginGroup != nil {
			if v, ok := data["priority"]; ok {
				origin.DeepCreatedOriginProperties.Priority = pointer.To(int32(v.(int)))
			}

			if v, ok := data["weight"]; ok {
				origin.DeepCreatedOriginProperties.Weight = pointer.To(int32(v.(int)))
			}

			if v, ok := data["origin_host_header"]; ok {
				origin.DeepCreatedOriginProperties.OriginHostHeader = pointer.To(v.(string))
			}
		}

		origins = append(origins, origin)
	}

	return origins
}

// TODO: Remove in 4.0
func flattenAzureRMCdnEndpointOrigin(input *[]cdn.DeepCreatedOrigin) []interface{} {
	results := make([]interface{}, 0)

	if list := input; list != nil {
		for _, i := range *list {
			name := ""
			if i.Name != nil {
				name = *i.Name
			}

			hostName := ""
			httpPort := 80
			httpsPort := 443
			if props := i.DeepCreatedOriginProperties; props != nil {
				if props.HostName != nil {
					hostName = *props.HostName
				}
				if port := props.HTTPPort; port != nil {
					httpPort = int(*port)
				}
				if port := props.HTTPSPort; port != nil {
					httpsPort = int(*port)
				}
			}

			results = append(results, map[string]interface{}{
				"name":       name,
				"host_name":  hostName,
				"http_port":  httpPort,
				"https_port": httpsPort,
			})
		}
	}

	return results
}

func flattenAzureRMCdnEndpointOrigins(input *[]cdn.DeepCreatedOrigin, subscriptionId string, endpointId *parse.EndpointId) []interface{} {
	results := make([]interface{}, 0)

	if list := input; list != nil {
		for _, i := range *list {
			name := ""
			if i.Name != nil {
				name = *i.Name
			}

			id := parse.NewOriginID(subscriptionId, endpointId.ResourceGroup, endpointId.ProfileName, endpointId.Name, name)

			var hostName string
			var httpPort int32
			var httpsPort int32
			var originHostHeader string
			var priority int32
			var weight int32

			if props := i.DeepCreatedOriginProperties; props != nil {
				if v := props.HostName; v != nil {
					hostName = pointer.From(v)
				}

				if v := props.HTTPPort; v != nil {
					httpPort = pointer.From(v)
				}

				if v := props.HTTPSPort; v != nil {
					httpsPort = pointer.From(v)
				}

				if v := props.OriginHostHeader; v != nil {
					originHostHeader = pointer.From(v)
				}

				if v := props.Priority; v != nil {
					priority = pointer.From(v)
				}

				if v := props.Weight; v != nil {
					weight = pointer.From(v)
				}
			}

			results = append(results, map[string]interface{}{
				"name":               name,
				"host_name":          hostName,
				"http_port":          httpPort,
				"https_port":         httpsPort,
				"origin_host_header": originHostHeader,
				"priority":           priority,
				"weight":             weight,
				"id":                 id.ID(),
			})
		}
	}

	return results
}

func expandArmCdnEndpointDeliveryPolicy(globalRulesRaw []interface{}, deliveryRulesRaw []interface{}) (*cdn.EndpointPropertiesUpdateParametersDeliveryPolicy, error) {
	deliveryRules := make([]cdn.DeliveryRule, 0)
	deliveryPolicy := cdn.EndpointPropertiesUpdateParametersDeliveryPolicy{
		Description: utils.String(""),
		Rules:       &deliveryRules,
	}

	if len(globalRulesRaw) > 0 && globalRulesRaw[0] != nil {
		ruleRaw := globalRulesRaw[0].(map[string]interface{})
		rule, err := expandArmCdnEndpointGlobalDeliveryRule(ruleRaw)
		if err != nil {
			return nil, err
		}
		deliveryRules = append(deliveryRules, *rule)
	}

	for _, ruleV := range deliveryRulesRaw {
		ruleRaw := ruleV.(map[string]interface{})
		rule, err := expandArmCdnEndpointDeliveryRule(ruleRaw)
		if err != nil {
			return nil, err
		}
		deliveryRules = append(deliveryRules, *rule)
	}

	return &deliveryPolicy, nil
}

type flattenedEndpointDeliveryPolicies struct {
	globalDeliveryRules []interface{}
	deliveryRules       []interface{}
}

func flattenEndpointDeliveryPolicy(input *cdn.EndpointPropertiesUpdateParametersDeliveryPolicy) (*flattenedEndpointDeliveryPolicies, error) {
	output := flattenedEndpointDeliveryPolicies{
		globalDeliveryRules: make([]interface{}, 0),
		deliveryRules:       make([]interface{}, 0),
	}
	if input == nil || input.Rules == nil {
		return &output, nil
	}

	for _, rule := range *input.Rules {
		if rule.Order == nil {
			continue
		}

		if int(*rule.Order) == 0 {
			flattenedRule, err := flattenArmCdnEndpointGlobalDeliveryRule(rule)
			if err != nil {
				return nil, err
			}

			output.globalDeliveryRules = append(output.globalDeliveryRules, flattenedRule)
			continue
		}

		flattenedRule, err := flattenArmCdnEndpointDeliveryRule(rule)
		if err != nil {
			return nil, err
		}

		output.deliveryRules = append(output.deliveryRules, flattenedRule)
	}

	return &output, nil
}

func validateAzureRmCdnEndpointOriginsInvalidProperties(origin map[string]interface{}, id parse.EndpointId) error {
	var invalidProps []string
	var propsValue []string
	if v, ok := origin["priority"]; ok && v.(int) != 0 {
		invalidProps = append(invalidProps, "priority")
		propsValue = append(propsValue, strconv.Itoa(v.(int)))
	}

	if v, ok := origin["weight"]; ok && v.(int) != 0 {
		invalidProps = append(invalidProps, "weight")
		propsValue = append(propsValue, strconv.Itoa(v.(int)))
	}

	if v, ok := origin["origin_host_header"]; ok && v.(string) != "" {
		invalidProps = append(invalidProps, "origin_host_header")
		propsValue = append(propsValue, fmt.Sprintf("%q", v.(string)))
	}

	if len(invalidProps) > 0 {
		errTxt := ""
		valueTxt := ""
		switch len(invalidProps) {
		case 1:
			errTxt = fmt.Sprintf("`%s`", invalidProps[0])
			valueTxt = fmt.Sprintf(", got (`%s = %s`)", invalidProps[0], propsValue[0])
		case 2:
			errTxt = fmt.Sprintf("`%s` and `%s`", invalidProps[0], invalidProps[1])
			valueTxt = fmt.Sprintf(", got (`%s = %s` and `%s = %s`)", invalidProps[0], propsValue[0], invalidProps[1], propsValue[1])
		case 3:
			errTxt = fmt.Sprintf("`%s`, `%s` and `%s`", invalidProps[0], invalidProps[1], invalidProps[2])
			valueTxt = fmt.Sprintf(", got (`%s = %s`, `%s = %s` and `%s = %s`)", invalidProps[0], propsValue[0], invalidProps[1], propsValue[1], invalidProps[2], propsValue[2])
		}

		return fmt.Errorf("%s: %s cannot be set for single origin endpoints, %s are only supported for multi-origin endpoints%s", id, errTxt, errTxt, valueTxt)
	}

	return nil
}
