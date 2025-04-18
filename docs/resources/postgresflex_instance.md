---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stackit_postgresflex_instance Resource - stackit"
subcategory: ""
description: |-
  Postgres Flex instance resource schema. Must have a region specified in the provider configuration.
---

# stackit_postgresflex_instance (Resource)

Postgres Flex instance resource schema. Must have a `region` specified in the provider configuration.

## Example Usage

```terraform
resource "stackit_postgresflex_instance" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-instance"
  acl             = ["XXX.XXX.XXX.X/XX", "XX.XXX.XX.X/XX"]
  backup_schedule = "00 00 * * *"
  flavor = {
    cpu = 2
    ram = 4
  }
  replicas = 3
  storage = {
    class = "class"
    size  = 5
  }
  version = 14
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `acl` (List of String) The Access Control List (ACL) for the PostgresFlex instance.
- `backup_schedule` (String)
- `flavor` (Attributes) (see [below for nested schema](#nestedatt--flavor))
- `name` (String) Instance name.
- `project_id` (String) STACKIT project ID to which the instance is associated.
- `replicas` (Number)
- `storage` (Attributes) (see [below for nested schema](#nestedatt--storage))
- `version` (String)

### Optional

- `region` (String) The resource region. If not defined, the provider region is used.

### Read-Only

- `id` (String) Terraform's internal resource ID. It is structured as "`project_id`,`region`,`instance_id`".
- `instance_id` (String) ID of the PostgresFlex instance.

<a id="nestedatt--flavor"></a>
### Nested Schema for `flavor`

Required:

- `cpu` (Number)
- `ram` (Number)

Read-Only:

- `description` (String)
- `id` (String)


<a id="nestedatt--storage"></a>
### Nested Schema for `storage`

Required:

- `class` (String)
- `size` (Number)
