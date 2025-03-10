include "api" {
  path           = "${get_repo_root()}/.infrastructure/architecture/_components/api/api.hcl"
  expose         = true
  merge_strategy = "deep"
}


include "appsettings" {
  path           = "${get_repo_root()}/.infrastructure/appsettings/appsettings.hcl"
  expose         = true
  merge_strategy = "deep"
}



include "policy" {
  path           = "${get_repo_root()}/.infrastructure/policies_appsvc/policies.hcl"
  expose         = true
  merge_strategy = "deep"
}


include "root" {
  path   = "${get_repo_root()}/.infrastructure/architecture/root.hcl"
  expose = true
}