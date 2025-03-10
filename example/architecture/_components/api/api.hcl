terraform {
  source = "git::https://github.com/exeloncorp/terragrunt-modules.git//.//modules/api/appservice-linux"

  extra_arguments "common_vars" {
    commands = get_terraform_commands_that_need_vars()

    required_var_files = [

    ]
  }
}

dependency "serviceplan" {
  config_path = "${get_repo_root()}/.infrastructure/architecture/${local.account_vars.name}/${local.region_vars.location}/${local.tier_vars.tier}/serviceplan/${local.opco}"
  mock_outputs = {
    primary_connection_string = "primary-string"
  }
}

dependency "redis" {
  config_path = "${get_repo_root()}/.infrastructure/architecture/${local.account_vars.name}/eastus2/${local.tier_vars.tier}/rediscache"
  mock_outputs = {
    primary_connection_string = "primary-string"
  }
}

dependency "servicebus" {
  config_path = "${get_repo_root()}/.infrastructure/architecture/${local.account_vars.name}/eastus2/${local.tier_vars.tier}/servicebus"
  mock_outputs = {
    primary_connection_string = "primary-string"
  }
}

dependency "cosmosdb" {
  config_path = "${get_repo_root()}/.infrastructure/architecture/${local.account_vars.name}/eastus2/${local.tier_vars.tier}/cosmos/dbs/${local.opco}"
  mock_outputs = {
    primary_connection_string = "primary-string"
  }
}

locals {

  // local vars from config
  global_vars  = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/global.hcl").locals
  account_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl")).locals
  region_vars  = read_terragrunt_config(find_in_parent_folders("region.hcl")).locals
  env_vars     = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals
  tier_vars    = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.account_vars.name}/${local.env_vars.name}.env.hcl").locals

  // misc


  opco = lower(basename(get_terragrunt_dir()))
  opco_vars = local.global_vars.opcos["${local.opco}"]
  opco_abbr = upper(local.opco_vars.short_name)

  keyvault_settings_keys = {
    for k, v in local.tier_vars.keyvaults : k => v.name
  }

  // dynamic vars for app settings tpl files
  settings_keys = merge({

    opco      = local.opco_vars.alt_name
    full_name = lower(local.opco_vars.full_name)

    region = lower(local.region_vars.prefix)
  }, local.keyvault_settings_keys)

  // dynamic vars for policy tpl files
  
}

inputs = {
  resource_group_name = local.tier_vars.resources["${local.region_vars.location}"]
  identifier = {
    primary   = local.global_vars.identifier.primary
    type      = "API"
    secondary = upper(local.opco_abbr)
  }
  keyvaults       = local.tier_vars.keyvaults
  connection_strings = {

  }
  acr_name              = local.account_vars.acr_name
  acr_rg_name           = local.account_vars.acr_rg_name
  serviceplan_id        = dependency.serviceplan.outputs.id
  keyvaults             = local.tier_vars.keyvaults
}