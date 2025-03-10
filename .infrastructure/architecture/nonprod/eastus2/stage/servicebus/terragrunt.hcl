include "servicebus" {
  path           = "${get_repo_root()}/.infrastructure/architecture/_components/servicebus/servicebus.hcl"
  expose         = true
  merge_strategy = "deep"
}


include "root" {
  path   = "${get_repo_root()}/.infrastructure/architecture/root.hcl"
  expose = true
}