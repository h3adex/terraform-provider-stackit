---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_objectstorage_credentials_group Resource - stackit"
subcategory: ""
description: |-
  ObjectStorage credentials group resource schema. Must have a region specified in the provider configuration. If you are creating credentialsgroup and bucket resources simultaneously, please include the depends_on field so that they are created sequentially. This prevents errors from concurrent calls to the service enablement that is done in the background.
---

# stackit_objectstorage_credentials_group (Resource)

ObjectStorage credentials group resource schema. Must have a `region` specified in the provider configuration. If you are creating `credentialsgroup` and `bucket` resources simultaneously, please include the `depends_on` field so that they are created sequentially. This prevents errors from concurrent calls to the service enablement that is done in the background.

## Example Usage

```terraform
resource "stackit_objectstorage_credentials_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-credentials-group"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The credentials group's display name.
- `project_id` (String) Project ID to which the credentials group is associated.

### Optional

- `region` (String) The resource region. If not defined, the provider region is used.

### Read-Only

- `credentials_group_id` (String) The credentials group ID
- `id` (String) Terraform's internal data source identifier. It is structured as "`project_id`,`credentials_group_id`".
- `urn` (String) Credentials group uniform resource name (URN)
