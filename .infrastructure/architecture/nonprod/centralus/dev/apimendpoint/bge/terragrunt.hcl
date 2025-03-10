include "apimendpoint" {
  path           = "${get_repo_root()}/.infrastructure/architecture/_components/apimendpoint/apimendpoint.hcl"
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