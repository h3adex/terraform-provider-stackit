---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_volume Resource - stackit"
subcategory: ""
description: |-
  Volume resource schema. Must have a region specified in the provider configuration.
---

# stackit_volume (Resource)

Volume resource schema. Must have a `region` specified in the provider configuration.

## Example Usage

```terraform
resource "stackit_volume" "example" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name              = "my_volume"
  availability_zone = "eu01-1"
  size              = 64
  labels = {
    "key" = "value"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `availability_zone` (String) The availability zone of the volume.
- `project_id` (String) STACKIT project ID to which the volume is associated.

### Optional

- `description` (String) The description of the volume.
- `labels` (Map of String) Labels are key-value string pairs which can be attached to a resource container
- `name` (String) The name of the volume.
- `performance_class` (String) The performance class of the volume. Possible values are documented in [Service plans BlockStorage](https://docs.stackit.cloud/stackit/en/service-plans-blockstorage-75137974.html#ServiceplansBlockStorage-CurrentlyavailableServicePlans%28performanceclasses%29)
- `size` (Number) The size of the volume in GB. It can only be updated to a larger value than the current size. Either `size` or `source` must be provided
- `source` (Attributes) The source of the volume. It can be either a volume, an image, a snapshot or a backup. Either `size` or `source` must be provided (see [below for nested schema](#nestedatt--source))

### Read-Only

- `id` (String) Terraform's internal resource ID. It is structured as "`project_id`,`volume_id`".
- `server_id` (String) The server ID of the server to which the volume is attached to.
- `volume_id` (String) The volume ID.

<a id="nestedatt--source"></a>
### Nested Schema for `source`

Required:

- `id` (String) The ID of the source, e.g. image ID
- `type` (String) The type of the source. Supported values are: `volume`, `image`, `snapshot`, `backup`.
