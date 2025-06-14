package publicipranges

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &publicIpRangesDataSource{}
)

// NewPublicIpRangesDataSource is a helper function to simplify the provider implementation.
func NewPublicIpRangesDataSource() datasource.DataSource {
	return &publicIpRangesDataSource{}
}

// publicIpRangesDataSource is the data source implementation.
type publicIpRangesDataSource struct {
	client *iaas.APIClient
}

type Model struct {
	Id             types.String `tfsdk:"id"` // needed by TF
	PublicIpRanges types.List   `tfsdk:"public_ip_ranges"`
}

var publicIpRangesTypes = map[string]attr.Type{
	"cidr": types.StringType,
}

// Metadata returns the data source type name.
func (d *publicIpRangesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ip_ranges"
}

func (d *publicIpRangesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "iaas client configured")
}

// Schema defines the schema for the resource.
func (d *publicIpRangesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "A list of all public IP ranges that STACKIT uses."

	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It takes the values of \"`public_ip_ranges.*.cidr`\".",
				Computed:    true,
				Optional:    false,
			},
			"public_ip_ranges": schema.ListNestedAttribute{
				Description: "A list of all public IP ranges.",
				Computed:    true,
				Optional:    false,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						validate.CIDR(),
					),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cidr": schema.StringAttribute{
							Description: "Classless Inter-Domain Routing (CIDR)",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *publicIpRangesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	publicIpRangeResp, err := d.client.ListPublicIPRangesExecute(ctx)
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading public ip ranges",
			"Public ip ranges cannot be found",
			map[int]string{
				http.StatusForbidden: "Forbidden access",
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to schema
	err = mapFields(ctx, publicIpRangeResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading public IP ranges", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "read public IP ranges")
}

func mapFields(ctx context.Context, publicIpRangeResp *iaas.PublicNetworkListResponse, model *Model) error {
	if publicIpRangeResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	err := mapPublicIpRanges(ctx, publicIpRangeResp.Items, model)
	if err != nil {
		return fmt.Errorf("error mapping public IP ranges: %w", err)
	}
	return nil
}

// mapPublicIpRanges map the response publicIpRanges to the model
func mapPublicIpRanges(_ context.Context, publicIpRanges *[]iaas.PublicNetwork, model *Model) error {
	if publicIpRanges == nil {
		return fmt.Errorf("publicIpRanges input is nil")
	}
	if len(*publicIpRanges) == 0 {
		model.PublicIpRanges = types.ListNull(types.ObjectType{AttrTypes: publicIpRangesTypes})
		return nil
	}

	var apiIpRanges []string
	for _, ipRange := range *publicIpRanges {
		if ipRange.Cidr != nil || *ipRange.Cidr != "" {
			apiIpRanges = append(apiIpRanges, *ipRange.Cidr)
		}
	}

	// Sort to prevent unnecessary recreation of dependent resources due to order changes.
	sort.Strings(apiIpRanges)

	model.Id = utils.BuildInternalTerraformId(apiIpRanges...)

	var ipRangesList []attr.Value
	for _, cidr := range apiIpRanges {
		ipRangeValues := map[string]attr.Value{
			"cidr": types.StringValue(cidr),
		}
		ipRangeObject, diag := types.ObjectValue(publicIpRangesTypes, ipRangeValues)
		if diag.HasError() {
			return core.DiagsToError(diag)
		}
		ipRangesList = append(ipRangesList, ipRangeObject)
	}

	ipRangesTF, diags := types.ListValue(
		types.ObjectType{AttrTypes: publicIpRangesTypes},
		ipRangesList,
	)
	if diags.HasError() {
		return core.DiagsToError(diags)
	}

	model.PublicIpRanges = ipRangesTF
	return nil
}
