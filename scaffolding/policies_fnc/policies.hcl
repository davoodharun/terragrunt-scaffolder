locals {

  // local vars from config
  global_vars  = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/global.hcl").locals
  account_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl")).locals
  region_vars  = read_terragrunt_config(find_in_parent_folders("region.hcl")).locals
  env_vars     = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals
  tier_vars    = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.account_vars.name}/${local.env_vars.name}.env.hcl").locals

  // misc
  opco_info = basename(get_terragrunt_dir())
  opco_vars = local.global_vars.opcos["${local.opco_info}"]
  opco_abbr = upper(local.opco_vars.short_name)

  // dynamic vars for app settings tpl files

  // dynamic vars for policy tpl files
  policy_keys = {
    opco      = local.opco_vars.alt_name
    full_name = lower(local.opco_vars.full_name)
  }

  # policy_file = templatefile("${get_repo_root()}/.infrastructure/policies_fnc/${local.account_vars.name}/${local.tier_vars.tier}/policy.xml", local.policy_keys)
}

inputs = {
  # policy_path = local.policy_file
}