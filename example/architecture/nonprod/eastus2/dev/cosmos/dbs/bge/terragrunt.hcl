include "cosmos_db" {
  path           = "${get_repo_root()}/.infrastructure/architecture/_components/cosmos_db/cosmos_db.hcl"
  expose         = true
  merge_strategy = "deep"
}

include "root" {
  path   = "${get_repo_root()}/.infrastructure/architecture/root.hcl"
  expose = true
}