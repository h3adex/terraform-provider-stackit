package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/mariadb"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *mariadb.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.MariaDBCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.MariaDBCustomEndpoint))
	} else {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithRegion(providerData.GetRegion()))
	}
	apiClient, err := mariadb.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}
