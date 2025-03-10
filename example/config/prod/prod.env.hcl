locals {
  tier        = "prod"
  tier_prefix = "p"
  resources = {
    eastus    = "XZE-E-P-CUSTTP-P-RGP-10"
    centralus = "XZC-E-P-CUSTTP-P-RGP-10"
  }
  resource_group_name = "XZC-E-P-CUSTTP-P-RGP-10"
  keyvaults = {
    prodkvt = {
      name           = "P-E-KVT-UCDSEC-ALL-10"
      resource_group = "XZE-E-P-UCDSEC-P-RGP-10"
    }
  }
}