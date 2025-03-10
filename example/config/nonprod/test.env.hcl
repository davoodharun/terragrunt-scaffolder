locals {
  tier        = "test"
  tier_prefix = "t"
  resources = {
    eastus    = "XZE-E-N-CUSTTP-S-RGP-10"
    centralus = "XZC-E-N-CUSTTP-S-RGP-10"
  }
  keyvaults = {
    kvref = {
      name           = "S-E-KVT-UCDSEC-ALL-10"
      resource_group = "XZE-E-N-UCDSEC-S-RGP-10"
    }
  }
}