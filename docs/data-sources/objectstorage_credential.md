---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_objectstorage_credential Data Source - stackit"
subcategory: ""
description: |-
  ObjectStorage credential data source schema. Must have a region specified in the provider configuration.
---

# stackit_objectstorage_credential (Data Source)

ObjectStorage credential data source schema. Must have a `region` specified in the provider configuration.

## Example Usage

```terraform
data "stackit_objectstorage_credential" "example" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  credentials_group_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  credential_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `credential_id` (String) The credential ID.
- `credentials_group_id` (String) The credential group ID.
- `project_id` (String) STACKIT Project ID to which the credential group is associated.

### Optional

- `region` (String) The resource region. If not defined, the provider region is used.

### Read-Only

- `expiration_timestamp` (String)
- `id` (String) Terraform's internal resource identifier. It is structured as "`project_id`,`credentials_group_id`,`credential_id`".
- `name` (String)
