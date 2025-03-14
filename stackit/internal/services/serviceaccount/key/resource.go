package key

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
	"time"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &serviceAccountKeyResource{}
	_ resource.ResourceWithConfigure = &serviceAccountKeyResource{}
)

// Model represents the schema for the service account key resource in Terraform.
type Model struct {
	Id                  types.String `tfsdk:"id"`
	KeyId               types.String `tfsdk:"key_id"`
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
	ProjectId           types.String `tfsdk:"project_id"`
	RotateWhenChanged   types.Map    `tfsdk:"rotate_when_changed"`
	TtlDays             types.Int64  `tfsdk:"ttl_days"`
	CreatedAt           types.String `tfsdk:"created_at"`
	ValidUntil          types.String `tfsdk:"valid_until"`
	Audience            types.String `tfsdk:"audience"`
	Issuer              types.String `tfsdk:"issuer"`
	Kid                 types.String `tfsdk:"kid"`
	PrivateKey          types.String `tfsdk:"private_key"`
	Subject             types.String `tfsdk:"subject"`
	KeyAlgorithm        types.String `tfsdk:"key_algorithm"`
	KeyOrigin           types.String `tfsdk:"key_origin"`
	KeyType             types.String `tfsdk:"key_type"`
	PublicKey           types.String `tfsdk:"public_key"`
	RawResponse         types.String `tfsdk:"raw_response"`
}

// NewServiceAccountKeyResource is a helper function to create a new service account key resource instance.
func NewServiceAccountKeyResource() resource.Resource {
	return &serviceAccountKeyResource{}
}

// serviceAccountKeyResource implements the resource interface for service account key.
type serviceAccountKeyResource struct {
	client *serviceaccount.APIClient
}

