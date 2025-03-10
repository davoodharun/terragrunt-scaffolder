include "cosmos_account" {
  path           = "${get_repo_root()}/.infrastructure/architecture/_components/cosmos_account/cosmos_account.hcl"
  expose         = true
  merge_strategy = "deep"
}

include "root" {
  path   = "${get_repo_root()}/.infrastructure/architecture/root.hcl"
  expose = true
}