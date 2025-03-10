locals {

  // terragrunt configs
  global_vars  = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/global.hcl").locals
  account_vars = read_terragrunt_config(find_in_parent_folders("subscription.hcl")).locals
  region_vars  = read_terragrunt_config(find_in_parent_folders("region.hcl")).locals
  env_vars     = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals
  tier_vars    = read_terragrunt_config("${get_repo_root()}/.infrastructure/config/${local.account_vars.name}/${local.env_vars.name}.env.hcl").locals

  opco_info = basename(get_terragrunt_dir())
  opco_vars = local.global_vars.opcos["${local.opco_info}"]
  opco_abbr = upper(local.opco_vars.short_name)

  keyvault_settings_keys = {
    for k, v in local.tier_vars.keyvaults : k => v.name
  }
  // dynamic vars for app settings tpl files
  settings_keys = merge({

    opco      = upper(local.opco_vars.alt_name)
    full_name = upper(local.opco_vars.full_name)

    region = lower(local.region_vars.prefix)
  }, local.keyvault_settings_keys)


  settings_paths = {
    global = jsondecode(templatefile("${get_repo_root()}/.infrastructure/appsettings/global.appsettings.tpl", local.settings_keys))
    tier   = jsondecode(templatefile("${get_repo_root()}/.infrastructure/appsettings/${local.account_vars.name}/${local.tier_vars.tier}/${local.tier_vars.tier}.appsettings.tpl", local.settings_keys))

    opco = jsondecode(templatefile("${get_repo_root()}/.infrastructure/appsettings/${local.account_vars.name}/${local.tier_vars.tier}/${lower(local.opco_abbr)}.appsettings.tpl", local.settings_keys))

  }

  appsettings = merge(local.settings_paths.global, local.settings_paths.tier, local.settings_paths.opco)


}

inputs = {
  appsettings = local.appsettings
  # domain = local.domain
}