package opensearch

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

	opensearchUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/opensearch/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch/wait"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
)

type Model struct {
	Id                 types.String `tfsdk:"id"` // needed by TF
	InstanceId         types.String `tfsdk:"instance_id"`
	ProjectId          types.String `tfsdk:"project_id"`
	CfGuid             types.String `tfsdk:"cf_guid"`
	CfSpaceGuid        types.String `tfsdk:"cf_space_guid"`
	DashboardUrl       types.String `tfsdk:"dashboard_url"`
	ImageUrl           types.String `tfsdk:"image_url"`
	Name               types.String `tfsdk:"name"`
	CfOrganizationGuid types.String `tfsdk:"cf_organization_guid"`
	Parameters         types.Object `tfsdk:"parameters"`
	Version            types.String `tfsdk:"version"`
	PlanName           types.String `tfsdk:"plan_name"`
	PlanId             types.String `tfsdk:"plan_id"`
}

// Struct corresponding to DataSourceModel.Parameters
type parametersModel struct {
	SgwAcl               types.String `tfsdk:"sgw_acl"`
	EnableMonitoring     types.Bool   `tfsdk:"enable_monitoring"`
	Graphite             types.String `tfsdk:"graphite"`
	JavaGarbageCollector types.String `tfsdk:"java_garbage_collector"`
	JavaHeapspace        types.Int64  `tfsdk:"java_heapspace"`
	JavaMaxmetaspace     types.Int64  `tfsdk:"java_maxmetaspace"`
	MaxDiskThreshold     types.Int64  `tfsdk:"max_disk_threshold"`
	MetricsFrequency     types.Int64  `tfsdk:"metrics_frequency"`
	MetricsPrefix        types.String `tfsdk:"metrics_prefix"`
	MonitoringInstanceId types.String `tfsdk:"monitoring_instance_id"`
	Plugins              types.List   `tfsdk:"plugins"`
	Syslog               types.List   `tfsdk:"syslog"`
	TlsCiphers           types.List   `tfsdk:"tls_ciphers"`
	TlsProtocols         types.List   `tfsdk:"tls_protocols"`
}

// Types corresponding to parametersModel
var parametersTypes = map[string]attr.Type{
	"sgw_acl":                basetypes.StringType{},
	"enable_monitoring":      basetypes.BoolType{},
	"graphite":               basetypes.StringType{},
	"java_garbage_collector": basetypes.StringType{},
	"java_heapspace":         basetypes.Int64Type{},
	"java_maxmetaspace":      basetypes.Int64Type{},
	"max_disk_threshold":     basetypes.Int64Type{},
	"metrics_frequency":      basetypes.Int64Type{},
	"metrics_prefix":         basetypes.StringType{},
	"monitoring_instance_id": basetypes.StringType{},
	"plugins":                basetypes.ListType{ElemType: types.StringType},
	"syslog":                 basetypes.ListType{ElemType: types.StringType},
	"tls_ciphers":            basetypes.ListType{ElemType: types.StringType},
	"tls_protocols":          basetypes.ListType{ElemType: types.StringType},
}

// NewInstanceResource is a helper function to simplify the provider implementation.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource is the resource implementation.
type instanceResource struct {
	client *opensearch.APIClient
}

// Metadata returns the resource type name.
func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_opensearch_instance"
}

// Configure adds the provider configured client to the resource.
func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := opensearchUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = apiClient
	tflog.Info(ctx, "OpenSearch instance client configured")
}

