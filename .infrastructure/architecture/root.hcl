locals {
  identifier = {
    primary = "custtp"
  }
  # _id = "${get_env("TFTG_KEY", "default")}"
  tags = {
    ADOREPO = "EU-CustomerTouchPoints"
    Project = local.account_vars.locals.key
  }
  # global_vars = read_terragrunt_config("${get_repo_root()}/.infrastructure/_components/global.hcl")
  account_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl"))
}

remote_state {
  backend = "azurerm"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite"
  }
  config = {
    resource_group_name  = local.account_vars.locals.resource_group_name
    storage_account_name = local.account_vars.locals.storage_account_name
    container_name       = lower(local.account_vars.locals.container_name)
    key                  = lower("${local.account_vars.locals.key}/${path_relative_to_include()}.tfstate")
  }
}
inputs = {
  tags       = local.tags
  identifier = local.identifier

}
