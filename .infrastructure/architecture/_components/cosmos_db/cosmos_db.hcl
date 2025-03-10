terraform {
  source = "git::https://github.com/exeloncorp/terragrunt-modules.git//.//modules/storage/cosmos/cosmos_db"

  extra_arguments "common_vars" {
    commands = get_terraform_commands_that_need_vars()

    required_var_files = [

    ]
  }
}



dependency "cosmos_account" {
  config_path = "${get_repo_root()}/.infrastructure/architecture/${local.account_vars.name}/${local.region_vars.location}/${local.tier_vars.tier}/cosmos/account"
}

locals {

  global_vars  = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/global.hcl").locals
  account_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl")).locals
  env_vars     = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals
  tier_vars    = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.account_vars.name}/${local.env_vars.name}.env.hcl").locals

  opco_info = lower(basename(get_terragrunt_dir()))
  opco_vars = local.global_vars.opcos["${local.opco_info}"]
  opco_abbr = upper(local.opco_vars.short_name)

  region_vars = read_terragrunt_config(find_in_parent_folders("region.hcl")).locals

}

inputs = {
  resource_group_name = local.tier_vars.resources["${local.region_vars.location}"]
  identifier = {
    type      = "COSMOS"
    secondary = "DB"
  }
  database_throughput  = local.tier_vars.cosmos.database_throughput
  container_throughput = local.tier_vars.cosmos.container_throughput
  tags                 = local.global_vars.tags
  name                 = format("%s", upper(local.opco_vars.alt_name))
  cosmos_account_name  = dependency.cosmos_account.outputs.name
}