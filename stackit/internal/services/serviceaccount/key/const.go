package key

const markdownDescription = `
Schema for a STACKIT service account access token resource.` + "\n" + `
~> This resource is in beta and may be subject to breaking changes in the future. Use with caution. See our [guide](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs/guides/opting_into_beta_resources) for how to opt-in to use beta resources.
## Example Usage` + "\n" + `

### Automatically rotate access tokens` + "\n" +
	"```terraform" + `
resource "stackit_service_account" "sa" {
  project_id = var.stackit_project_id
  name = "sa01"
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

//TODO: add Code
` + "\n```"
