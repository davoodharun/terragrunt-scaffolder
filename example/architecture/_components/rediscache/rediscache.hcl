terraform {
  source = "git::https://github.com/exeloncorp/terragrunt-modules.git//.//modules/rediscache"

  extra_arguments "common_vars" {
    commands = get_terraform_commands_that_need_vars()

    required_var_files = [

    ]
  }
}

locals {

  global_vars  = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/global.hcl").locals
  account_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl")).locals
  region_vars  = read_terragrunt_config(find_in_parent_folders("region.hcl")).locals
  env_vars     = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals
  tier_vars    = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.account_vars.name}/${local.env_vars.name}.env.hcl").locals




}

inputs = {
  tags                = local.global_vars.tags
  resource_group_name = local.tier_vars.resources["${local.region_vars.location}"]
  sku_capacity        = local.tier_vars.redis.sku_capacity
  sku_family          = local.tier_vars.redis.sku_family
  sku_name            = local.tier_vars.redis.sku_name
}