// Configure sets up the API client for the service account resource.
func (r *serviceAccountKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent potential panics if the provider is not properly configured.
	if req.ProviderData == nil {
		return
	}

	// Validate provider data type before proceeding.
	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_service_account_key", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	// Initialize the API client with the appropriate authentication and endpoint settings.
	var apiClient *serviceaccount.APIClient
	var err error
	if providerData.ServiceAccountCustomEndpoint != "" {
		apiClient, err = serviceaccount.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ServiceAccountCustomEndpoint),
		)
	} else {
		apiClient, err = serviceaccount.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	// Handle API client initialization errors.
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	// Store the initialized client.
	r.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

// Metadata sets the resource type name for the service account resource.
func (r *serviceAccountKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_key"
}

// Schema defines the resource schema for the service account access token.
func (r *serviceAccountKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"id":                    "Unique internal resource ID for Terraform, formatted as \"project_id,key_id\".",
		"project_id":            "STACKIT project ID associated with the service account token.",
		"key_id":                "Identifier for the key linked to the service account.",
		"service_account_email": "Email address linked to the service account.",
		"valid_until":           "Specifies the key's validity duration in days. If unspecified, key is valid until deleted.",
		"rotate_when_changed":   "A map of arbitrary key/value pairs that will force recreation of the token when they change, enabling token rotation based on external conditions such as a rotating timestamp. Changing this forces a new resource to be created.",
		"created_at":            "Timestamp indicating when the access token was created.",
		"audience":              "",
		"issuer":                "",
		"kid":                   "",
		"private_key":           "",
		"subject":               "",
		"key_algorithm":         "",
		"key_origin":            "",
		"key_type":              "",
		"public_key":            "",
		"raw_json":              "",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescription,
		Description:         "Schema for managing a STACKIT service account access token.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_id": schema.StringAttribute{
				Description: descriptions["key_id"],
				Computed:    true,
			},
			"service_account_email": schema.StringAttribute{
				Description: descriptions["service_account_email"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl_days": schema.Int64Attribute{
				Description: descriptions["ttl_days"],
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"rotate_when_changed": schema.MapAttribute{
				Description: descriptions["rotate_when_changed"],
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: descriptions["created_at"],
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: descriptions["valid_until"],
				Computed:    true,
			},
			"audience": schema.StringAttribute{
				Description: descriptions["audience"],
				Computed:    true,
			},
			"issuer": schema.StringAttribute{
				Description: descriptions["issuer"],
				Computed:    true,
			},
			"kid": schema.StringAttribute{
				Description: descriptions["kid"],
				Computed:    true,
			},
			"private_key": schema.StringAttribute{
				Description: descriptions["private_key"],
				Computed:    true,
				Sensitive:   true,
			},
			"subject": schema.StringAttribute{
				Description: descriptions["subject"],
				Computed:    true,
			},
			"key_algorithm": schema.StringAttribute{
				Description: descriptions["key_algorithm"],
				Computed:    true,
			},
			"key_origin": schema.StringAttribute{
				Description: descriptions["key_origin"],
				Computed:    true,
			},
			"key_type": schema.StringAttribute{
				Description: descriptions["key_type"],
				Computed:    true,
			},
			"public_key": schema.StringAttribute{
				Description: descriptions["public_key"],
				Computed:    true,
			},
			"raw_response": schema.StringAttribute{
				Description: descriptions["raw_response"],
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state for service accounts.
func (r *serviceAccountKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set logging context with the project ID and service account email.
	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)

	if model.TtlDays.IsUnknown() {
		model.TtlDays = types.Int64Null()
	}

	// Generate the API request payload.
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account access token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Initialize the API request with the required parameters.
	serviceAccountAccessTokenResp, err := r.client.CreateServiceAccountKey(ctx, projectId, serviceAccountEmail).CreateServiceAccountKeyPayload(*payload).Execute()

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Failed to create service account access token", fmt.Sprintf("API call error: %v", err))
		return
	}

	// Map the response to the resource schema.
	err = mapResponse(serviceAccountAccessTokenResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account access token", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Service account access token created")
}

// Read refreshes the Terraform state with the latest service account data.
func (r *serviceAccountKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	/*var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	keyId := model.KeyId.ValueString()*/

	/*saKeyResp, err := r.client.GetServiceAccountKey(ctx, projectId, serviceAccountEmail, keyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account key", fmt.Sprintf("Error calling API: %v", err))
		return
	}*/

}

// Update attempts to update the resource. In this case, service account token cannot be updated.
// Note: This method is intentionally left without update logic because changes
// to 'project_id', 'service_account_email' or 'ttl_days' require the resource to be entirely replaced.
// As a result, the Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (r *serviceAccountKeyResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Service accounts cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating service account access token", "Service accounts can't be updated")
}

// Delete deletes the service account and removes it from the Terraform state on success.
func (r *serviceAccountKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve current state of the resource.
}

func toCreatePayload(model *Model) (*serviceaccount.CreateServiceAccountKeyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	// Prepare the payload
	payload := &serviceaccount.CreateServiceAccountKeyPayload{}

	// Set ValidUntil based on TtlDays if specified
	if !model.TtlDays.IsNull() {
		validUntil, err := computeValidUntil(model.TtlDays.ValueInt64Pointer())
		if err != nil {
			return nil, err
		}
		payload.ValidUntil = &validUntil
	}

	// Set PublicKey if specified
	if model.PublicKey.String() != "" {
		payload.PublicKey = conversion.StringValueToPointer(model.PublicKey)
	}

	return payload, nil
}

// Helper function to compute ValidUntil timestamp
func computeValidUntil(ttlDays *int64) (time.Time, error) {
	if ttlDays == nil {
		// Return zero value of time.Time if ttlDays is nil
		return time.Time{}, fmt.Errorf("ttlDays is nil")
	}

	// Add the number of days to the current time in UTC
	daysDuration := time.Duration(*ttlDays) * 24 * time.Hour
	return time.Now().UTC().Add(daysDuration), nil
}

func mapResponse(resp *serviceaccount.CreateServiceAccountKeyResponse, model *Model) error {
	if resp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	if resp.Credentials.PrivateKey == nil {
		return fmt.Errorf("service account private key not present")
	}

	if resp.Id == nil {
		return fmt.Errorf("service account key id not present")
	}

	var createdAt basetypes.StringValue
	if resp.CreatedAt != nil {
		createdAtValue := *resp.CreatedAt
		createdAt = types.StringValue(createdAtValue.Format(time.RFC3339))
	}

	var validUntil basetypes.StringValue
	if resp.ValidUntil != nil {
		validUntilValue := *resp.ValidUntil
		validUntil = types.StringValue(validUntilValue.Format(time.RFC3339))
	}

	mapString := func(s *string) types.String {
		if s != nil {
			return types.StringValue(*s)
		}
		return types.StringNull()
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("error marshaling to JSON %s", err)
	}
	jsonString := string(jsonData)

	idParts := []string{model.ProjectId.ValueString(), *resp.Id}
	model.Id = types.StringValue(strings.Join(idParts, core.Separator))
	model.KeyId = mapString(resp.Id)
	model.CreatedAt = createdAt
	model.ValidUntil = validUntil
	model.KeyAlgorithm = mapString(resp.KeyAlgorithm)
	model.KeyOrigin = mapString(resp.KeyOrigin)
	model.KeyType = mapString(resp.KeyType)
	model.Audience = mapString(resp.Credentials.Aud)
	model.Issuer = mapString(resp.Credentials.Iss)
	model.Kid = mapString(resp.Credentials.Kid)
	model.PrivateKey = mapString(resp.Credentials.PrivateKey)
	model.PublicKey = mapString(resp.PublicKey)
	model.Subject = mapString(resp.Credentials.Sub)
	model.RawResponse = types.StringValue(jsonString)

	return nil
}
