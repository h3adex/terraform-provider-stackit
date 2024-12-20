---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_security_group Resource - stackit"
subcategory: ""
description: |-
  Security group resource schema. Must have a region specified in the provider configuration.
  ~> This resource is in beta and may be subject to breaking changes in the future. Use with caution. See our guide https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/opting_into_beta_resources for how to opt-in to use beta resources.
---

# stackit_security_group (Resource)

Security group resource schema. Must have a `region` specified in the provider configuration.

~> This resource is in beta and may be subject to breaking changes in the future. Use with caution. See our [guide](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/opting_into_beta_resources) for how to opt-in to use beta resources.

## Example Usage

```terraform
resource "stackit_security_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "my_security_group"
  labels = {
    "key" = "value"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the security group.
- `project_id` (String) STACKIT project ID to which the security group is associated.

### Optional

- `description` (String) The description of the security group.
- `labels` (Map of String) Labels are key-value string pairs which can be attached to a resource container
- `stateful` (Boolean) Configures if a security group is stateful or stateless. There can only be one type of security groups per network interface/server.

### Read-Only

- `id` (String) Terraform's internal resource ID. It is structured as "`project_id`,`security_group_id`".
- `security_group_id` (String) The security group ID.
