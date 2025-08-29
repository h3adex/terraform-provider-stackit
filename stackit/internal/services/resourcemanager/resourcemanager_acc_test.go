package resourcemanager_test

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	sdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/resource-project.tf
var resourceProject string

//go:embed testdata/resource-folder.tf
var resourceFolder string

var projectNameParentContainerId = fmt.Sprintf("project-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var projectNameParentContainerIdUpdated = fmt.Sprintf("%s-updated", projectNameParentContainerId)

var projectNameParentUUID = fmt.Sprintf("project-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var projectNameParentUUIDUpdated = fmt.Sprintf("%s-updated", projectNameParentUUID)

var folderNameParentContainerId = fmt.Sprintf("folder-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var folderNameParentContainerIdUpdated = fmt.Sprintf("%s-updated", folderNameParentContainerId)

var folderNameParentUUID = fmt.Sprintf("folder-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
var folderNameParentUUIDUpdated = fmt.Sprintf("%s-updated", folderNameParentUUID)

var testConfigResourceProjectParentContainerId = config.Variables{
	"name":                config.StringVariable(projectNameParentContainerId),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
}

var testConfigResourceProjectParentUUID = config.Variables{
	"name":                config.StringVariable(projectNameParentUUID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentUUID),
}

var testConfigResourceFolderParentContainerId = config.Variables{
	"name":                config.StringVariable(folderNameParentContainerId),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentContainerID),
}

var testConfigResourceFolderParentUUID = config.Variables{
	"name":                config.StringVariable(folderNameParentUUID),
	"owner_email":         config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"parent_container_id": config.StringVariable(testutil.TestProjectParentUUID),
}

func testConfigProjectNameParentContainerIdUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigResourceProjectParentContainerId))
	maps.Copy(tempConfig, testConfigResourceProjectParentContainerId)
	tempConfig["name"] = config.StringVariable(projectNameParentContainerIdUpdated)
	return tempConfig
}

func testConfigProjectNameParentUUIDUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigResourceProjectParentUUID))
	maps.Copy(tempConfig, testConfigResourceProjectParentUUID)
	tempConfig["name"] = config.StringVariable(projectNameParentUUIDUpdated)
	return tempConfig
}

func testConfigFolderNameParentContainerIdUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigResourceFolderParentContainerId))
	maps.Copy(tempConfig, testConfigResourceFolderParentContainerId)
	tempConfig["name"] = config.StringVariable(folderNameParentContainerIdUpdated)
	return tempConfig
}

func testConfigFolderNameParentUUIDUpdated() config.Variables {
	tempConfig := make(config.Variables, len(testConfigResourceFolderParentUUID))
	maps.Copy(tempConfig, testConfigResourceFolderParentUUID)
	tempConfig["name"] = config.StringVariable(folderNameParentUUIDUpdated)
	return tempConfig
}

func getImportIdFromID(s *terraform.State, resourceName string, keyName string) (string, error) {
	r, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return "", fmt.Errorf("couldn't find resource %s", resourceName)
	}
	id, ok := r.Primary.Attributes[keyName]
	if !ok {
		return "", fmt.Errorf("couldn't find attribute %s", keyName)
	}
	return id, nil
}

func TestAccResourceManagerProjectContainerId(t *testing.T) {
	resourceName := "stackit_resourcemanager_project.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceManagerDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigResourceProjectParentContainerId,
				Config:          testutil.ResourceManagerProviderConfig() + resourceProject,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["name"])),
					resource.TestCheckResourceAttr(resourceName, "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceProjectParentContainerId["parent_container_id"])),
					resource.TestCheckResourceAttrSet(resourceName, "container_id"),
					resource.TestCheckResourceAttrSet(resourceName, "project_id"),
				),
			},

			// Data Source
			{
				ConfigVariables: testConfigResourceProjectParentContainerId,
				Config: fmt.Sprintf(`
                    %s
                    %s

                    data "stackit_resourcemanager_project" "example" {
                        project_id = stackit_resourcemanager_project.project.project_id
                    }
                `, testutil.ResourceManagerProviderConfig(), resourceProject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("stackit_resourcemanager_project.project", "project_id", "data.stackit_resourcemanager_project.example", "project_id"),
					resource.TestCheckResourceAttrPair("stackit_resourcemanager_project.project", "name", "data.stackit_resourcemanager_project.example", "name"),
				),
			},

			// Import
			{
				ConfigVariables:   testConfigResourceProjectParentContainerId,
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return getImportIdFromID(s, resourceName, "container_id")
				},
				ImportStateVerifyIgnore: []string{"owner_email"},
			},

			// Update
			{
				ConfigVariables: testConfigProjectNameParentContainerIdUpdated(),
				Config:          testutil.ResourceManagerProviderConfig() + resourceProject,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", projectNameParentContainerIdUpdated),
				),
			},
		},
	})
}

func TestAccResourceManagerProjectParentUUID(t *testing.T) {
	resourceName := "stackit_resourcemanager_project.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceManagerDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigResourceProjectParentUUID,
				Config:          testutil.ResourceManagerProviderConfig() + resourceProject,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["name"])),
					resource.TestCheckResourceAttr(resourceName, "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceProjectParentUUID["parent_container_id"])),
					resource.TestCheckResourceAttrSet(resourceName, "container_id"),
					resource.TestCheckResourceAttrSet(resourceName, "project_id"),
				),
			},

			// Data Source
			{
				ConfigVariables: testConfigResourceProjectParentUUID,
				Config: fmt.Sprintf(`
                    %s
                    %s

                    data "stackit_resourcemanager_project" "example" {
                        project_id = stackit_resourcemanager_project.project.project_id
                    }
                `, testutil.ResourceManagerProviderConfig(), resourceProject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("stackit_resourcemanager_project.project", "project_id", "data.stackit_resourcemanager_project.example", "project_id"),
					resource.TestCheckResourceAttrPair("stackit_resourcemanager_project.project", "name", "data.stackit_resourcemanager_project.example", "name"),
				),
			},

			// Import
			{
				ConfigVariables:   testConfigResourceProjectParentUUID,
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return getImportIdFromID(s, resourceName, "container_id")
				},
				ImportStateVerifyIgnore: []string{"owner_email"},
			},

			// Update
			{
				ConfigVariables: testConfigProjectNameParentUUIDUpdated(),
				Config:          testutil.ResourceManagerProviderConfig() + resourceProject,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", projectNameParentUUIDUpdated),
				),
			},
		},
	})
}

