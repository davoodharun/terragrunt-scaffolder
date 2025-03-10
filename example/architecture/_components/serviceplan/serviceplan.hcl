terraform {
  source = "git::https://github.com/exeloncorp/terragrunt-modules.git//.//modules/serviceplan"

  extra_arguments "common_vars" {
    commands = get_terraform_commands_that_need_vars()

    required_var_files = [

    ]
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


  opco_info = lower(basename(get_terragrunt_dir()))
  opco_vars = local.global_vars.opcos["${local.opco_info}"]
  opco_abbr = upper(local.opco_vars.short_name)


}

inputs = {
  os_type             = local.tier_vars.serviceplan.os_type
  sku_name            = local.tier_vars.serviceplan.sku_name
  resource_group_name = local.tier_vars.resources["${local.region_vars.location}"]
  tags = {

  }
  identifier = {
    secondary = upper(local.opco_abbr)
  }
}


