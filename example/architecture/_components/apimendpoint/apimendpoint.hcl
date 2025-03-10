terraform {
  source = "git::https://github.com/exeloncorp/terragrunt-modules.git//.//modules/apimendpoint"

  extra_arguments "common_vars" {
    commands = get_terraform_commands_that_need_vars()

    required_var_files = [

    ]
  }
}

dependency "api" {
  config_path = "${get_repo_root()}/.infrastructure/architecture/${local.account_vars.name}/${local.region_vars.location}/${local.tier_vars.tier}/api/${local.opco_abbr}"
  mock_outputs = {
    primary_connection_string = "primary-string"
  }
}

dependency "functionapp" {
  config_path = "${get_repo_root()}/.infrastructure/architecture/${local.account_vars.name}/${local.region_vars.location}/${local.tier_vars.tier}/api/${local.opco_abbr}"
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


  opco      = lower(basename(get_terragrunt_dir()))
  opco_vars = local.global_vars.opcos["${local.opco}"]
  opco_abbr = local.opco_vars.short_name


}

inputs = {
  resource_group_name = local.tier_vars.resources["${local.region_vars.location}"]
  app_name = dependency.api.outputs.name
  identifier = {
    type      = "API"
    secondary = upper(local.opco_abbr)
  }
  tags = {}
  apim_identifier = local.tier_vars.functionapp_apim_identifier
  apim_path       = local.tier_vars.functionapp_apim_path
  apim_instance = {
    name                = local.global_vars.apim_instances["${local.env_vars.name}"]["${local.region_vars.location}"]["${lower(local.opco)}"].name
    resource_group_name = local.global_vars.apim_instances["${local.env_vars.name}"]["${local.region_vars.location}"]["${lower(local.opco)}"].resource_group_name
  }
  operations = {
    GET = {
      operation    = "GET"
      url_template = "/api/*"
    }
    POST = {
      operation    = "POST"
      url_template = "/api/*"
    }
    PUT = {
      operation    = "PUT"
      url_template = "/api/*"
    }
    DELETE = {
      operation    = "DELETE"
      url_template = "/api/*"
    }
    OPTIONS = {
      operation    = "OPTIONS"
      url_template = "/api/*"
    }
    PATCH = {
      operation    = "PATCH"
      url_template = "/api/*"
    }
    HEAD = {
      operation    = "HEAD"
      url_template = "/api/*"
    }
    TRACE = {
      operation    = "TRACE"
      url_template = "/api/*"
    }
    SWAGGERUI = {
      operation    = "GET"
      url_template = "/swaggerui/*"
    }
    SWAGGERUIREDIRECT = {
      operation    = "GET"
      url_template = "/swaggerui"
    }
    SWAGGERDEFINITION = {
      operation    = "GET"
      url_template = "/swagger/*"
    }
  }
}


