include "serviceplan" {
  path           = "${get_repo_root()}/.infrastructure/architecture/_components/serviceplan/serviceplan.hcl"
  expose         = true
  merge_strategy = "deep"
}

include "root" {
  path   = "${get_repo_root()}/.infrastructure/architecture/root.hcl"
  expose = true
}