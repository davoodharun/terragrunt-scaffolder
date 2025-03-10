locals {
  tier        = "stage"
  tier_prefix = "s"
  resources = {
    eastus2   = "XZE-E-N-CUSTTP-S-RGP-10"
    centralus = "XZC-E-N-CUSTTP-S-RGP-10"
  }
  keyvaults = {
    kvref = {
      name           = "S-E-KVT-UCDSEC-ALL-10"
      resource_group = "XZE-E-N-UCDSEC-S-RGP-10"
    }
  }
  serviceplan = {
    sku_name = "P2v2"
    os_type  = "Linux"
  }

  storage_account = {
    tier             = "Standard"
    replication_type = "GRS"
  }

  cosmos = {
    database_throughput  = 1000
    container_throughput = 1000
  }
  redis = {
    sku_name     = "Standard"
    sku_capacity = 2
    sku_family   = "C"
  }
  apim_identifier             = "customer-touchpoints"
  apim_path                   = "customer-touchpoints"
  functionapp_apim_identifier = "customer-touchpoints"
  functionapp_apim_path       = "customer-touchpoints"

  servicebus = {
    sku_name = "Standard"
    topics = [
      {
        name                      = "content-model-updated",
        default_message_ttl       = "P14D",
        enable_batched_operations = true,
        enable_partitioning       = false,
        status                    = "Active"
        max_size_in_megabytes     = 1024,
        POC                       = "TBD",
        support_ordering          = true,
        subscriptions = [
          {
            name               = "EuContentMiddlewareACE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareBGE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareCOMED",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareDPL",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePECO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePEPCO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareViktorLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareRoystonLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          }
        ]
      },
      {
        name                      = "content-entry-updated",
        default_message_ttl       = "P14D",
        enable_batched_operations = true,
        enable_partitioning       = false,
        status                    = "Active"
        max_size_in_megabytes     = 1024,
        POC                       = "TBD",
        support_ordering          = true,
        subscriptions = [
          {
            name               = "EuContentMiddlewareACE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareBGE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareCOMED",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareDPL",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePECO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePEPCO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareViktorLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareRoystonLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          }
        ]
      },
      {
        name                      = "content-cache-expired",
        default_message_ttl       = "P14D",
        enable_batched_operations = true,
        enable_partitioning       = false,
        status                    = "Active"
        max_size_in_megabytes     = 1024,
        POC                       = "TBD",
        support_ordering          = true,
        subscriptions = [
          {
            name               = "EuContentMiddlewareACE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareBGE",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareCOMED",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewareDPL",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePECO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "EuContentMiddlewarePEPCO",
            lock_duration      = "PT1M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareViktorLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          },
          {
            name               = "DevMiddlewareRoystonLocal",
            lock_duration      = "PT5M",
            max_delivery_count = 10,
            rules              = []
          }
        ]
      }
    ]
  }
  autoscale_setting = {
    enabled = true
    profiles = [
      {
        name = "Auto created default scale condition"
        capacity = {
          default = 5
          maximum = 20
          minimum = 5
        }
        rules = [
          {
            scale_action = {
              direction = "Increase"
              type      = "ChangeCount"
              value     = "3"
              cooldown  = "PT15M"
            }
            metric_trigger = {
              metric_name      = "CpuPercentage"
              metric_namespace = "microsoft.web/serverfarms"
              operator         = "GreaterThan"
              statistic        = "Average"
              threshold        = 75
              time_aggregation = "Average"
              time_grain       = "PT1M"
              time_window      = "PT15M"
            }
          },
          {
            scale_action = {
              direction = "Increase"
              type      = "ChangeCount"
              value     = "2"
              cooldown  = "PT15M"
            }
            metric_trigger = {
              metric_name      = "MemoryPercentage"
              metric_namespace = "microsoft.web/serverfarms"
              operator         = "GreaterThan"
              statistic        = "Average"
              threshold        = 75
              time_aggregation = "Average"
              time_grain       = "PT1M"
              time_window      = "PT15M"
            }
          },
          {
            scale_action = {
              direction = "Decrease"
              type      = "ChangeCount"
              value     = "1"
              cooldown  = "PT5M"
            }
            metric_trigger = {
              metric_name      = "CpuPercentage"
              metric_namespace = "microsoft.web/serverfarms"
              operator         = "LessThan"
              statistic        = "Average"
              threshold        = 40
              time_aggregation = "Average"
              time_grain       = "PT1M"
              time_window      = "PT10M"
            }
          },
          {
            scale_action = {
              direction = "Decrease"
              type      = "ChangeCount"
              value     = "2"
              cooldown  = "PT5M"
            }
            metric_trigger = {
              metric_name      = "MemoryPercentage"
              metric_namespace = "microsoft.web/serverfarms"
              operator         = "LessThan"
              statistic        = "Average"
              threshold        = 30
              time_aggregation = "Average"
              time_grain       = "PT1M"
              time_window      = "PT10M"
            }
          }
        ]
      }
    ]
  }
}