// Schema defines the schema for the resource.
func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":        "OpenSearch instance resource schema. Must have a `region` specified in the provider configuration.",
		"id":          "Terraform's internal resource ID. It is structured as \"`project_id`,`instance_id`\".",
		"instance_id": "ID of the OpenSearch instance.",
		"project_id":  "STACKIT project ID to which the instance is associated.",
		"name":        "Instance name.",
		"version":     "The service version.",
		"plan_name":   "The selected plan name.",
		"plan_id":     "The selected plan ID.",
		"parameters":  "Configuration parameters. Please note that removing a previously configured field from your Terraform configuration won't replace its value in the API. To update a previously configured field, explicitly set a new value for it.",
	}

	parametersDescriptions := map[string]string{
		"sgw_acl":                "Comma separated list of IP networks in CIDR notation which are allowed to access this instance.",
		"enable_monitoring":      "Enable monitoring.",
		"graphite":               "If set, monitoring with Graphite will be enabled. Expects the host and port where the Graphite metrics should be sent to (host:port).",
		"max_disk_threshold":     "The maximum disk threshold in MB. If the disk usage exceeds this threshold, the instance will be stopped.",
		"metrics_frequency":      "The frequency in seconds at which metrics are emitted (in seconds).",
		"metrics_prefix":         "The prefix for the metrics. Could be useful when using Graphite monitoring to prefix the metrics with a certain value, like an API key.",
		"monitoring_instance_id": "The ID of the STACKIT monitoring instance.",
		"java_garbage_collector": "The garbage collector to use for OpenSearch.",
		"java_heapspace":         "The amount of memory (in MB) allocated as heap by the JVM for OpenSearch.",
		"java_maxmetaspace":      "The amount of memory (in MB) used by the JVM to store metadata for OpenSearch.",
		"plugins":                "List of plugins to install. Must be a supported plugin name. The plugins `repository-s3` and `repository-azure` are enabled by default and cannot be disabled.",
		"syslog":                 "List of syslog servers to send logs to.",
		"tls_ciphers":            "List of TLS ciphers to use.",
		"tls_protocols":          "The TLS protocol to use.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description: descriptions["instance_id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"version": schema.StringAttribute{
				Description: descriptions["version"],
				Required:    true,
			},
			"plan_name": schema.StringAttribute{
				Description: descriptions["plan_name"],
				Required:    true,
			},
			"plan_id": schema.StringAttribute{
				Description: descriptions["plan_id"],
				Computed:    true,
			},
			"parameters": schema.SingleNestedAttribute{
				Description: descriptions["parameters"],
				Attributes: map[string]schema.Attribute{
					"sgw_acl": schema.StringAttribute{
						Description: parametersDescriptions["sgw_acl"],
						Optional:    true,
						Computed:    true,
					},
					"enable_monitoring": schema.BoolAttribute{
						Description: parametersDescriptions["enable_monitoring"],
						Optional:    true,
						Computed:    true,
					},
					"graphite": schema.StringAttribute{
						Description: parametersDescriptions["graphite"],
						Optional:    true,
						Computed:    true,
					},
					"java_garbage_collector": schema.StringAttribute{
						Description: parametersDescriptions["java_garbage_collector"],
						Optional:    true,
						Computed:    true,
					},
					"java_heapspace": schema.Int64Attribute{
						Description: parametersDescriptions["java_heapspace"],
						Optional:    true,
						Computed:    true,
					},
					"java_maxmetaspace": schema.Int64Attribute{
						Description: parametersDescriptions["java_maxmetaspace"],
						Optional:    true,
						Computed:    true,
					},
					"max_disk_threshold": schema.Int64Attribute{
						Description: parametersDescriptions["max_disk_threshold"],
						Optional:    true,
						Computed:    true,
					},
					"metrics_frequency": schema.Int64Attribute{
						Description: parametersDescriptions["metrics_frequency"],
						Optional:    true,
						Computed:    true,
					},
					"metrics_prefix": schema.StringAttribute{
						Description: parametersDescriptions["metrics_prefix"],
						Optional:    true,
						Computed:    true,
					},
					"monitoring_instance_id": schema.StringAttribute{
						Description: parametersDescriptions["monitoring_instance_id"],
						Optional:    true,
						Computed:    true,
					},
					"plugins": schema.ListAttribute{
						ElementType: types.StringType,
						Description: parametersDescriptions["plugins"],
						Optional:    true,
						Computed:    true,
					},
					"syslog": schema.ListAttribute{
						ElementType: types.StringType,
						Description: parametersDescriptions["syslog"],
						Optional:    true,
						Computed:    true,
					},
					"tls_ciphers": schema.ListAttribute{
						ElementType: types.StringType,
						Description: parametersDescriptions["tls_ciphers"],
						Optional:    true,
						Computed:    true,
					},
					"tls_protocols": schema.ListAttribute{
						ElementType: types.StringType,
						Description: parametersDescriptions["tls_protocols"],
						Optional:    true,
						Computed:    true,
					},
				},
				Optional: true,
				Computed: true,
			},
			"cf_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cf_space_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"image_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cf_organization_guid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	var parameters *parametersModel
	if !(model.Parameters.IsNull() || model.Parameters.IsUnknown()) {
		parameters = &parametersModel{}
		diags = model.Parameters.As(ctx, parameters, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err := r.loadPlanId(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Loading service plan: %v", err))
		return
	}

	// Generate API request body from model
	payload, err := toCreatePayload(&model, parameters)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Create new instance
	createResp, err := r.client.CreateInstance(ctx, projectId).CreateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	instanceId := *createResp.InstanceId
	ctx = tflog.SetField(ctx, "instance_id", instanceId)
	waitResp, err := wait.CreateInstanceWaitHandler(ctx, r.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Instance creation waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "OpenSearch instance created")
}

// Read refreshes the Terraform state with the latest data.
func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	instanceResp, err := r.client.GetInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && (oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusGone) {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(instanceResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Compute and store values not present in the API response
	err = loadPlanNameAndVersion(ctx, r.client, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading instance", fmt.Sprintf("Loading service plan details: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "OpenSearch instance read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	var parameters *parametersModel
	if !(model.Parameters.IsNull() || model.Parameters.IsUnknown()) {
		parameters = &parametersModel{}
		diags = model.Parameters.As(ctx, parameters, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	err := r.loadPlanId(ctx, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Loading service plan: %v", err))
		return
	}

	// Generate API request body from model
	payload, err := toUpdatePayload(&model, parameters)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	// Update existing instance
	err = r.client.PartialUpdateInstance(ctx, projectId, instanceId).PartialUpdateInstancePayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	waitResp, err := wait.PartialUpdateInstanceWaitHandler(ctx, r.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Instance update waiting: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(waitResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating instance", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "OpenSearch instance updated")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve values from state
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	instanceId := model.InstanceId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "instance_id", instanceId)

	// Delete existing instance
	err := r.client.DeleteInstance(ctx, projectId, instanceId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Calling API: %v", err))
		return
	}
	_, err = wait.DeleteInstanceWaitHandler(ctx, r.client, projectId, instanceId).WaitWithContext(ctx)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting instance", fmt.Sprintf("Instance deletion waiting: %v", err))
		return
	}
	tflog.Info(ctx, "OpenSearch instance deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the resource import identifier is: project_id,instance_id
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing instance",
			fmt.Sprintf("Expected import identifier with format: [project_id],[instance_id]  Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[1])...)
	tflog.Info(ctx, "OpenSearch instance state imported")
}

func mapFields(instance *opensearch.Instance, model *Model) error {
	if instance == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	var instanceId string
	if model.InstanceId.ValueString() != "" {
		instanceId = model.InstanceId.ValueString()
	} else if instance.InstanceId != nil {
		instanceId = *instance.InstanceId
	} else {
		return fmt.Errorf("instance id not present")
	}

	model.Id = utils.BuildInternalTerraformId(model.ProjectId.ValueString(), instanceId)
	model.InstanceId = types.StringValue(instanceId)
	model.PlanId = types.StringPointerValue(instance.PlanId)
	model.CfGuid = types.StringPointerValue(instance.CfGuid)
	model.CfSpaceGuid = types.StringPointerValue(instance.CfSpaceGuid)
	model.DashboardUrl = types.StringPointerValue(instance.DashboardUrl)
	model.ImageUrl = types.StringPointerValue(instance.ImageUrl)
	model.Name = types.StringPointerValue(instance.Name)
	model.CfOrganizationGuid = types.StringPointerValue(instance.CfOrganizationGuid)

	if instance.Parameters == nil {
		model.Parameters = types.ObjectNull(parametersTypes)
	} else {
		parameters, err := mapParameters(*instance.Parameters)
		if err != nil {
			return fmt.Errorf("mapping parameters: %w", err)
		}
		model.Parameters = parameters
	}
	return nil
}

func mapParameters(params map[string]interface{}) (types.Object, error) {
	attributes := map[string]attr.Value{}
	for attribute := range parametersTypes {
		var valueInterface interface{}
		var ok bool

		// This replacement is necessary because Terraform does not allow hyphens in attribute names
		// And the API uses hyphens in some of the attribute names, which would cause a mismatch
		// The following attributes have hyphens in the API but underscores in the schema
		hyphenAttributes := []string{
			"tls_ciphers",
			"tls_protocols",
		}
		if slices.Contains(hyphenAttributes, attribute) {
			alteredAttribute := strings.ReplaceAll(attribute, "_", "-")
			valueInterface, ok = params[alteredAttribute]
		} else {
			valueInterface, ok = params[attribute]
		}
		if !ok {
			// All fields are optional, so this is ok
			// Set the value as nil, will be handled accordingly
			valueInterface = nil
		}

		var value attr.Value
		switch parametersTypes[attribute].(type) {
		default:
			return types.ObjectNull(parametersTypes), fmt.Errorf("found unexpected attribute type '%T'", parametersTypes[attribute])
		case basetypes.StringType:
			if valueInterface == nil {
				value = types.StringNull()
			} else {
				valueString, ok := valueInterface.(string)
				if !ok {
					return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' of type %T, failed to assert as string", attribute, valueInterface)
				}
				value = types.StringValue(valueString)
			}
		case basetypes.BoolType:
			if valueInterface == nil {
				value = types.BoolNull()
			} else {
				valueBool, ok := valueInterface.(bool)
				if !ok {
					return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' of type %T, failed to assert as bool", attribute, valueInterface)
				}
				value = types.BoolValue(valueBool)
			}
		case basetypes.Int64Type:
			if valueInterface == nil {
				value = types.Int64Null()
			} else {
				// This may be int64, int32, int or float64
				// We try to assert all 4
				var valueInt64 int64
				switch temp := valueInterface.(type) {
				default:
					return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' of type %T, failed to assert as int", attribute, valueInterface)
				case int64:
					valueInt64 = temp
				case int32:
					valueInt64 = int64(temp)
				case int:
					valueInt64 = int64(temp)
				case float64:
					valueInt64 = int64(temp)
				}
				value = types.Int64Value(valueInt64)
			}
		case basetypes.ListType: // Assumed to be a list of strings
			if valueInterface == nil {
				value = types.ListNull(types.StringType)
			} else {
				// This may be []string{} or []interface{}
				// We try to assert all 2
				var valueList []attr.Value
				switch temp := valueInterface.(type) {
				default:
					return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' of type %T, failed to assert as array of interface", attribute, valueInterface)
				case []string:
					for _, x := range temp {
						valueList = append(valueList, types.StringValue(x))
					}
				case []interface{}:
					for _, x := range temp {
						xString, ok := x.(string)
						if !ok {
							return types.ObjectNull(parametersTypes), fmt.Errorf("found attribute '%s' with element '%s' of type %T, failed to assert as string", attribute, x, x)
						}
						valueList = append(valueList, types.StringValue(xString))
					}
				}
				temp2, diags := types.ListValue(types.StringType, valueList)
				if diags.HasError() {
					return types.ObjectNull(parametersTypes), fmt.Errorf("failed to map %s: %w", attribute, core.DiagsToError(diags))
				}
				value = temp2
			}
		}
		attributes[attribute] = value
	}

	output, diags := types.ObjectValue(parametersTypes, attributes)
	if diags.HasError() {
		return types.ObjectNull(parametersTypes), fmt.Errorf("failed to create object: %w", core.DiagsToError(diags))
	}
	return output, nil
}

func toCreatePayload(model *Model, parameters *parametersModel) (*opensearch.CreateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	payloadParams, err := toInstanceParams(parameters)
	if err != nil {
		return nil, fmt.Errorf("convert parameters: %w", err)
	}
	return &opensearch.CreateInstancePayload{
		InstanceName: conversion.StringValueToPointer(model.Name),
		Parameters:   payloadParams,
		PlanId:       conversion.StringValueToPointer(model.PlanId),
	}, nil
}

func toUpdatePayload(model *Model, parameters *parametersModel) (*opensearch.PartialUpdateInstancePayload, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	payloadParams, err := toInstanceParams(parameters)
	if err != nil {
		return nil, fmt.Errorf("convert parameters: %w", err)
	}
	return &opensearch.PartialUpdateInstancePayload{
		Parameters: payloadParams,
		PlanId:     conversion.StringValueToPointer(model.PlanId),
	}, nil
}

func toInstanceParams(parameters *parametersModel) (*opensearch.InstanceParameters, error) {
	if parameters == nil {
		return nil, nil
	}
	payloadParams := &opensearch.InstanceParameters{}

	payloadParams.SgwAcl = conversion.StringValueToPointer(parameters.SgwAcl)
	payloadParams.EnableMonitoring = conversion.BoolValueToPointer(parameters.EnableMonitoring)
	payloadParams.Graphite = conversion.StringValueToPointer(parameters.Graphite)
	payloadParams.JavaGarbageCollector = opensearch.InstanceParametersGetJavaGarbageCollectorAttributeType(conversion.StringValueToPointer(parameters.JavaGarbageCollector))
	payloadParams.JavaHeapspace = conversion.Int64ValueToPointer(parameters.JavaHeapspace)
	payloadParams.JavaMaxmetaspace = conversion.Int64ValueToPointer(parameters.JavaMaxmetaspace)
	payloadParams.MaxDiskThreshold = conversion.Int64ValueToPointer(parameters.MaxDiskThreshold)
	payloadParams.MetricsFrequency = conversion.Int64ValueToPointer(parameters.MetricsFrequency)
	payloadParams.MetricsPrefix = conversion.StringValueToPointer(parameters.MetricsPrefix)
	payloadParams.MonitoringInstanceId = conversion.StringValueToPointer(parameters.MonitoringInstanceId)

	var err error
	payloadParams.Plugins, err = conversion.StringListToPointer(parameters.Plugins)
	if err != nil {
		return nil, fmt.Errorf("convert plugins: %w", err)
	}

	payloadParams.Syslog, err = conversion.StringListToPointer(parameters.Syslog)
	if err != nil {
		return nil, fmt.Errorf("convert syslog: %w", err)
	}

	payloadParams.TlsCiphers, err = conversion.StringListToPointer(parameters.TlsCiphers)
	if err != nil {
		return nil, fmt.Errorf("convert tls_ciphers: %w", err)
	}

	payloadParams.TlsProtocols, err = conversion.StringListToPointer(parameters.TlsProtocols)
	if err != nil {
		return nil, fmt.Errorf("convert tls_protocols: %w", err)
	}

	return payloadParams, nil
}

func (r *instanceResource) loadPlanId(ctx context.Context, model *Model) error {
	projectId := model.ProjectId.ValueString()
	res, err := r.client.ListOfferings(ctx, projectId).Execute()
	if err != nil {
		return fmt.Errorf("getting OpenSearch offerings: %w", err)
	}

	version := model.Version.ValueString()
	planName := model.PlanName.ValueString()
	availableVersions := ""
	availablePlanNames := ""
	isValidVersion := false
	for _, offer := range *res.Offerings {
		if !strings.EqualFold(*offer.Version, version) {
			availableVersions = fmt.Sprintf("%s\n- %s", availableVersions, *offer.Version)
			continue
		}
		isValidVersion = true

		for _, plan := range *offer.Plans {
			if plan.Name == nil {
				continue
			}
			if strings.EqualFold(*plan.Name, planName) && plan.Id != nil {
				model.PlanId = types.StringPointerValue(plan.Id)
				return nil
			}
			availablePlanNames = fmt.Sprintf("%s\n- %s", availablePlanNames, *plan.Name)
		}
	}

	if !isValidVersion {
		return fmt.Errorf("couldn't find version '%s', available versions are: %s", version, availableVersions)
	}
	return fmt.Errorf("couldn't find plan_name '%s' for version %s, available names are: %s", planName, version, availablePlanNames)
}

func loadPlanNameAndVersion(ctx context.Context, client *opensearch.APIClient, model *Model) error {
	projectId := model.ProjectId.ValueString()
	planId := model.PlanId.ValueString()
	res, err := client.ListOfferings(ctx, projectId).Execute()
	if err != nil {
		return fmt.Errorf("getting OpenSearch offerings: %w", err)
	}

	for _, offer := range *res.Offerings {
		for _, plan := range *offer.Plans {
			if strings.EqualFold(*plan.Id, planId) && plan.Id != nil {
				model.PlanName = types.StringPointerValue(plan.Name)
				model.Version = types.StringPointerValue(offer.Version)
				return nil
			}
		}
	}

	return fmt.Errorf("couldn't find plan_name and version for plan_id '%s'", planId)
}
