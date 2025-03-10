locals {
  type = "api"
  identifier = {
    primary    = "custtp"
    repository = "EU-CustomerTouchPoints"
  }
  tags = {
    ApplicationName = "EU CONVERGED WEBSITES"
    DevOpsCreator   = "Eu-DevOps"
    PaaSOnly        = "YES"
    ProjectName     = "custtp"
    SharedService   = "NO"
    app_id          = "custtp"
  }

  keyvaults = {
    dev = {
      name           = "T-E-KVT-UCDSEC-ALL-10"
      resource_group = "XZE-E-N-UCDSEC-T-RGP-10"
    }
    test = {
      name           = "T-E-KVT-UCDSEC-ALL-10"
      resource_group = "XZE-E-N-UCDSEC-T-RGP-10"
    }
    stage = {
      name           = "S-E-KVT-UCDSEC-ALL-10"
      resource_group = "XZE-E-N-UCDSEC-S-RGP-10"
    }
    prod = {
      name           = "P-E-KVT-UCDSEC-ALL-10"
      resource_group = "XZE-E-P-UCDSEC-P-RGP-10"
    }

  }

  apim_instances = {
    dev = {
      eastus2 = {
        pep = {
          name                = "D-E-APIM-UCDINT-PEP-10"
          resource_group_name = "XZE-E-N-UCDINT-D-RGP-10"
        }
        ace = {
          name                = "D-E-APIM-UCDINT-ACE-10"
          resource_group_name = "XZE-E-N-UCDINT-D-RGP-10"

        }

        dpl = {
          name                = "D-E-APIM-UCDINT-DPL-10"
          resource_group_name = "XZE-E-N-UCDINT-D-RGP-10"
        }
        bge = {
          name                = "D-E-APIM-UCDINT-BGE-10"
          resource_group_name = "XZE-E-N-UCDINT-D-RGP-10"

        }
        com = {
          name                = "D-E-APIM-UCDINT-COM-10"
          resource_group_name = "XZE-E-N-UCDINT-D-RGP-10"

        }
        pec = {
          name                = "D-E-APIM-UCDINT-PEC-10"
          resource_group_name = "XZE-E-N-UCDINT-D-RGP-10"

        }
      }

      centralus = {
        pep = {
          name                = "D-C-APIM-UCDINT-PEP-10"
          resource_group_name = "XZC-E-N-UCDINT-D-RGP-10"
        }
        ace = {
          name                = "D-C-APIM-UCDINT-ACE-10"
          resource_group_name = "XZC-E-N-UCDINT-D-RGP-10"

        }

        dpl = {
          name                = "D-C-APIM-UCDINT-DPL-10"
          resource_group_name = "XZC-E-N-UCDINT-D-RGP-10"
        }
        bge = {
          name                = "D-C-APIM-UCDINT-BGE-10"
          resource_group_name = "XZC-E-N-UCDINT-D-RGP-10"

        }
        com = {
          name                = "D-C-APIM-UCDINT-COM-10"
          resource_group_name = "XZC-E-N-UCDINT-D-RGP-10"

        }
        pec = {
          name                = "D-C-APIM-UCDINT-PEC-10"
          resource_group_name = "XZC-E-N-UCDINT-D-RGP-10"

        }
      }
    }
    test = {
      eastus2 = {
        pep = {
          name                = "T-E-APIM-UCDINT-PEP-10"
          resource_group_name = "XZE-E-N-UCDINT-T-RGP-10"
        }
        ace = {
          name                = "T-E-APIM-UCDINT-ACE-10"
          resource_group_name = "XZE-E-N-UCDINT-T-RGP-10"

        }

        dpl = {
          name                = "T-E-APIM-UCDINT-DPL-10"
          resource_group_name = "XZE-E-N-UCDINT-T-RGP-10"
        }
        bge = {
          name                = "T-E-APIM-UCDINT-BGE-10"
          resource_group_name = "XZE-E-N-UCDINT-T-RGP-10"

        }
        com = {
          name                = "T-E-APIM-UCDINT-COM-10"
          resource_group_name = "XZE-E-N-UCDINT-T-RGP-10"

        }
        pec = {
          name                = "T-E-APIM-UCDINT-PEC-10"
          resource_group_name = "XZE-E-N-UCDINT-T-RGP-10"

        }
      }

      centralus = {
        pep = {
          name                = "T-C-APIM-UCDINT-PEP-10"
          resource_group_name = "XZC-E-N-UCDINT-T-RGP-10"
        }
        ace = {
          name                = "T-C-APIM-UCDINT-ACE-10"
          resource_group_name = "XZC-E-N-UCDINT-T-RGP-10"

        }

        dpl = {
          name                = "T-C-APIM-UCDINT-DPL-10"
          resource_group_name = "XZC-E-N-UCDINT-T-RGP-10"
        }
        bge = {
          name                = "T-C-APIM-UCDINT-BGE-10"
          resource_group_name = "XZC-E-N-UCDINT-T-RGP-10"

        }
        com = {
          name                = "T-C-APIM-UCDINT-COM-10"
          resource_group_name = "XZC-E-N-UCDINT-T-RGP-10"

        }
        pec = {
          name                = "T-C-APIM-UCDINT-PEC-10"
          resource_group_name = "XZC-E-N-UCDINT-T-RGP-10"

        }
      }
    }
    stage = {
      eastus2 = {
        pep = {
          name                = "S-E-APIM-UCDINT-PEP-10"
          resource_group_name = "XZE-E-N-UCDINT-S-RGP-10"
        }
        ace = {
          name                = "S-E-APIM-UCDINT-ACE-10"
          resource_group_name = "XZE-E-N-UCDINT-S-RGP-10"

        }

        dpl = {
          name                = "S-E-APIM-UCDINT-DPL-10"
          resource_group_name = "XZE-E-N-UCDINT-S-RGP-10"
        }
        bge = {
          name                = "S-E-APIM-UCDINT-BGE-10"
          resource_group_name = "XZE-E-N-UCDINT-S-RGP-10"

        }
        com = {
          name                = "S-E-APIM-UCDINT-COM-10"
          resource_group_name = "XZE-E-N-UCDINT-S-RGP-10"

        }
        pec = {
          name                = "S-E-APIM-UCDINT-PEC-10"
          resource_group_name = "XZE-E-N-UCDINT-S-RGP-10"

        }
      }

      centralus = {
        pep = {
          name                = "S-C-APIM-UCDINT-PEP-10"
          resource_group_name = "XZC-E-N-UCDINT-S-RGP-10"
        }
        ace = {
          name                = "S-C-APIM-UCDINT-ACE-10"
          resource_group_name = "XZC-E-N-UCDINT-S-RGP-10"

        }

        dpl = {
          name                = "S-C-APIM-UCDINT-DPL-10"
          resource_group_name = "XZC-E-N-UCDINT-S-RGP-10"
        }
        bge = {
          name                = "S-C-APIM-UCDINT-BGE-10"
          resource_group_name = "XZC-E-N-UCDINT-S-RGP-10"

        }
        com = {
          name                = "S-C-APIM-UCDINT-COM-10"
          resource_group_name = "XZC-E-N-UCDINT-S-RGP-10"

        }
        pec = {
          name                = "S-C-APIM-UCDINT-PEC-10"
          resource_group_name = "XZC-E-N-UCDINT-S-RGP-10"

        }
      }
    }
    prod = {
      eastus2 = {
        pep = {
          name                = "P-E-APIM-UCDINT-PEP-10"
          resource_group_name = "XZE-E-P-UCDINT-P-RGP-10"
        }
        ace = {
          name                = "P-E-APIM-UCDINT-ACE-10"
          resource_group_name = "XZE-E-P-UCDINT-P-RGP-10"

        }

        dpl = {
          name                = "P-E-APIM-UCDINT-DPL-10"
          resource_group_name = "XZE-E-P-UCDINT-P-RGP-10"
        }
        bge = {
          name                = "P-E-APIM-UCDINT-BGE-10"
          resource_group_name = "XZE-E-P-UCDINT-P-RGP-10"

        }
        com = {
          name                = "P-E-APIM-UCDINT-COM-10"
          resource_group_name = "XZE-E-P-UCDINT-P-RGP-10"

        }
        pec = {
          name                = "P-E-APIM-UCDINT-PEC-10"
          resource_group_name = "XZE-E-P-UCDINT-P-RGP-10"

        }
      }

      centralus = {
        pep = {
          name                = "P-C-APIM-UCDINT-PEP-10"
          resource_group_name = "XZC-E-P-UCDINT-P-RGP-10"
        }
        ace = {
          name                = "P-C-APIM-UCDINT-ACE-10"
          resource_group_name = "XZC-E-P-UCDINT-P-RGP-10"

        }

        dpl = {
          name                = "P-C-APIM-UCDINT-DPL-10"
          resource_group_name = "XZC-E-P-UCDINT-P-RGP-10"
        }
        bge = {
          name                = "P-C-APIM-UCDINT-BGE-10"
          resource_group_name = "XZC-E-P-UCDINT-P-RGP-10"

        }
        com = {
          name                = "P-C-APIM-UCDINT-COM-10"
          resource_group_name = "XZC-E-P-UCDINT-P-RGP-10"

        }
        pec = {
          name                = "P-C-APIM-UCDINT-PEC-10"
          resource_group_name = "XZC-E-P-UCDINT-P-RGP-10"

        }
      }
    }
  }

  opcos = {
    ace = {
      full_name  = "atlanticcityelectric"
      short_name = "ace"
      alt_name   = "ace"
    }
    dpl = {
      full_name  = "delmarva"
      short_name = "dpl"
      alt_name   = "dpl"
    }
    pep = {
      full_name  = "pepco"
      short_name = "pep"
      alt_name   = "pepco"
    }
    bge = {
      full_name  = "bge"
      short_name = "bge"
      alt_name   = "bge"
    }
    pec = {
      full_name  = "peco"
      short_name = "pec"
      alt_name   = "peco"
    }
    com = {
      full_name  = "comed"
      short_name = "com"
      alt_name   = "comed"
    }
  }

}
inputs = {
  identifier = local.identifier
}