func TestAccResourceManagerFolderContainerId(t *testing.T) {
	resourceName := "stackit_resourcemanager_folder.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceManagerDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigResourceFolderParentContainerId,
				Config:          testutil.ResourceManagerProviderConfig() + resourceFolder,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["name"])),
					resource.TestCheckResourceAttr(resourceName, "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceFolderParentContainerId["parent_container_id"])),
					resource.TestCheckResourceAttrSet(resourceName, "container_id"),
				),
			},

			// Data source
			{
				ConfigVariables: testConfigResourceFolderParentContainerId,
				Config: fmt.Sprintf(`
                    %s
                    %s

                    data "stackit_resourcemanager_folder" "example" {
                        container_id = stackit_resourcemanager_folder.folder.container_id
                    }
                `, testutil.ResourceManagerProviderConfig(), resourceFolder),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("stackit_resourcemanager_folder.folder", "container_id", "data.stackit_resourcemanager_folder.example", "container_id"),
					resource.TestCheckResourceAttrPair("stackit_resourcemanager_folder.folder", "name", "data.stackit_resourcemanager_folder.example", "name"),
				),
			},

			// Import
			{
				ConfigVariables:   testConfigResourceFolderParentContainerId,
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return getImportIdFromID(s, resourceName, "container_id")
				},
				ImportStateVerifyIgnore: []string{"owner_email"},
			},

			// Update
			{
				ConfigVariables: testConfigFolderNameParentContainerIdUpdated(),
				Config:          testutil.ResourceManagerProviderConfig() + resourceFolder,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", folderNameParentContainerIdUpdated),
				),
			},
		},
	})
}

func TestAccResourceManagerFolderUUID(t *testing.T) {
	resourceName := "stackit_resourcemanager_folder.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceManagerDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				ConfigVariables: testConfigResourceFolderParentUUID,
				Config:          testutil.ResourceManagerProviderConfig() + resourceFolder,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", testutil.ConvertConfigVariable(testConfigResourceFolderParentUUID["name"])),
					resource.TestCheckResourceAttr(resourceName, "parent_container_id", testutil.ConvertConfigVariable(testConfigResourceFolderParentUUID["parent_container_id"])),
					resource.TestCheckResourceAttrSet(resourceName, "container_id"),
				),
			},

			// Data source
			{
				ConfigVariables: testConfigResourceFolderParentUUID,
				Config: fmt.Sprintf(`
                    %s
                    %s

                    data "stackit_resourcemanager_folder" "" {
                        container_id = stackit_resourcemanager_folder.folder.container_id
                    }
                `, testutil.ResourceManagerProviderConfig(), resourceFolder),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("stackit_resourcemanager_folder.folder", "container_id", "data.stackit_resourcemanager_folder.example", "container_id"),
					resource.TestCheckResourceAttrPair("stackit_resourcemanager_folder.folder", "name", "data.stackit_resourcemanager_folder.example", "name"),
				),
			},

			// Import
			{
				ConfigVariables:   testConfigResourceFolderParentUUID,
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return getImportIdFromID(s, resourceName, "container_id")
				},
				ImportStateVerifyIgnore: []string{"owner_email"},
			},

			// Update
			{
				ConfigVariables: testConfigFolderNameParentUUIDUpdated(),
				Config:          testutil.ResourceManagerProviderConfig() + resourceFolder,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", folderNameParentUUIDUpdated),
				),
			},
		},
	})
}

func testAccCheckResourceManagerDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *resourcemanager.APIClient
	var err error
	if testutil.ResourceManagerCustomEndpoint == "" {
		client, err = resourcemanager.NewAPIClient()
	} else {
		client, err = resourcemanager.NewAPIClient(
			sdkConfig.WithEndpoint(testutil.ResourceManagerCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	projectsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_resourcemanager_project" {
			continue
		}
		// project terraform ID: "[container_id]"
		containerId := rs.Primary.ID
		projectsToDestroy = append(projectsToDestroy, containerId)
	}

	projectsResp, err := client.ListProjects(ctx).ContainerParentId(testutil.TestProjectParentContainerID).Execute()
	if err != nil {
		return fmt.Errorf("getting projectsResp: %w", err)
	}

	items := *projectsResp.Items
	for i := range items {
		if *items[i].LifecycleState == resourcemanager.LIFECYCLESTATE_DELETING {
			continue
		}
		if !utils.Contains(projectsToDestroy, *items[i].ContainerId) {
			continue
		}

		err := client.DeleteProjectExecute(ctx, *items[i].ContainerId)
		if err != nil {
			return fmt.Errorf("destroying project %s during CheckDestroy: %w", *items[i].ContainerId, err)
		}
		_, err = wait.DeleteProjectWaitHandler(ctx, client, *items[i].ContainerId).WaitWithContext(ctx)
		if err != nil {
			return fmt.Errorf("destroying project %s during CheckDestroy: waiting for deletion %w", *items[i].ContainerId, err)
		}
	}
	return nil
}
